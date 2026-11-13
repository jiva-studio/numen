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
			return nil, s.getReadError(ctx, method, err)
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
		if method == methodPrompt && s.cancelTurn(finished) {
			return
		}
		s.halt()
	}()
	return func() {
		close(finished)
		<-ended
	}
}

func (s *Session) cancelTurn(finished <-chan struct{}) bool {
	cancel := rpcRequest{JSONRPC: "", ID: nil, Method: methodCancel, Params: map[string]any{keySessionID: s.id}, Result: nil, Error: nil}
	if s.send(cancel) != nil {
		return false
	}
	grace := time.NewTimer(cancelGrace)
	defer grace.Stop()
	select {
	case <-finished:
		return true
	case <-grace.C:
		return false
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
	reply := rpcRequest{
		JSONRPC: "", ID: request.ID, Method: "", Params: nil, Result: nil,
		Error: &rpcError{Code: methodNotFound, Message: "numen serves no " + request.Method},
	}
	if request.Method == "session/request_permission" {
		reply.Error = &rpcError{Code: permissionRefused, Message: "numen grants no permission for this request"}
		if outcome := s.answerPermission(request.Params, handler); outcome != nil {
			reply.Result, reply.Error = outcome, nil
		}
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

func (s *Session) getReadError(ctx context.Context, method string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, io.EOF) {
		return s.getExitError(method)
	}
	return fmt.Errorf("read %s: %w", s.program, err)
}

func (s *Session) getExitError(method string) error {
	cause := fmt.Errorf("%s exited before answering %s", s.program, method)
	return errors.New(writeFailureMessage(cause, s.proc.Stderr()))
}

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
