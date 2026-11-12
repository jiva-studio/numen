package chatapi

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

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
)

// MaxStreamedCalls is the highest index plus one a streamed tool call may have.
const MaxStreamedCalls = 128

// ErrStreamEnded is a stream that ended before the reply did.
var ErrStreamEnded = errors.New("the chat API's stream ended before the reply did")

// streamedReply is a message as far as its pieces have arrived.
type streamedReply struct {
	text     strings.Builder
	calls    []conversation.ToolCall
	finished string
}

type streamedToolPiece struct {
	Index    int                       `json:"index"`
	ID       string                    `json:"id"`
	Type     string                    `json:"type"`
	Function conversation.CallFunction `json:"function"`
}

// Stream asks for the next message of the conversation and passes its text to emit as it arrives.
// A chat API that answers 400 Bad Request or 422 Unprocessable Entity to a streaming request gives
// an error that wraps conversation.ErrStreamRejected.
func (c *Client) Stream(
	ctx context.Context, cfg settings.Settings, turn conversation.Turn, emit func(string) error,
) (reply conversation.Reply, err error) {
	if err := checkKey(cfg); err != nil {
		return conversation.Reply{}, err
	}
	body, err := json.Marshal(newChatRequest(cfg, turn, true))
	if err != nil {
		return conversation.Reply{}, fmt.Errorf("marshal chat request: %w", err)
	}
	resp, err := c.open(ctx, cfg, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return conversation.Reply{}, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		return conversation.Reply{}, readStreamStatusError(resp)
	}

	reader := bufio.NewReader(resp.Body)
	if isWholeReply(reader) {
		return readWholeStreamReply(reader, emit)
	}
	return readStream(reader, emit)
}

func readStreamStatusError(resp *http.Response) error {
	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read chat response: %w", err)
	}
	reqErr := newStatusError(resp, answer)
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("%w: %w", conversation.ErrStreamRejected, reqErr)
	}
	return reqErr
}

func readWholeStreamReply(reader io.Reader, emit func(string) error) (conversation.Reply, error) {
	answer, err := io.ReadAll(reader)
	if err != nil {
		return conversation.Reply{}, fmt.Errorf("read chat response: %w", err)
	}
	reply, err := readReply(answer)
	if err != nil {
		return conversation.Reply{}, err
	}
	if text := reply.Message.Content; text != nil && *text != "" {
		if err := emit(*text); err != nil {
			return conversation.Reply{}, err
		}
	}
	return reply, nil
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

func appendStreamData(data *bytes.Buffer, line []byte) {
	if field, ok := bytes.CutPrefix(line, []byte("data:")); ok {
		if data.Len() > 0 {
			data.WriteByte('\n')
		}
		data.Write(bytes.TrimPrefix(field, []byte(" ")))
	}
}

func (r *streamedReply) finish() (conversation.Reply, error) {
	if r.finished == "" {
		return conversation.Reply{}, ErrStreamEnded
	}
	return conversation.Reply{Message: r.getMessage(), FinishReason: r.finished}, nil
}

func (r *streamedReply) flush(data *bytes.Buffer, emit func(string) error) (bool, error) {
	if data.Len() == 0 {
		return false, nil
	}
	defer data.Reset()
	return r.add(data.Bytes(), emit)
}

func readStream(reader *bufio.Reader, emit func(string) error) (conversation.Reply, error) {
	var reply streamedReply
	var data bytes.Buffer
	for {
		raw, readErr := reader.ReadBytes('\n')
		line := bytes.TrimRight(raw, "\r\n")
		appendStreamData(&data, line)
		if len(line) == 0 || readErr != nil {
			isDone, err := reply.flush(&data, emit)
			if err != nil {
				return conversation.Reply{}, err
			}
			if isDone {
				return reply.finish()
			}
		}
		if errors.Is(readErr, io.EOF) {
			return reply.finish()
		}
		if readErr != nil {
			return conversation.Reply{}, fmt.Errorf("read chat stream: %w", readErr)
		}
	}
}

func (r *streamedReply) mergeToolCalls(pieces []streamedToolPiece) error {
	for _, piece := range pieces {
		if piece.Index < 0 || piece.Index >= MaxStreamedCalls {
			return fmt.Errorf("the chat API streamed a tool call at index %d", piece.Index)
		}
		for len(r.calls) <= piece.Index {
			r.calls = append(r.calls, conversation.ToolCall{
				ID: "", Type: "function", Function: conversation.CallFunction{Name: "", Arguments: ""},
			})
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

func (r *streamedReply) getMessage() conversation.Message {
	reply := conversation.Message{Role: "assistant", Content: nil, ToolCalls: nil, ToolCallID: ""}
	if r.text.Len() > 0 {
		reply.Content = new(r.text.String())
	}
	for _, call := range r.calls {
		if call.ID != "" || call.Function.Name != "" {
			reply.ToolCalls = append(reply.ToolCalls, call)
		}
	}
	return reply
}

func readStreamError(raw json.RawMessage) string {
	var described struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &described); err == nil && described.Message != "" {
		return described.Message
	}
	return string(raw)
}
