package harness

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

// streamChunk is one event of a streamed reply: a piece of the message, or the error that ends it.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

// streamedReply is a message as far as its pieces have arrived.
type streamedReply struct {
	text     strings.Builder
	calls    []toolCall
	finished string
}

// stream asks the model for the next message as send does, with the reply streamed as server-sent
// events. Each piece of its text goes to emit as it arrives, and an error from emit ends the stream
// and is returned as it is. A chat API that answers the stream with one whole JSON reply has its
// text go to emit at once. A 400 or 422 answer is errStreamRejected.
func (c chat) stream(
	ctx context.Context, model string, effort effort, messages []message, tools []functionTool, emit func(string) error,
) (message, string, error) {
	if err := c.checkKey(); err != nil {
		return message{}, "", err
	}
	body, err := json.Marshal(newChatRequest(model, effort, messages, tools, true))
	if err != nil {
		return message{}, "", fmt.Errorf("marshal chat request: %w", err)
	}
	resp, err := c.open(ctx, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return message{}, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		answer, err := io.ReadAll(resp.Body)
		if err != nil {
			return message{}, "", fmt.Errorf("read chat response: %w", err)
		}
		reqErr := newStatusError(resp, answer)
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity {
			return message{}, "", fmt.Errorf("%w: %w", errStreamRejected, reqErr)
		}
		return message{}, "", reqErr
	}

	reader := bufio.NewReader(resp.Body)
	if isWholeReply(reader) {
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

// readStream reads server-sent events until the `[DONE]` event, sending each piece of text to emit.
// A stream that ends before it, with no reason the model stopped, is errStreamEnded.
func readStream(reader *bufio.Reader, emit func(string) error) (message, string, error) {
	var reply streamedReply
	var data bytes.Buffer
	for {
		line, readErr := reader.ReadBytes('\n')
		line = bytes.TrimRight(line, "\r\n")
		if field, ok := bytes.CutPrefix(line, []byte("data:")); ok {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.Write(bytes.TrimPrefix(field, []byte(" ")))
		}
		if len(line) == 0 || readErr != nil {
			if data.Len() > 0 {
				isDone, err := reply.add(data.Bytes(), emit)
				if err != nil {
					return message{}, "", err
				}
				if isDone {
					if reply.finished == "" {
						return message{}, "", errStreamEnded
					}
					return reply.getMessage(), reply.finished, nil
				}
				data.Reset()
			}
		}
		if errors.Is(readErr, io.EOF) {
			if reply.finished == "" {
				return message{}, "", errStreamEnded
			}
			return reply.getMessage(), reply.finished, nil
		}
		if readErr != nil {
			return message{}, "", fmt.Errorf("read chat stream: %w", readErr)
		}
	}
}

// add takes one event's data into the reply, sending its text to emit. It reports whether the
// event is the stream's last.
func (r *streamedReply) add(data []byte, emit func(string) error) (bool, error) {
	if string(data) == "[DONE]" {
		return true, nil
	}
	var chunk streamChunk
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
	for _, piece := range choice.Delta.ToolCalls {
		if piece.Index < 0 || piece.Index >= maxStreamedCalls {
			return false, fmt.Errorf("the chat API streamed a tool call at index %d", piece.Index)
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
