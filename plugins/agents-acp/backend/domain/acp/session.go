package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	protocolVersion = 1
	// methodNotFound is the JSON-RPC error code for a method the client does not serve.
	methodNotFound = -32601
	// permissionRefused is the JSON-RPC error code for a permission request that offers no option to
	// reject.
	permissionRefused = -32000

	methodPrompt  = "session/prompt"
	methodCancel  = "session/cancel"
	methodClose   = "session/close"
	methodSetting = "session/set_config_option"

	// cancelGrace is how long a canceled turn waits for the program to end it before the program is
	// stopped.
	cancelGrace = 2 * time.Second
	// closeLimit is how long the program has to end a session before it is stopped.
	closeLimit = 5 * time.Second

	// JSON object keys and values of the protocol that more than one message uses.
	keyName      = "name"
	keySessionID = "sessionId"
	keyText      = "text"
	keyValue     = "value"
	keyOutcome   = "outcome"
)

var (
	errNoModelChoice = errors.New("the agent offers no choice of model")
	errNoHTTPServers = errors.New("the agent takes no MCP server over HTTP")
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

// Launcher starts the program of an agent in a folder of its own. The program runs until ctx ends or
// the process is closed.
type Launcher interface {
	Launch(ctx context.Context, program string, command []string) (Process, error)
}

// Server is an MCP server over HTTP that the session offers the agent, with the token it is reached with.
type Server struct {
	Name  string
	URL   string
	Token string
}

// Start is what a session is opened with: the command of the program, the MCP servers of the session and
// the _meta of session/new, nil for none.
type Start struct {
	Command []string
	Servers []Server
	Meta    map[string]any
}

// Handler receives the updates of a turn and decides the permission requests of the agent.
type Handler interface {
	HandleUpdate(update Update) error
	// IsAllowed reports whether the agent may call the tool of the name.
	IsAllowed(tool string) bool
}

// Session is one ACP session with an agent program started for it alone. It grants a permission only
// where the handler of the turn allows the tool, and rejects every other request of the agent.
type Session struct {
	proc           Process
	program        string
	sendMu         sync.Mutex
	nextID         int
	id             string
	canClose       bool
	hasHTTPServers bool
	models         []Model
	current        string
	modelOption    string
	thoughtOption  string
	thoughtLevels  []Level
	thoughtCurrent string
	names          map[string]string
	stopOnce       sync.Once
	stopErr        error
	isStopped      atomic.Bool
}

// Open starts the program and opens a session with it. The program outlives ctx, which bounds the
// opening alone; Close ends it.
func Open(ctx context.Context, launcher Launcher, start Start) (*Session, error) {
	if len(start.Command) == 0 {
		return nil, errors.New("no command starts the agent")
	}
	program := getProgramName(start.Command[0])
	proc, err := launcher.Launch(context.WithoutCancel(ctx), program, start.Command)
	if err != nil {
		return nil, err
	}
	s := &Session{
		proc: proc, program: program, sendMu: sync.Mutex{}, nextID: 0, id: "", canClose: false, hasHTTPServers: false,
		models: nil, current: "", modelOption: "",
		thoughtOption: "", thoughtLevels: nil, thoughtCurrent: "",
		names:    map[string]string{},
		stopOnce: sync.Once{}, stopErr: nil, isStopped: atomic.Bool{},
	}
	if err := s.initialize(ctx); err != nil {
		return nil, errors.Join(err, s.stop())
	}
	if len(start.Servers) > 0 && !s.hasHTTPServers {
		return nil, errors.Join(fmt.Errorf("%w: %s", errNoHTTPServers, program), s.stop())
	}
	if err := s.create(ctx, start); err != nil {
		return nil, errors.Join(err, s.stop())
	}
	return s, nil
}

// GetListing returns the models the agent reported for the session, and the levels of the current one.
func (s *Session) GetListing() Listing {
	return Listing{Models: s.models, Current: s.current, Levels: s.thoughtLevels, Level: s.thoughtCurrent}
}

// SetModel makes the agent answer with model. An empty model or the current one changes nothing.
func (s *Session) SetModel(ctx context.Context, model string) error {
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
		_, err = s.call(ctx, methodSetting, map[string]any{keySessionID: s.id, "configId": s.modelOption, keyValue: model}, nil)
	} else {
		_, err = s.call(ctx, "session/set_model", map[string]any{keySessionID: s.id, "modelId": model}, nil)
	}
	if err != nil {
		return err
	}
	s.current = model
	return nil
}

// SetReasoning makes the agent think at level. A level the agent does not report changes nothing.
func (s *Session) SetReasoning(ctx context.Context, level string) error {
	if level == "" || level == s.thoughtCurrent || !containsLevel(s.thoughtLevels, level) {
		return nil
	}
	if _, err := s.call(ctx, methodSetting, map[string]any{keySessionID: s.id, "configId": s.thoughtOption, keyValue: level}, nil); err != nil {
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

// Prompt sends the texts as one turn, passes each update of the turn to handler, and returns the stop
// reason of the program. When ctx ends, the turn is canceled with session/cancel and the program is
// stopped if it does not end the turn within a grace period. The error of a failed HandleUpdate is
// returned as it is; the session is not usable after an error.
func (s *Session) Prompt(ctx context.Context, texts []string, handler Handler) (StopReason, error) {
	blocks := make([]map[string]any, 0, len(texts))
	for _, text := range texts {
		if text != "" {
			blocks = append(blocks, map[string]any{"type": keyText, keyText: text})
		}
	}
	result, err := s.call(ctx, methodPrompt, map[string]any{keySessionID: s.id, "prompt": blocks}, handler)
	if err != nil {
		return "", err
	}
	var reply struct {
		StopReason StopReason `json:"stopReason"`
	}
	if err := json.Unmarshal(result, &reply); err != nil {
		return "", fmt.Errorf("read the end of the turn of %s: %w", s.program, err)
	}
	return reply.StopReason, nil
}

// Close ends the session, then the program. The program has a limited time to end the session, whether
// ctx has ended or not.
func (s *Session) Close(ctx context.Context) error {
	var err error
	if s.canClose && s.id != "" && !s.isStopped.Load() {
		closing, cancel := context.WithTimeout(context.WithoutCancel(ctx), closeLimit)
		_, err = s.call(closing, methodClose, map[string]any{keySessionID: s.id}, nil)
		cancel()
	}
	return errors.Join(err, s.stop())
}

// Stop ends the program without ending the session first; a turn that runs ends with an error.
func (s *Session) Stop() error {
	return s.stop()
}

// stop ends the program once, from whichever goroutine comes first, and returns why it failed to end.
func (s *Session) stop() error {
	s.halt()
	return s.stopErr
}

// halt ends the program once; Close and Stop return the error of its end.
func (s *Session) halt() {
	s.stopOnce.Do(func() {
		s.isStopped.Store(true)
		s.stopErr = s.proc.Close()
	})
}

func getProgramName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
