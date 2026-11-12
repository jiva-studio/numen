package acp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

// rpcRequest is a JSON-RPC 2.0 request or response the client sends.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
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

func (s *Session) handleMessage(msg rpcMessage, targetID, method string) (bool, json.RawMessage, error) {
	if msg.Method != "" && len(msg.ID) > 0 {
		return false, nil, s.answerRequest(msg)
	}
	if msg.Method == "session/update" {
		return false, nil, s.readUpdate(msg.Params)
	}
	if msg.Method == "" && string(msg.ID) == targetID {
		if msg.Error != nil {
			return true, nil, fmt.Errorf("%s answered %s with: %s", s.program, method, msg.Error.Message)
		}
		return true, msg.Result, nil
	}
	return false, nil, nil
}

func (s *Session) call(method string, params any) (json.RawMessage, error) {
	s.nextID++
	id := strconv.Itoa(s.nextID)
	if err := s.send(rpcRequest{JSONRPC: "", ID: json.RawMessage(id), Method: method, Params: params, Result: nil, Error: nil}); err != nil {
		return nil, fmt.Errorf("send %s to %s: %w", method, s.program, err)
	}
	for {
		line, err := s.proc.Receive()
		if errors.Is(err, io.EOF) {
			return nil, s.getExitError(method)
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", s.program, err)
		}
		var message rpcMessage
		if json.Unmarshal(line, &message) != nil {
			continue
		}
		done, res, err := s.handleMessage(message, id, method)
		if err != nil {
			return nil, err
		}
		if done {
			return res, nil
		}
	}
}

func (s *Session) answerRequest(request rpcMessage) error {
	reply := rpcRequest{JSONRPC: "", ID: request.ID, Method: "", Params: nil, Result: nil, Error: nil}
	if request.Method == "session/request_permission" {
		reply.Result = s.answerPermission(request.Params)
	} else {
		reply.Error = &rpcError{Code: methodNotFound, Message: "numen serves no " + request.Method}
	}
	if err := s.send(reply); err != nil {
		return fmt.Errorf("answer %s of %s: %w", request.Method, s.program, err)
	}
	return nil
}

func (s *Session) readUpdate(params json.RawMessage) error {
	if step := s.readToolStep(params); step != nil && s.report != nil {
		return s.report(step)
	}
	text, isChunk := s.readChunk(params)
	if !isChunk {
		return nil
	}
	if s.answer.Len()+len(text) > maxAnswerBytes {
		return errAnswerTooLarge
	}
	s.answer.WriteString(text)
	return nil
}

func (s *Session) readChunk(params json.RawMessage) (string, bool) {
	var update struct {
		SessionID string `json:"sessionId"`
		Update    struct {
			SessionUpdate string `json:"sessionUpdate"`
			Content       struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"update"`
	}
	if json.Unmarshal(params, &update) != nil || update.SessionID != s.id {
		return "", false
	}
	if update.Update.SessionUpdate != "agent_message_chunk" || update.Update.Content.Type != keyText {
		return "", false
	}
	return update.Update.Content.Text, true
}

func (s *Session) send(message rpcRequest) error {
	message.JSONRPC = "2.0"
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return s.proc.Send(raw)
}

func (s *Session) getExitError(method string) error {
	return errors.New(agentrun.FailureMessage(
		fmt.Errorf("%s exited before answering %s", s.program, method), s.proc.Stderr()))
}
