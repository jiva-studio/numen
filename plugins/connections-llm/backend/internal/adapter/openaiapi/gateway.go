// Package openaiapi sends the requests of a connection to a service that implements the OpenAI API:
// chat completions, embeddings and the list of models. It keeps no key and no connection between
// calls, and sends a request only to an address the domain allows.
package openaiapi

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/ports"
)

// Clock gets the current time, to read a Retry-After given as a date.
type Clock interface {
	Now() time.Time
}

// Gateway is the OpenAI API as the use cases see it. A request that gets no answer ends with the end of the
// call where the call has ended, stops for an address the domain refuses, and is unavailable, to try the
// next connection after, for a service that is unreachable or slow.
type Gateway struct {
	clients clients
	clock   Clock
}

var _ ports.Gateway = (*Gateway)(nil)

// New returns a Gateway that reads a date of the services against clock.
func New(clock Clock) *Gateway {
	return &Gateway{clients: newClients(), clock: clock}
}

// Close closes the idle connections.
func (g *Gateway) Close() {
	g.clients.close()
}

type reasoningBody struct {
	Effort string `json:"effort"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatBody struct {
	Model               string         `json:"model"`
	Messages            []chatMessage  `json:"messages"`
	Temperature         *float64       `json:"temperature,omitempty"`
	MaxTokens           int            `json:"max_tokens,omitempty"`
	MaxCompletionTokens int            `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string         `json:"reasoning_effort,omitempty"`
	Reasoning           *reasoningBody `json:"reasoning,omitempty"`
}

type chatAnswer struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete asks the chat completions endpoint of the service for an answer.
func (g *Gateway) Complete(ctx context.Context, call connection.Call, limit time.Duration, request *llm.CompleteRequest) (*llm.CompleteResult, error) {
	messages := make([]chatMessage, len(request.GetMessages()))
	for i, message := range request.GetMessages() {
		messages[i] = chatMessage{Role: message.GetRole(), Content: message.GetText()}
	}
	body := chatBody{
		Model: call.Options[llm.OptionModel], Messages: messages, Temperature: request.Temperature,
		MaxTokens: 0, MaxCompletionTokens: 0, ReasoningEffort: "", Reasoning: nil,
	}
	body.MaxTokens, body.MaxCompletionTokens = splitTokenLimit(call.Type, int(request.GetMaxTokens()))
	body.ReasoningEffort, body.Reasoning = splitReasoning(call.Type, call.Options[llm.OptionReasoning])

	var answer chatAnswer
	if err := g.send(ctx, call, limit, endpoint{http.MethodPost, "/chat/completions"}, body, &answer); err != nil {
		return nil, err
	}
	if len(answer.Choices) == 0 {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	return &llm.CompleteResult{Text: answer.Choices[0].Message.Content}, nil
}

func splitTokenLimit(typeID string, limit int) (maxTokens, maxCompletionTokens int) {
	if typeID == servicetype.OpenAI {
		return 0, limit
	}
	return limit, 0
}

func splitReasoning(typeID, level string) (effort string, object *reasoningBody) {
	if typeID == servicetype.OpenRouter && level != "" {
		return "", &reasoningBody{Effort: level}
	}
	return level, nil
}

type embedBody struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedAnswer struct {
	Data []struct {
		Index     *int      `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed asks the embeddings endpoint of the service for one vector for each text.
func (g *Gateway) Embed(ctx context.Context, call connection.Call, limit time.Duration, texts []string) ([][]float32, error) {
	var answer embedAnswer
	if err := g.send(ctx, call, limit, endpoint{http.MethodPost, "/embeddings"}, embedBody{Model: call.Options[llm.OptionModel], Input: texts}, &answer); err != nil {
		return nil, err
	}
	if len(answer.Data) != len(texts) {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	vectors := make([][]float32, len(texts))
	for position, one := range answer.Data {
		index := position
		if one.Index != nil {
			index = *one.Index
		}
		if index < 0 || index >= len(vectors) || vectors[index] != nil || len(one.Embedding) == 0 {
			return nil, connection.NewStopError(connection.ReasonUnavailable)
		}
		vectors[index] = one.Embedding
	}
	return vectors, nil
}

type modelsAnswer struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Architecture struct {
			OutputModalities []string `json:"output_modalities"`
		} `json:"architecture"`
	} `json:"data"`
}

// ListModels lists the models of the service that fit the operations of a setting of rows. OpenRouter
// labels its models, so the list holds its embedding models for a setting used for llm.embed alone and its
// models of text for a setting used for llm.complete or llm.chat, and none for any other; the other
// services give no label, and every model is listed. A model with no declared output modalities counts as
// a model of text, because OpenRouter leaves the list out for some.
func (g *Gateway) ListModels(ctx context.Context, call connection.Call, limit time.Duration, operations []string) ([]connection.Option, error) {
	isText := slices.Contains(operations, llm.OperationComplete) || slices.Contains(operations, llm.OperationChat)
	isEmbed := slices.Contains(operations, llm.OperationEmbed)
	path, isLabeled := "/models", call.Type == servicetype.OpenRouter
	if isLabeled && isEmbed && !isText {
		path = "/embeddings/models"
	} else if isLabeled && !isText {
		return []connection.Option{}, nil
	}
	var answer modelsAnswer
	if err := g.send(ctx, call, limit, endpoint{http.MethodGet, path}, nil, &answer); err != nil {
		return nil, err
	}
	models := make([]connection.Option, 0, len(answer.Data))
	for _, one := range answer.Data {
		if isLabeled && isText && !makesText(one.Architecture.OutputModalities) {
			continue
		}
		title := one.Name
		if title == "" {
			title = one.ID
		}
		models = append(models, connection.Option{Value: one.ID, Title: title})
	}
	slices.SortFunc(models, func(a, b connection.Option) int { return strings.Compare(a.Value, b.Value) })
	return models, nil
}

func makesText(modalities []string) bool {
	return len(modalities) == 0 || slices.Contains(modalities, "text")
}
