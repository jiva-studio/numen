package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/reasoning"
)

const (
	acpProtocolVersion = 1
	// maxAnswerBytes is the most text an answer may hold.
	maxAnswerBytes = 4 << 20
	// stderrTailBytes is how much of the agent's error output is retained for its failure message.
	stderrTailBytes = 8 << 10
	// methodNotFound is the JSON-RPC error code for a method the client does not serve.
	methodNotFound = -32601
)

var (
	errAnswerTooLarge = errors.New("the agent's answer is too large")
	errNoModelChoice  = errors.New("the agent offers no choice of model")
)

// acpModel is one model an ACP agent reports for its sessions.
type acpModel struct {
	ID   string
	Name string
}

// acpSession is one ACP session with an agent started for it alone. The client offers the agent no
// file system, no terminal and no MCP servers, and refuses every permission it asks for.
type acpSession struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	lines    *bufio.Scanner
	stderr   *tailBuffer
	stop     context.CancelFunc
	cleanDir func()
	dir      string
	program  string
	nextID   int
	id       string
	canClose bool
	// models are what the agent reports for the session, current is the one it answers with, and
	// modelOption is the id of its model config option, empty where it reports models the older way.
	models      []acpModel
	current     string
	modelOption string
	// thoughtOption is the id of the config option of the category thought_level, empty where the agent
	// reports none; thoughtLevels are its values and thoughtCurrent the one the session works at. They
	// hold for the current model.
	thoughtOption  string
	thoughtLevels  []reasoning.Level
	thoughtCurrent string
	// answer is the text of the agent's message chunks.
	answer strings.Builder
}

// openACPSession starts the agent, initializes the connection and opens a session in an empty
// folder of its own.
func openACPSession(ctx context.Context, agent acpAgent) (*acpSession, error) {
	if len(agent.Command) == 0 {
		return nil, fmt.Errorf("no command starts %s", agent.Program)
	}
	dir, cleanDir, err := agentrun.CreateTempDir("numen-acp-*")
	if err != nil {
		return nil, fmt.Errorf("create the session folder: %w", err)
	}
	running, stop := context.WithCancel(ctx)
	s := &acpSession{stop: stop, cleanDir: cleanDir, dir: dir, program: agent.Program, stderr: &tailBuffer{}}
	s.cmd = exec.CommandContext(running, agent.Command[0], agent.Command[1:]...)
	s.cmd.Dir = dir
	s.cmd.Env = os.Environ()
	s.cmd.Stderr = s.stderr
	agentrun.Supervise(s.cmd)
	if err := s.start(); err != nil {
		s.close()
		return nil, err
	}
	if err := s.initialize(); err != nil {
		s.close()
		return nil, err
	}
	if err := s.create(); err != nil {
		s.close()
		return nil, err
	}
	return s, nil
}

func (s *acpSession) start() error {
	stdin, err := s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open the input of %s: %w", s.program, err)
	}
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open the output of %s: %w", s.program, err)
	}
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", s.program, err)
	}
	s.stdin = stdin
	s.lines = bufio.NewScanner(stdout)
	s.lines.Buffer(nil, maxAnswerBytes)
	return nil
}

func (s *acpSession) initialize() error {
	result, err := s.call("initialize", map[string]any{
		"protocolVersion": acpProtocolVersion,
		"clientCapabilities": map[string]any{
			"fs":       map[string]any{"readTextFile": false, "writeTextFile": false},
			"terminal": false,
		},
		"clientInfo": map[string]any{"name": "numen", "title": "Numen", "version": "1.0.0"},
	})
	if err != nil {
		return err
	}
	var reply struct {
		AgentCapabilities struct {
			SessionCapabilities struct {
				Close json.RawMessage `json:"close"`
			} `json:"sessionCapabilities"`
		} `json:"agentCapabilities"`
	}
	if err := json.Unmarshal(result, &reply); err != nil {
		return fmt.Errorf("read the initialize reply of %s: %w", s.program, err)
	}
	closing := string(reply.AgentCapabilities.SessionCapabilities.Close)
	s.canClose = closing != "" && closing != "null" && closing != "false"
	return nil
}

// create opens the session and reads the models the agent reports for it.
func (s *acpSession) create() error {
	result, err := s.call("session/new", map[string]any{"cwd": s.dir, "mcpServers": []any{}})
	if err != nil {
		return err
	}
	var reply struct {
		SessionID     string            `json:"sessionId"`
		ConfigOptions []acpConfigOption `json:"configOptions"`
		Models        *struct {
			AvailableModels []struct {
				ModelID string `json:"modelId"`
				Name    string `json:"name"`
			} `json:"availableModels"`
			CurrentModelID string `json:"currentModelId"`
		} `json:"models"`
	}
	if err := json.Unmarshal(result, &reply); err != nil {
		return fmt.Errorf("read the session of %s: %w", s.program, err)
	}
	if reply.SessionID == "" {
		return fmt.Errorf("%s opened a session with no id", s.program)
	}
	s.id = reply.SessionID
	for _, option := range reply.ConfigOptions {
		if option.Category == "thought_level" && option.Type == "select" {
			s.thoughtOption, s.thoughtCurrent, s.thoughtLevels = option.ID, option.CurrentValue, option.getLevels()
		}
		if option.Category == "model" && option.Type == "select" {
			s.modelOption, s.current, s.models = option.ID, option.CurrentValue, option.getModels()
		}
	}
	if s.modelOption == "" && reply.Models != nil {
		s.current = reply.Models.CurrentModelID
		for _, one := range reply.Models.AvailableModels {
			s.models = append(s.models, acpModel{ID: one.ModelID, Name: one.Name})
		}
	}
	return nil
}

// setModel makes the session answer with the model, one of those the agent reports.
func (s *acpSession) setModel(model string) error {
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
			"sessionId": s.id, "configId": s.modelOption, "value": model,
		})
	} else {
		_, err = s.call("session/set_model", map[string]any{"sessionId": s.id, "modelId": model})
	}
	if err != nil {
		return err
	}
	s.current = model
	return nil
}

// setReasoning makes the session work at the level, one the agent reports for its current model. A level
// the agent does not report is not set.
func (s *acpSession) setReasoning(level string) error {
	if level == "" || level == s.thoughtCurrent || !reasoning.Contains(s.thoughtLevels, level) {
		return nil
	}
	if _, err := s.call("session/set_config_option", map[string]any{
		"sessionId": s.id, "configId": s.thoughtOption, "value": level,
	}); err != nil {
		return err
	}
	s.thoughtCurrent = level
	return nil
}

func (s *acpSession) hasModel(model string) bool {
	for _, one := range s.models {
		if one.ID == model {
			return true
		}
	}
	return false
}

// ask sends one prompt, the system text first where there is one, and returns the text of the
// agent's message chunks once the turn ends.
func (s *acpSession) ask(system, prompt string) (string, error) {
	var blocks []map[string]any
	for _, text := range []string{system, prompt} {
		if text != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": text})
		}
	}
	s.answer.Reset()
	result, err := s.call("session/prompt", map[string]any{"sessionId": s.id, "prompt": blocks})
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
	case "refusal":
		return "", fmt.Errorf("%s refused to answer", s.program)
	case "cancelled":
		return "", fmt.Errorf("%s cancelled the turn", s.program)
	}
	return strings.TrimSpace(s.answer.String()), nil
}

// close ends the session where the agent can, then the agent, and removes the session folder.
func (s *acpSession) close() {
	if s.canClose && s.id != "" {
		_, _ = s.call("session/close", map[string]any{"sessionId": s.id})
	}
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	s.stop()
	if s.cmd.Process != nil {
		_ = s.cmd.Wait()
	}
	if s.cleanDir != nil {
		s.cleanDir()
	}
}

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

// call sends a request and returns its result. What the agent sends meanwhile is handled as it
// comes: session updates are read, and requests of the agent are answered.
func (s *acpSession) call(method string, params any) (json.RawMessage, error) {
	s.nextID++
	id := strconv.Itoa(s.nextID)
	if err := s.write(rpcRequest{ID: json.RawMessage(id), Method: method, Params: params}); err != nil {
		return nil, fmt.Errorf("send %s to %s: %w", method, s.program, err)
	}
	for s.lines.Scan() {
		var message rpcMessage
		if err := json.Unmarshal(s.lines.Bytes(), &message); err != nil {
			continue
		}
		switch {
		case message.Method != "" && len(message.ID) > 0:
			if err := s.answerRequest(message); err != nil {
				return nil, err
			}
		case message.Method == "session/update":
			if err := s.readUpdate(message.Params); err != nil {
				return nil, err
			}
		case message.Method == "" && string(message.ID) == id:
			if message.Error != nil {
				return nil, fmt.Errorf("%s answered %s with: %s", s.program, method, message.Error.Message)
			}
			return message.Result, nil
		}
	}
	if err := s.lines.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", s.program, err)
	}
	return nil, errors.New(agentrun.FailureMessage(
		fmt.Errorf("%s exited before answering %s", s.program, method), s.stderr.String()))
}

// answerRequest refuses every permission the agent asks for and serves none of its other methods.
func (s *acpSession) answerRequest(request rpcMessage) error {
	reply := rpcRequest{ID: request.ID}
	if request.Method == "session/request_permission" {
		reply.Result = map[string]any{"outcome": map[string]any{"outcome": "cancelled"}}
	} else {
		reply.Error = &rpcError{Code: methodNotFound, Message: "numen serves no " + request.Method}
	}
	if err := s.write(reply); err != nil {
		return fmt.Errorf("answer %s of %s: %w", request.Method, s.program, err)
	}
	return nil
}

// readUpdate appends the text of the agent's message chunks for this session.
func (s *acpSession) readUpdate(params json.RawMessage) error {
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

// readChunk returns the text of an agent message chunk for this session. Any other update, and one
// that cannot be read, is no chunk.
func (s *acpSession) readChunk(params json.RawMessage) (string, bool) {
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
	if update.Update.SessionUpdate != "agent_message_chunk" || update.Update.Content.Type != "text" {
		return "", false
	}
	return update.Update.Content.Text, true
}

func (s *acpSession) write(message rpcRequest) error {
	message.JSONRPC = "2.0"
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = s.stdin.Write(append(raw, '\n'))
	return err
}

// acpConfigOption is one session config option. A select option's values are listed alone or in
// groups.
type acpConfigOption struct {
	ID           string           `json:"id"`
	Category     string           `json:"category"`
	Type         string           `json:"type"`
	CurrentValue string           `json:"currentValue"`
	Options      []acpOptionValue `json:"options"`
}

type acpOptionValue struct {
	Value   string           `json:"value"`
	Name    string           `json:"name"`
	Options []acpOptionValue `json:"options"`
}

// getModels returns the values of the option, groups flattened.
func (o acpConfigOption) getModels() []acpModel {
	var models []acpModel
	var walk func([]acpOptionValue)
	walk = func(values []acpOptionValue) {
		for _, one := range values {
			if one.Value != "" {
				models = append(models, acpModel{ID: one.Value, Name: one.Name})
			}
			walk(one.Options)
		}
	}
	walk(o.Options)
	return models
}

// getLevels returns the values of the option as levels of reasoning, in the order the agent lists them.
func (o acpConfigOption) getLevels() []reasoning.Level {
	var levels []reasoning.Level
	for _, one := range o.getModels() {
		title := one.Name
		if title == "" {
			title = one.ID
		}
		levels = append(levels, reasoning.Level{Name: one.ID, Title: title})
	}
	return levels
}

// tailBuffer retains the last bytes written to it.
type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - stderrTailBytes; over > 0 {
		t.buf = t.buf[over:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}
