package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
)

const (
	protocolVersion = 1
	// maxAnswerBytes is the most text an answer may hold.
	maxAnswerBytes = 4 << 20
	// methodNotFound is the JSON-RPC error code for a method the client does not serve.
	methodNotFound = -32601
	// methodSessionClose is the JSON-RPC method that ends a session.
	methodSessionClose = "session/close"

	// JSON object keys and values of the protocol that more than one message uses.
	keyName      = "name"
	keySessionID = "sessionId"
	keyText      = "text"
	keyValue     = "value"
	keyOutcome   = "outcome"
)

var (
	errAnswerTooLarge = errors.New("the agent's answer is too large")
	errNoModelChoice  = errors.New("the agent offers no choice of model")
)

// Process is an agent program the client talks to line by line.
type Process interface {
	Dir() string
	Send(line []byte) error
	// Receive returns the next line, valid until the next call, and io.EOF when the output has ended.
	Receive() ([]byte, error)
	Stderr() string
	Close() error
}

// Launcher starts the program of an agent in a folder of its own.
type Launcher interface {
	Launch(ctx context.Context, program string, command []string) (Process, error)
}

// Session is one ACP session with an agent started for it alone. It grants the permission to call
// one of the tools of the run and rejects every other request of the agent.
type Session struct {
	proc           Process
	program        string
	nextID         int
	id             string
	canClose       bool
	hasHTTPServers bool
	tools          Tools
	models         []Model
	current        string
	modelOption    string
	thoughtOption  string
	thoughtLevels  []reasoning.Level
	thoughtCurrent string
	answer         strings.Builder
	calls          map[string]*trackedCall
	words          map[string]toolcall.Declaration
	report         func(*agentrun.Response) error
}

// Open starts the agent and opens a session with it.
func Open(ctx context.Context, launcher Launcher, program string, command []string, tools Tools) (*Session, error) {
	proc, err := launcher.Launch(ctx, program, command)
	if err != nil {
		return nil, err
	}
	s := &Session{
		proc: proc, program: program, nextID: 0, id: "", canClose: false, hasHTTPServers: false,
		tools:  tools,
		models: nil, current: "", modelOption: "",
		thoughtOption: "", thoughtLevels: nil, thoughtCurrent: "",
		answer: strings.Builder{},
		calls:  map[string]*trackedCall{},
		words:  nil, report: nil,
	}
	if err := s.initialize(); err != nil {
		return nil, errors.Join(err, s.Close())
	}
	if err := s.create(); err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return s, nil
}

// SetReporter makes the session report each tool call of the agent to report, shown by the
// declaration of its tool in words.
func (s *Session) SetReporter(words map[string]toolcall.Declaration, report func(*agentrun.Response) error) {
	s.words, s.report = words, report
}

// GetListing returns the models the agent reported for the session, and the depths of the current one.
func (s *Session) GetListing() Listing {
	return Listing{Models: s.models, Current: s.current, Levels: s.thoughtLevels, Level: s.thoughtCurrent}
}

// SetModel makes the agent answer with model. An empty model or the current one changes nothing.
func (s *Session) SetModel(model string) error {
	if model == "" || model == s.current {
		return nil
	}
	if len(s.models) == 0 {
		return fmt.Errorf("%w: %s", errNoModelChoice, s.program)
	}
	if !s.hasModel(model) {
		return fmt.Errorf("%s offers no model %q", s.program, model)
	}
	var err error
	if s.modelOption != "" {
		_, err = s.call("session/set_config_option", map[string]any{
			keySessionID: s.id, "configId": s.modelOption, keyValue: model,
		})
	} else {
		_, err = s.call("session/set_model", map[string]any{keySessionID: s.id, "modelId": model})
	}
	if err != nil {
		return err
	}
	s.current = model
	return nil
}

// SetReasoning makes the agent think at level. A level the agent does not report changes nothing.
func (s *Session) SetReasoning(level string) error {
	if level == "" || level == s.thoughtCurrent || !reasoning.Contains(s.thoughtLevels, level) {
		return nil
	}
	if _, err := s.call("session/set_config_option", map[string]any{
		keySessionID: s.id, "configId": s.thoughtOption, keyValue: level,
	}); err != nil {
		return err
	}
	s.thoughtCurrent = level
	return nil
}

func (s *Session) hasModel(model string) bool {
	for _, one := range s.models {
		if one.ID == model {
			return true
		}
	}
	return false
}

// Ask sends the system text and the prompt as one turn and returns the answer of the agent.
func (s *Session) Ask(system, prompt string) (string, error) {
	var blocks []map[string]any
	for _, text := range []string{system, prompt} {
		if text != "" {
			blocks = append(blocks, map[string]any{"type": keyText, keyText: text})
		}
	}
	s.answer.Reset()
	result, err := s.call("session/prompt", map[string]any{keySessionID: s.id, "prompt": blocks})
	if err != nil {
		return "", err
	}
	var reply struct {
		StopReason string `json:"stopReason"`
	}
	if err := json.Unmarshal(result, &reply); err != nil {
		return "", fmt.Errorf("read the end of the turn of %s: %w", s.program, err)
	}
	switch reply.StopReason {
	case "rejected":
		return "", fmt.Errorf("%s rejected the prompt", s.program)
	case "canceled":
		return "", fmt.Errorf("%s canceled the turn", s.program)
	}
	return strings.TrimSpace(s.answer.String()), nil
}

// Close ends the session, then the program.
func (s *Session) Close() error {
	var err error
	if s.canClose && s.id != "" {
		_, err = s.call(methodSessionClose, map[string]any{keySessionID: s.id})
	}
	return errors.Join(err, s.proc.Close())
}
