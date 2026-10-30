package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxStreamedCalls is the highest index plus one a streamed tool call may have.
const maxStreamedCalls = 128

var (
	// errStreamRejected is a streaming request the chat API answered 400 Bad Request or 422
	// Unprocessable Entity.
	errStreamRejected = errors.New("the chat API rejected a streaming request")
	errStreamEnded    = errors.New("the chat API's stream ended before the reply did")
)

// streamedReply is a message as far as its pieces have arrived.
type streamedReply struct {
	text     strings.Builder
	calls    []toolCall
	finished string
}

func readStreamStatusError(resp *http.Response) error {
	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read chat response: %w", err)
	}
	reqErr := newStatusError(resp, answer)
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("%w: %w", errStreamRejected, reqErr)
	}
	return reqErr
}

func readWholeStreamReply(reader io.Reader, emit func(string) error) (message, string, error) {
	answer, err := io.ReadAll(reader)
	if err != nil {
		return message{}, "", fmt.Errorf("read chat response: %w", err)
	}
	reply, finished, err := readReply(answer)
	if err != nil {
		return message{}, "", err
	}
	if text := reply.Content; text != nil && *text != "" {
		if err := emit(*text); err != nil {
			return message{}, "", err
		}
	}
	return reply, finished, nil
}

// stream asks the model for the next message as send does, with the reply streamed as server-sent
// events. Each piece of its text goes to emit as it arrives, and an error from emit ends the stream
// and is returned as it is. A chat API that answers the stream with one whole JSON reply has its
// text go to emit at once. A 400 or 422 answer is errStreamRejected.
func (c chat) stream(
	ctx context.Context, model string, effort effort, messages []message, tools []functionTool, emit func(string) error,
) (reply message, reason string, err error) {
	if err := c.checkKey(); err != nil {
		return message{}, "", err
	}
	req := newChatRequest(model, effort, messages, tools, true)
	req.Temperature = c.temperature
	req.MaxTokens = c.maxTokens
	body, err := json.Marshal(req)
	if err != nil {
		return message{}, "", fmt.Errorf("marshal chat request: %w", err)
	}
	resp, err := c.open(ctx, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return message{}, "", err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		return message{}, "", readStreamStatusError(resp)
	}

	reader := bufio.NewReader(resp.Body)
	if isWholeReply(reader) {
		return readWholeStreamReply(reader, emit)
	}
	return readStream(reader, emit)
}

// isWholeReply reports whether the answer is one JSON object, which a stream of events never
// begins with.
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

func appendStreamData(data *bytes.Buffer, line []byte) {
	if field, ok := bytes.CutPrefix(line, []byte("data:")); ok {
		if data.Len() > 0 {
			data.WriteByte('\n')
		}
		data.Write(bytes.TrimPrefix(field, []byte(" ")))
	}
}

func (r *streamedReply) finish() (message, string, error) {
	if r.finished == "" {
		return message{}, "", errStreamEnded
	}
	return r.getMessage(), r.finished, nil
}

func (r *streamedReply) flush(data *bytes.Buffer, emit func(string) error) (bool, error) {
	if data.Len() == 0 {
		return false, nil
	}
	defer data.Reset()
	return r.add(data.Bytes(), emit)
}

// readStream reads server-sent events until the `[DONE]` event, sending each piece of text to emit.
// A stream that ends before it, with no reason the model stopped, is errStreamEnded.
func readStream(reader *bufio.Reader, emit func(string) error) (message, string, error) {
	var reply streamedReply
	var data bytes.Buffer
	for {
		raw, readErr := reader.ReadBytes('\n')
		line := bytes.TrimRight(raw, "\r\n")
		appendStreamData(&data, line)
		if len(line) == 0 || readErr != nil {
			isDone, err := reply.flush(&data, emit)
			if err != nil {
				return message{}, "", err
			}
			if isDone {
				return reply.finish()
			}
		}
		if errors.Is(readErr, io.EOF) {
			return reply.finish()
		}
		if readErr != nil {
			return message{}, "", fmt.Errorf("read chat stream: %w", readErr)
		}
	}
}

type streamedToolPiece struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (r *streamedReply) mergeToolCalls(pieces []streamedToolPiece) error {
	for _, piece := range pieces {
		if piece.Index < 0 || piece.Index >= maxStreamedCalls {
			return fmt.Errorf("the chat API streamed a tool call at index %d", piece.Index)
		}
		for len(r.calls) <= piece.Index {
			r.calls = append(r.calls, toolCall{Type: "function"})
		}
		call := &r.calls[piece.Index]
		if call.ID == "" {
			call.ID = piece.ID
		}
		if call.Function.Name == "" {
			call.Function.Name = piece.Function.Name
		}
		call.Function.Arguments += piece.Function.Arguments
	}
	return nil
}

// add takes one event's data into the reply, sending its text to emit. It reports whether the
// event is the stream's last.
func (r *streamedReply) add(data []byte, emit func(string) error) (bool, error) {
	if string(data) == "[DONE]" {
		return true, nil
	}
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content   string              `json:"content"`
				ToolCalls []streamedToolPiece `json:"tool_calls"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(data, &chunk); err != nil {
		return false, fmt.Errorf("decode chat stream: %w", err)
	}
	if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
		return false, fmt.Errorf("the chat API failed while streaming: %s", readStreamError(chunk.Error))
	}
	if len(chunk.Choices) == 0 {
		return false, nil
	}
	choice := chunk.Choices[0]
	if text := choice.Delta.Content; text != "" {
		r.text.WriteString(text)
		if err := emit(text); err != nil {
			return false, err
		}
	}
	if err := r.mergeToolCalls(choice.Delta.ToolCalls); err != nil {
		return false, err
	}
	if choice.FinishReason != "" {
		r.finished = choice.FinishReason
	}
	return false, nil
}

// getMessage is the message the pieces make: its text, and every call that has a name or an id.
func (r *streamedReply) getMessage() message {
	reply := message{Role: "assistant"}
	if r.text.Len() > 0 {
		reply.Content = newText(r.text.String())
	}
	for _, call := range r.calls {
		if call.ID != "" || call.Function.Name != "" {
			reply.ToolCalls = append(reply.ToolCalls, call)
		}
	}
	return reply
}

// readStreamError is what an error event says: its message, else the event as it came.
func readStreamError(raw json.RawMessage) string {
	var described struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &described); err == nil && described.Message != "" {
		return described.Message
	}
	return string(raw)
}
