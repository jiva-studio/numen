package openaiapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// sendError is a send that failed: the receiver of the frames is gone.
type sendError struct{ err error }

func (e *sendError) Error() string { return e.err.Error() }

// streamError is a failure the service reported inside the stream.
type streamError struct{ err error }

func (e *streamError) Error() string { return e.err.Error() }

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
		Message      *streamedMessage `json:"message"`
		FinishReason string           `json:"finish_reason"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

// maxStreamedCalls is the highest index plus one a streamed tool call may have.
const maxStreamedCalls = 128

type chatReader struct {
	send   func(*llm.ChatFrame) error
	calls  []*llm.ToolCall
	finish string
	reason llm.FinishReason
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

func readChatStream(ctx context.Context, reader *bufio.Reader, wait time.Duration, send func(*llm.ChatFrame) error) (*llm.ChatFinish, error) {
	state := &chatReader{send: send, calls: nil, finish: "", reason: llm.FinishReason_FINISH_REASON_STOP, isDone: false}
	if isWholeReply(reader) {
		answer, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		if err := state.read(answer); err != nil {
			return nil, err
		}
		if !state.isDone {
			return nil, errStreamEnded
		}
		return &llm.ChatFinish{Reason: state.reason}, nil
	}
	lines := readLines(ctx, reader)
	var data bytes.Buffer
	for !state.isDone {
		one, err := awaitLine(ctx, lines, wait)
		if err != nil {
			return nil, err
		}
		if err := state.takeLine(&data, one); err != nil {
			return nil, err
		}
	}
	return &llm.ChatFinish{Reason: state.reason}, nil
}

func awaitLine(ctx context.Context, lines <-chan streamLine, wait time.Duration) (streamLine, error) {
	waiting, stop := context.WithTimeout(ctx, wait)
	defer stop()
	select {
	case one := <-lines:
		return one, nil
	case <-waiting.Done():
		if ctx.Err() != nil {
			return streamLine{}, ctx.Err()
		}
		return streamLine{}, errSilent
	}
}

func (r *chatReader) takeLine(data *bytes.Buffer, one streamLine) error {
	line := bytes.TrimRight(one.raw, "\r\n")
	if field, ok := bytes.CutPrefix(line, []byte("data:")); ok {
		if data.Len() > 0 {
			data.WriteByte('\n')
		}
		data.Write(bytes.TrimPrefix(field, []byte(" ")))
	}
	if len(line) == 0 || one.err != nil {
		if err := r.flush(data); err != nil {
			return err
		}
	}
	if one.err == nil || r.isDone {
		return nil
	}
	if errors.Is(one.err, io.EOF) {
		return errStreamEnded
	}
	return one.err
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
		return &streamError{err: readStreamError(chunk.Error)}
	}
	if len(chunk.Choices) == 0 {
		return nil
	}
	choice := chunk.Choices[0]
	if text := choice.Delta.Content; text != "" {
		if err := r.emit(&llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: text}}); err != nil {
			return err
		}
	}
	if err := r.merge(choice.Delta.ToolCalls); err != nil {
		return err
	}
	if err := r.readMessage(choice.Message); err != nil {
		return err
	}
	if choice.FinishReason != "" {
		r.finish = choice.FinishReason
		return r.end()
	}
	return nil
}

func (r *chatReader) readMessage(message *streamedMessage) error {
	if message == nil {
		return nil
	}
	if message.Content != "" {
		if err := r.emit(&llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: message.Content}}); err != nil {
			return err
		}
	}
	for _, call := range message.ToolCalls {
		r.calls = append(r.calls, &llm.ToolCall{Id: call.ID, Name: call.Function.Name, ArgumentsJson: call.Function.Arguments})
	}
	return nil
}

func (r *chatReader) emit(frame *llm.ChatFrame) error {
	if err := r.send(frame); err != nil {
		return &sendError{err: err}
	}
	return nil
}

func (r *chatReader) merge(pieces []streamPiece) error {
	for _, piece := range pieces {
		if piece.Index < 0 || piece.Index >= maxStreamedCalls {
			return &streamError{err: connection.NewStopError(connection.ReasonUnavailable)}
		}
		for len(r.calls) <= piece.Index {
			r.calls = append(r.calls, &llm.ToolCall{})
		}
		call := r.calls[piece.Index]
		if call.GetId() == "" {
			call.Id = piece.ID
		}
		if call.GetName() == "" {
			call.Name = piece.Function.Name
		}
		call.ArgumentsJson += piece.Function.Arguments
	}
	return nil
}

func (r *chatReader) end() error {
	reason := llm.FinishReason_FINISH_REASON_STOP
	for _, call := range r.calls {
		if call.GetId() == "" && call.GetName() == "" {
			continue
		}
		if strings.TrimSpace(call.GetArgumentsJson()) == "" {
			call.ArgumentsJson = "{}"
		}
		if err := r.emit(&llm.ChatFrame{Event: &llm.ChatFrame_ToolCall{ToolCall: call}}); err != nil {
			return err
		}
		reason = llm.FinishReason_FINISH_REASON_TOOL_CALLS
	}
	switch r.finish {
	case "length":
		reason = llm.FinishReason_FINISH_REASON_LENGTH
	case "tool_calls", "function_call":
		reason = llm.FinishReason_FINISH_REASON_TOOL_CALLS
	}
	r.isDone, r.reason = true, reason
	return nil
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
	return connection.NewStopError(connection.ReasonUnavailable)
}

type streamedMessage struct {
	Content   string `json:"content"`
	ToolCalls []struct {
		ID       string           `json:"id"`
		Function chatToolFunction `json:"function"`
	} `json:"tool_calls"`
}
