package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// rpcRequest is a JSON-RPC 2.0 request, notification or response the client sends.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  any             `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcMessage is a JSON-RPC 2.0 message the agent sends: a request, a notification or a response.
type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// call sends a request and reads the messages of the program until its response. The updates and the
// permission requests that arrive meanwhile go to handler, which may be nil. When ctx ends before the
// response, a prompt is canceled and given a grace period, and any other request stops the program.
func (s *Session) call(ctx context.Context, method string, params any, handler Handler) (json.RawMessage, error) {
	s.nextID++
	id := strconv.Itoa(s.nextID)
	if err := s.send(rpcRequest{JSONRPC: "", ID: json.RawMessage(id), Method: method, Params: params, Result: nil, Error: nil}); err != nil {
		return nil, fmt.Errorf("send %s to %s: %w", method, s.program, err)
	}
	stopWatch := s.watch(ctx, method)
	defer stopWatch()
	for {
		line, err := s.proc.Receive()
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			if errors.Is(err, io.EOF) {
				return nil, s.getExitError(method)
			}
			return nil, fmt.Errorf("read %s: %w", s.program, err)
		}
		var message rpcMessage
		if json.Unmarshal(line, &message) != nil {
			continue
		}
		isDone, result, err := s.handleMessage(message, id, method, handler)
		if err != nil {
			return nil, err
		}
		if isDone {
			return result, nil
		}
	}
}

// watch acts on the end of ctx while a request of method waits for its response, and returns the
// function that ends the watch.
func (s *Session) watch(ctx context.Context, method string) func() {
	finished := make(chan struct{})
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		select {
		case <-finished:
			return
		case <-ctx.Done():
		}
		if method == methodPrompt {
			// A program that cannot be told is stopped after the grace period like one that does not answer.
			_ = s.send(rpcRequest{JSONRPC: "", ID: nil, Method: methodCancel, Params: map[string]any{keySessionID: s.id}, Result: nil, Error: nil})
			grace := time.NewTimer(cancelGrace)
			defer grace.Stop()
			select {
			case <-finished:
				return
			case <-grace.C:
			}
		}
		_ = s.stop()
	}()
	return func() {
		close(finished)
		<-ended
	}
}

func (s *Session) handleMessage(msg rpcMessage, targetID, method string, handler Handler) (bool, json.RawMessage, error) {
	if msg.Method != "" && len(msg.ID) > 0 {
		return false, nil, s.answerRequest(msg, handler)
	}
	if msg.Method == "session/update" {
		return false, nil, s.readUpdate(msg.Params, handler)
	}
	if msg.Method == "" && string(msg.ID) == targetID {
		if msg.Error != nil {
			return true, nil, fmt.Errorf("%s answered %s with: %s", s.program, method, msg.Error.Message)
		}
		return true, msg.Result, nil
	}
	return false, nil, nil
}

func (s *Session) answerRequest(request rpcMessage, handler Handler) error {
	reply := rpcRequest{JSONRPC: "", ID: request.ID, Method: "", Params: nil, Result: nil, Error: nil}
	if request.Method == "session/request_permission" {
		reply.Result = s.answerPermission(request.Params, handler)
	} else {
		reply.Error = &rpcError{Code: methodNotFound, Message: "numen serves no " + request.Method}
	}
	if err := s.send(reply); err != nil {
		return fmt.Errorf("answer %s of %s: %w", request.Method, s.program, err)
	}
	return nil
}

func (s *Session) send(message rpcRequest) error {
	message.JSONRPC = "2.0"
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.proc.Send(raw)
}

func (s *Session) getExitError(method string) error {
	cause := fmt.Errorf("%s exited before answering %s", s.program, method)
	return errors.New(writeFailureMessage(cause, s.proc.Stderr()))
}

// writeFailureMessage returns the line of the error output of a program that says why it failed: the
// first line that starts with "error:", else the last line, else the error.
func writeFailureMessage(err error, output string) string {
	clean := strings.TrimSpace(output)
	if clean == "" {
		return err.Error()
	}
	lines := strings.Split(clean, "\n")
	for _, line := range lines {
		if text := strings.TrimSpace(line); len(text) > len("error:") && strings.EqualFold(text[:len("error:")], "error:") {
			return strings.TrimSpace(text[len("error:"):])
		}
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
