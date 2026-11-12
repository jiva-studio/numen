package openaiapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
)

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type streamMessage struct {
	Role       string         `json:"role"`
	Content    *string        `json:"content,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type toolDefinitionBody struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters,omitempty"`
	} `json:"function"`
}

type streamBody struct {
	Model               string               `json:"model"`
	Messages            []streamMessage      `json:"messages"`
	Tools               []toolDefinitionBody `json:"tools,omitempty"`
	Stream              bool                 `json:"stream"`
	Temperature         *float64             `json:"temperature,omitempty"`
	MaxTokens           int                  `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                  `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string               `json:"reasoning_effort,omitempty"`
	Reasoning           *reasoningBody       `json:"reasoning,omitempty"`
}

// Chat asks the chat completions endpoint of the service for the next turn and sends its events as they
// arrive: text deltas, the complete tool calls, then the finish. The limit of the request is how long
// the service may stay silent; it restarts with every line the service sends.
func (g *Gateway) Chat(ctx context.Context, call connection.Call, limit time.Duration, request connection.ProviderChatRequest, send func(connection.ChatEvent) error) error {
	response, requestCtx, cancel, err := g.openStream(ctx, call, limit, newStreamBody(call, request))
	if err != nil {
		return err
	}
	defer cancel()
	defer closeBody(response.Body)
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return g.mapStatus(response)
	}
	err = readChatStream(requestCtx, bufio.NewReader(response.Body), limit, send)
	if err != nil {
		var sendError *sendError
		var failure *streamError
		switch {
		case errors.As(err, &sendError):
			return sendError.err
		case ctx.Err() != nil:
			return ctx.Err()
		case errors.Is(err, errSilent):
			return connection.NewTryNextError(connection.ReasonTimeout, 0)
		case errors.As(err, &failure):
			return failure.err
		}
		return connection.NewTryNextError(connection.ReasonUnreachable, 0)
	}
	return nil
}

func newStreamMessage(one connection.ChatMessage) streamMessage {
	message := streamMessage{Role: one.Role, Content: nil, ToolCalls: nil, ToolCallID: one.ToolCallID}
	if one.Text != "" || len(one.ToolCalls) == 0 {
		message.Content = &one.Text
	}
	for _, tool := range one.ToolCalls {
		message.ToolCalls = append(message.ToolCalls, chatToolCall{
			ID: tool.ID, Type: "function", Function: chatToolFunction{Name: tool.Name, Arguments: tool.Arguments},
		})
	}
	return message
}

func newStreamBody(call connection.Call, request connection.ProviderChatRequest) streamBody {
	messages := make([]streamMessage, len(request.Messages))
	for i, one := range request.Messages {
		messages[i] = newStreamMessage(one)
	}
	tools := make([]toolDefinitionBody, len(request.Tools))
	for i, one := range request.Tools {
		tools[i].Type = "function"
		tools[i].Function.Name, tools[i].Function.Description, tools[i].Function.Parameters = one.Name, one.Description, one.Parameters
	}
	body := streamBody{
		Model: call.Model, Messages: messages, Tools: tools, Stream: true, Temperature: request.Temperature,
		MaxTokens: 0, MaxCompletionTokens: 0, ReasoningEffort: "", Reasoning: nil,
	}
	body.MaxTokens, body.MaxCompletionTokens = splitTokenLimit(call.Type, request.MaxTokens)
	body.ReasoningEffort, body.Reasoning = splitReasoning(call.Type, call.Reasoning)
	return body
}

func (g *Gateway) openStream(
	ctx context.Context, call connection.Call, limit time.Duration, body streamBody,
) (*http.Response, context.Context, context.CancelFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	if !servicetype.IsKnown(call.Type) {
		return nil, nil, nil, connection.NewStopError(connection.ReasonBadRequest)
	}
	key := call.Secrets[servicetype.SecretAPIKey].Value
	base, err := address.Parse(call.Fields[servicetype.FieldBaseURL], key != "")
	if err != nil {
		return nil, nil, nil, connection.NewStopError(connection.ReasonBadRequest)
	}
	isLoopbackRequired := base.Scheme == address.SchemeHTTP && key != ""
	requestCtx, cancel := context.WithCancel(ctx)
	limited := context.WithValue(requestCtx, loopbackOnlyKey{}, isLoopbackRequired)
	request, err := newRequest(limited, endpoint{http.MethodPost, "/chat/completions"}, base, key, body)
	if err != nil {
		cancel()
		return nil, nil, nil, connection.NewStopError(connection.ReasonUnknown)
	}
	opening, stopOpening := context.WithTimeout(ctx, limit)
	defer stopOpening()
	stopCancelling := context.AfterFunc(opening, cancel)
	response, err := g.clients.get(isLoopbackRequired).Do(request)
	if err != nil {
		stopCancelling()
		cancel()
		if ctx.Err() == nil && opening.Err() != nil {
			return nil, nil, nil, connection.NewTryNextError(connection.ReasonTimeout, 0)
		}
		return nil, nil, nil, classifyTransportError(ctx, limited, err)
	}
	if !stopCancelling() {
		closeBody(response.Body)
		cancel()
		if ctx.Err() != nil {
			return nil, nil, nil, ctx.Err()
		}
		return nil, nil, nil, connection.NewTryNextError(connection.ReasonTimeout, 0)
	}
	return response, requestCtx, cancel, nil
}
