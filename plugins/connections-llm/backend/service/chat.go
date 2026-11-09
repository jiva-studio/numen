package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
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
func (s *Service) Chat(ctx context.Context, call connection.Call, request connection.ProviderChatRequest, send func(connection.ChatEvent) error) error {
	if call.Reasoning != "" && !slices.ContainsFunc(getReasoningLevels(call.Type, call.Model), func(level connection.ReasoningLevel) bool {
		return level.Value == call.Reasoning
	}) {
		return connection.NewStopError(connection.ReasonBadRequest)
	}
	response, requestCtx, cancel, err := s.openStream(ctx, call, newStreamBody(call, request))
	if err != nil {
		return err
	}
	defer cancel()
	defer closeBody(response.Body)
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return s.mapStatus(response)
	}
	err = readChatStream(requestCtx, bufio.NewReader(response.Body), s.getLimit(call, operationChat), send)
	if err != nil {
		var sendError *sendFailure
		var failure *streamFailure
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
	s.markServed(call)
	return nil
}

func newStreamBody(call connection.Call, request connection.ProviderChatRequest) streamBody {
	messages := make([]streamMessage, len(request.Messages))
	for i, one := range request.Messages {
		message := streamMessage{Role: one.Role, ToolCallID: one.ToolCallID}
		if one.Text != "" || len(one.ToolCalls) == 0 {
			message.Content = &one.Text
		}
		for _, tool := range one.ToolCalls {
			message.ToolCalls = append(message.ToolCalls, chatToolCall{
				ID: tool.ID, Type: "function", Function: chatToolFunction{Name: tool.Name, Arguments: tool.Arguments},
			})
		}
		messages[i] = message
	}
	tools := make([]toolDefinitionBody, len(request.Tools))
	for i, one := range request.Tools {
		tools[i].Type = "function"
		tools[i].Function.Name, tools[i].Function.Description, tools[i].Function.Parameters = one.Name, one.Description, one.Parameters
	}
	body := streamBody{Model: call.Model, Messages: messages, Tools: tools, Stream: true, Temperature: request.Temperature}
	if call.Type == typeOpenAI {
		body.MaxCompletionTokens = request.MaxTokens
	} else {
		body.MaxTokens = request.MaxTokens
	}
	if call.Type == typeOpenRouter && call.Reasoning != "" {
		body.Reasoning = &reasoningBody{Effort: call.Reasoning}
	} else {
		body.ReasoningEffort = call.Reasoning
	}
	return body
}

func (s *Service) openStream(
	ctx context.Context, call connection.Call, body streamBody,
) (*http.Response, context.Context, context.CancelFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	if !isKnownType(call.Type) {
		return nil, nil, nil, connection.NewStopError(connection.ReasonBadRequest)
	}
	key := call.Secrets[secretAPIKey].Value
	base, err := parseAddress(call.Fields[fieldBaseURL], key != "")
	if err != nil {
		return nil, nil, nil, connection.NewStopError(connection.ReasonBadRequest)
	}
	isLoopbackRequired := base.Scheme == schemeHTTP && key != ""
	requestCtx, cancel := context.WithCancel(ctx)
	limited := context.WithValue(requestCtx, loopbackOnlyKey{}, isLoopbackRequired)
	request, err := newRequest(limited, endpoint{operationChat, http.MethodPost, "/chat/completions"}, base, key, body)
	if err != nil {
		cancel()
		return nil, nil, nil, connection.NewStopError(connection.ReasonUnknown)
	}
	opening, stopOpening := context.WithTimeout(ctx, s.getLimit(call, operationChat))
	defer stopOpening()
	stopCancelling := context.AfterFunc(opening, cancel)
	response, err := s.clients.get(isLoopbackRequired).Do(request)
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

// sendFailure is a send that failed: the receiver of the events is gone.
type sendFailure struct{ err error }

func (e *sendFailure) Error() string { return e.err.Error() }

// streamFailure is a failure the service reported inside the stream.
type streamFailure struct{ err error }

func (e *streamFailure) Error() string { return e.err.Error() }

var errStreamEnded = errors.New("the stream ended before the turn did")

type streamPiece struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string        `json:"content"`
			ToolCalls []streamPiece `json:"tool_calls"`
		} `json:"delta"`
		Message *struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string           `json:"id"`
				Function chatToolFunction `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

// maxStreamedCalls is the highest index plus one a streamed tool call may have.
const maxStreamedCalls = 128

type chatReader struct {
	send   func(connection.ChatEvent) error
	calls  []connection.ToolCall
	finish string
	isOpen bool
	isDone bool
}

// errSilent is a service that sent nothing for as long as the limit of the request.
var errSilent = errors.New("the service stayed silent")

type streamLine struct {
	raw []byte
	err error
}

func readLines(ctx context.Context, reader *bufio.Reader) <-chan streamLine {
	lines := make(chan streamLine)
	go func() {
		defer close(lines)
		for {
			raw, err := reader.ReadBytes('\n')
			select {
			case lines <- streamLine{raw: raw, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return lines
}

func readChatStream(ctx context.Context, reader *bufio.Reader, wait time.Duration, send func(connection.ChatEvent) error) error {
	state := &chatReader{send: send}
	if isWholeReply(reader) {
		answer, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		return state.read(answer)
	}
	lines := readLines(ctx, reader)
	var data bytes.Buffer
	for !state.isDone {
		waiting, stop := context.WithTimeout(ctx, wait)
		var one streamLine
		select {
		case one = <-lines:
			stop()
		case <-waiting.Done():
			stop()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errSilent
		}
		line := bytes.TrimRight(one.raw, "\r\n")
		if field, ok := bytes.CutPrefix(line, []byte("data:")); ok {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.Write(bytes.TrimPrefix(field, []byte(" ")))
		}
		if len(line) == 0 || one.err != nil {
			if err := state.flush(&data); err != nil {
				return err
			}
		}
		if one.err != nil && !state.isDone {
			if errors.Is(one.err, io.EOF) {
				return errStreamEnded
			}
			return one.err
		}
	}
	return nil
}

func isWholeReply(reader *bufio.Reader) bool {
	for n := 1; ; n++ {
		peeked, err := reader.Peek(n)
		if err != nil {
			return false
		}
		switch peeked[n-1] {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
}

func (r *chatReader) flush(data *bytes.Buffer) error {
	if data.Len() == 0 {
		return nil
	}
	defer data.Reset()
	if data.String() == "[DONE]" {
		if r.finish == "" {
			return errStreamEnded
		}
		return nil
	}
	return r.read(data.Bytes())
}

func (r *chatReader) read(data []byte) error {
	var chunk streamChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return err
	}
	if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
		return &streamFailure{err: readStreamError(chunk.Error)}
	}
	if len(chunk.Choices) == 0 {
		return nil
	}
	choice := chunk.Choices[0]
	if text := choice.Delta.Content; text != "" {
		if err := r.emit(connection.ChatEvent{TextDelta: text}); err != nil {
			return err
		}
	}
	if err := r.merge(choice.Delta.ToolCalls); err != nil {
		return err
	}
	if message := choice.Message; message != nil {
		if message.Content != "" {
			if err := r.emit(connection.ChatEvent{TextDelta: message.Content}); err != nil {
				return err
			}
		}
		for _, call := range message.ToolCalls {
			r.calls = append(r.calls, connection.ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments})
		}
	}
	if choice.FinishReason != "" {
		r.finish = choice.FinishReason
		return r.end()
	}
	return nil
}

func (r *chatReader) emit(event connection.ChatEvent) error {
	r.isOpen = true
	if err := r.send(event); err != nil {
		return &sendFailure{err: err}
	}
	return nil
}

func (r *chatReader) merge(pieces []streamPiece) error {
	for _, piece := range pieces {
		if piece.Index < 0 || piece.Index >= maxStreamedCalls {
			return &streamFailure{err: connection.NewStopError(connection.ReasonUnknown)}
		}
		for len(r.calls) <= piece.Index {
			r.calls = append(r.calls, connection.ToolCall{})
		}
		call := &r.calls[piece.Index]
		if call.ID == "" {
			call.ID = piece.ID
		}
		if call.Name == "" {
			call.Name = piece.Function.Name
		}
		call.Arguments += piece.Function.Arguments
	}
	return nil
}

func (r *chatReader) end() error {
	reason := connection.FinishStop
	for _, call := range r.calls {
		if call.ID == "" && call.Name == "" {
			continue
		}
		if strings.TrimSpace(call.Arguments) == "" {
			call.Arguments = "{}"
		}
		if err := r.emit(connection.ChatEvent{ToolCall: &call}); err != nil {
			return err
		}
		reason = connection.FinishToolCalls
	}
	switch r.finish {
	case "length":
		reason = connection.FinishLength
	case "tool_calls", "function_call":
		reason = connection.FinishToolCalls
	}
	r.isDone = true
	return r.emit(connection.ChatEvent{Finish: &connection.Finish{Reason: reason}})
}

func readStreamError(raw json.RawMessage) error {
	var described struct {
		Code json.RawMessage `json:"code"`
	}
	if err := json.Unmarshal(raw, &described); err == nil {
		if status, err := strconv.Atoi(strings.Trim(string(described.Code), `"`)); err == nil {
			return mapStatusCode(status, "", time.Time{})
		}
	}
	return connection.NewStopError(connection.ReasonUnknown)
}
