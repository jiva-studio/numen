// Package acp reaches external coding agents communicating over JSON streams.
package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Endpoint is where the agent reaches this vault's MCP tools.
type Endpoint struct {
	URL   string
	Token string
}

// Agent reaches an external agent CLI using standard Agent Client Protocol (ACP).
type Agent struct {
	// Program names the agent identifier ("antigravity", "codex", etc.)
	Program string
	// Command starts it. Empty uses discovered binary from path or known places.
	Command []string
	// Root is the folder the agent is started in.
	Root string
	// Tools is where it reaches this vault over MCP.
	Tools Endpoint
	// Model is which model answers.
	Model string
	// Turns is max step count.
	Turns int
	// ErrorHandler is notified of background execution errors.
	ErrorHandler func(error)

	carried struct {
		sync.Mutex
		sessions map[string]string
	}

	taken struct {
		sync.Mutex
		running map[*work]bool
		isShut  bool
	}
}

type work struct {
	cmd          *exec.Cmd
	stop         context.CancelFunc
	conversation string
	steps        chan port.Step
	reader       sync.WaitGroup
	once         sync.Once
	cleanup      func()
}

func (w *work) Steps() <-chan port.Step { return w.steps }

func (w *work) Stop() error {
	w.once.Do(w.stop)
	w.reader.Wait()
	return nil
}

func (a *Agent) hold(w *work) bool {
	a.taken.Lock()
	defer a.taken.Unlock()

	if a.taken.isShut {
		return false
	}
	if a.taken.running == nil {
		a.taken.running = map[*work]bool{}
	}
	a.taken.running[w] = true
	return true
}

func (a *Agent) letGo(w *work) {
	a.taken.Lock()
	defer a.taken.Unlock()
	delete(a.taken.running, w)
}

// Close stops every running agent process.
func (a *Agent) Close() error {
	a.taken.Lock()
	a.taken.isShut = true
	running := make([]*work, 0, len(a.taken.running))
	for w := range a.taken.running {
		running = append(running, w)
	}
	a.taken.running = nil
	a.taken.Unlock()

	for _, w := range running {
		w.once.Do(w.stop)
	}
	for _, w := range running {
		w.reader.Wait()
	}
	return nil
}

// Finish ends a conversation session.
func (a *Agent) Finish(_ context.Context, conversation string) error {
	if conversation == "" {
		return nil
	}

	a.taken.Lock()
	answering := make([]*work, 0, len(a.taken.running))
	for w := range a.taken.running {
		if w.conversation == conversation {
			answering = append(answering, w)
		}
	}
	a.taken.Unlock()

	for _, w := range answering {
		w.once.Do(w.stop)
	}
	for _, w := range answering {
		w.reader.Wait()
	}

	a.carried.Lock()
	defer a.carried.Unlock()
	delete(a.carried.sessions, conversation)
	return nil
}

func (a *Agent) resolveCommand() (string, []string) {
	if len(a.Command) > 0 {
		return a.Command[0], a.Command[1:]
	}
	if a.Program == "codex" {
		return FindCodex(), nil
	}
	return FindAntigravity(), nil
}

func setupAntigravityHome(tools Endpoint) (string, func(), error) {
	tempHome, err := os.MkdirTemp("", "numen-antigravity-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tempHome) }

	realHome, err := os.UserHomeDir()
	if err != nil {
		cleanup()
		return "", func() {}, err
	}

	if entries, err := os.ReadDir(realHome); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if name != ".gemini" {
				_ = os.Symlink(filepath.Join(realHome, name), filepath.Join(tempHome, name))
			}
		}
	}

	realGemini := filepath.Join(realHome, ".gemini")
	tempGemini := filepath.Join(tempHome, ".gemini")
	_ = os.MkdirAll(filepath.Join(tempGemini, "config"), 0o700)

	if entries, err := os.ReadDir(realGemini); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if name != "config" {
				_ = os.Symlink(filepath.Join(realGemini, name), filepath.Join(tempGemini, name))
			}
		}
	}

	mcpConfig := map[string]any{
		"mcpServers": map[string]any{
			"numen": map[string]any{
				"type":      "http",
				"serverUrl": tools.URL,
				"headers": map[string]string{
					"Authorization": "Bearer " + tools.Token,
				},
			},
		},
	}

	realMCPPath := filepath.Join(realGemini, "config", "mcp_config.json")
	if raw, err := os.ReadFile(realMCPPath); err == nil {
		var existing struct {
			MCPServers map[string]any `json:"mcpServers"`
		}
		if err := json.Unmarshal(raw, &existing); err == nil && existing.MCPServers != nil {
			if servers, ok := mcpConfig["mcpServers"].(map[string]any); ok {
				for k, v := range existing.MCPServers {
					if k != "numen" {
						servers[k] = v
					}
				}
			}
		}
	}

	if encoded, err := json.MarshalIndent(mcpConfig, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(tempGemini, "config", "mcp_config.json"), append(encoded, '\n'), 0o600)
	}

	return tempHome, cleanup, nil
}

// Take starts the agent on a task.
func (a *Agent) Take(ctx context.Context, task port.Task) (port.Run, error) {
	if a.Tools.URL == "" || a.Tools.Token == "" {
		return nil, errors.New("no tools to give an agent")
	}

	running, stop := context.WithCancel(ctx)
	binName, baseArgs := a.resolveCommand()

	var cmd *exec.Cmd
	isAntigravity := a.Program == "antigravity" || strings.HasSuffix(binName, "agy") || strings.HasSuffix(binName, "antigravity")
	isCodex := a.Program == "codex" || strings.HasSuffix(binName, "codex")

	var cleanupTempHome func()

	switch {
	case isAntigravity:
		args := append([]string{}, baseArgs...)
		args = append(args,
			"-p", task.Question,
			"--output-format", "stream-json",
			"--dangerously-skip-permissions",
		)
		model := a.Model
		if task.Model != "" {
			model = task.Model
		}
		if model != "" && model != "default" {
			args = append(args, "--model", model)
		}
		a.carried.Lock()
		if a.carried.sessions != nil && task.Conversation != "" {
			if sessID, ok := a.carried.sessions[task.Conversation]; ok && sessID != "" {
				args = append(args, "--conversation", sessID)
			}
		}
		a.carried.Unlock()

		cmd = exec.CommandContext(running, binName, args...)
		tempHome, clean, err := setupAntigravityHome(a.Tools)
		if err == nil {
			cleanupTempHome = clean
			cmd.Env = append(os.Environ(), "HOME="+tempHome)
		}
	case isCodex:
		args := append([]string{}, baseArgs...)
		mcpURLConfig := fmt.Sprintf(`mcp_servers.numen.url=%q`, a.Tools.URL)
		mcpHeaderConfig := fmt.Sprintf(`mcp_servers.numen.http_headers={"Authorization"="Bearer %s"}`, a.Tools.Token)
		args = append(args,
			"exec",
			"--json",
			"--dangerously-bypass-approvals-and-sandbox",
			"-c", mcpURLConfig,
			"-c", mcpHeaderConfig,
		)
		model := a.Model
		if task.Model != "" {
			model = task.Model
		}
		if model != "" && model != "default" {
			args = append(args, "--model", model)
		}
		args = append(args, task.Question)

		cmd = exec.CommandContext(running, binName, args...)
	default:
		cmd = exec.CommandContext(running, binName, baseArgs...)
	}

	cmd.Dir = a.Root
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "NUMEN_TOKEN="+a.Tools.Token)
	detach(cmd)

	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = 2 * time.Second

	var stdin io.WriteCloser
	var err error
	if !isAntigravity && !isCodex {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			if cleanupTempHome != nil {
				cleanupTempHome()
			}
			stop()
			return nil, err
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if stdin != nil {
			stdin.Close()
		}
		if cleanupTempHome != nil {
			cleanupTempHome()
		}
		stop()
		return nil, err
	}

	w := &work{
		cmd:          cmd,
		stop:         stop,
		conversation: task.Conversation,
		steps:        make(chan port.Step, 64),
		cleanup:      cleanupTempHome,
	}

	if !a.hold(w) {
		if stdin != nil {
			stdin.Close()
		}
		stdout.Close()
		if cleanupTempHome != nil {
			cleanupTempHome()
		}
		stop()
		return nil, errors.New("this agent is closing")
	}

	if err := cmd.Start(); err != nil {
		if stdin != nil {
			stdin.Close()
		}
		stdout.Close()
		a.letGo(w)
		if cleanupTempHome != nil {
			cleanupTempHome()
		}
		stop()
		return nil, fmt.Errorf("start %s: %w", binName, err)
	}

	w.reader.Add(1)
	go func() {
		defer func() {
			if w.cleanup != nil {
				w.cleanup()
			}
		}()
		defer w.reader.Done()
		defer a.letGo(w)
		defer close(w.steps)
		if stdin != nil {
			defer stdin.Close()
		}
		defer stdout.Close()

		if !isAntigravity && !isCodex && stdin != nil {
			initReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "initialize",
				"params": map[string]any{
					"protocolVersion": "2025-08-01",
					"clientInfo":      map[string]any{"name": "Numen", "version": "1.0.0"},
					"capabilities":    map[string]any{"mcp": true, "sessionPersistence": true},
				},
			}
			if raw, err := json.Marshal(initReq); err == nil {
				raw = append(raw, '\n')
				_, _ = stdin.Write(raw)
			}

			sessionReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      2,
				"method":  "session/new",
				"params": map[string]any{
					"cwd": a.Root,
					"mcpServers": []map[string]any{
						{
							"name": "numen",
							"transport": map[string]any{
								"type": "http",
								"url":  a.Tools.URL,
								"headers": map[string]string{
									"Authorization": "Bearer " + a.Tools.Token,
								},
							},
						},
					},
				},
			}
			if raw, err := json.Marshal(sessionReq); err == nil {
				raw = append(raw, '\n')
				_, _ = stdin.Write(raw)
			}

			promptReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      3,
				"method":  "session/prompt",
				"params": map[string]any{
					"model": a.Model,
					"prompt": []map[string]any{
						{"role": "user", "text": task.Question},
					},
					"context": map[string]any{
						"focus": task.Focus,
					},
				},
			}
			if raw, err := json.Marshal(promptReq); err == nil {
				raw = append(raw, '\n')
				_, _ = stdin.Write(raw)
			}
		}

		currentTool := ""
		var currentArgs strings.Builder

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var rawMap map[string]any
			if err := json.Unmarshal(line, &rawMap); err == nil {
				// Handle Claude-style stream_event format
				if evType, ok := rawMap["type"].(string); ok && evType == "stream_event" {
					if evObj, ok := rawMap["event"].(map[string]any); ok {
						subType, _ := evObj["type"].(string)
						switch subType {
						case "content_block_start":
							if blk, ok := evObj["content_block"].(map[string]any); ok {
								blkType, _ := blk["type"].(string)
								if blkType == "tool_use" {
									currentTool, _ = blk["name"].(string)
									currentArgs.Reset()
									w.steps <- port.Step{Kind: port.StepToolCall, Tool: currentTool}
								}
							}
						case "content_block_delta":
							if delta, ok := evObj["delta"].(map[string]any); ok {
								deltaType, _ := delta["type"].(string)
								switch deltaType {
								case "text_delta":
									if t, _ := delta["text"].(string); t != "" {
										w.steps <- port.Step{Kind: port.StepSaying, Text: t}
									}
								case "thinking_delta":
									w.steps <- port.Step{Kind: port.StepThinking}
								case "input_json_delta":
									if partial, _ := delta["partial_json"].(string); partial != "" {
										currentArgs.WriteString(partial)
										w.steps <- port.Step{Kind: port.StepToolCall, Tool: currentTool, About: currentArgs.String()}
									}
								}
							}
						case "content_block_stop":
							if currentTool != "" {
								w.steps <- port.Step{Kind: port.StepToolCall, Tool: currentTool, About: currentArgs.String()}
								currentTool = ""
								currentArgs.Reset()
							}
						}
					}
					continue
				}

				// Session ID tracking
				if sessID, ok := rawMap["conversation_id"].(string); ok && sessID != "" && task.Conversation != "" {
					a.carried.Lock()
					if a.carried.sessions == nil {
						a.carried.sessions = map[string]string{}
					}
					a.carried.sessions[task.Conversation] = sessID
					a.carried.Unlock()
				}

				if sessID, ok := rawMap["session_id"].(string); ok && sessID != "" && task.Conversation != "" {
					a.carried.Lock()
					if a.carried.sessions == nil {
						a.carried.sessions = map[string]string{}
					}
					a.carried.sessions[task.Conversation] = sessID
					a.carried.Unlock()
				}

				// Thinking
				if thinking, ok := rawMap["thinking"].(string); ok && thinking != "" {
					w.steps <- port.Step{Kind: port.StepThinking}
				}

				// Codex events (item.started, item.completed, item.updated)
				if evType, ok := rawMap["type"].(string); ok {
					switch evType {
					case "turn.started":
						w.steps <- port.Step{Kind: port.StepThinking}
					case "item.started":
						if item, ok := rawMap["item"].(map[string]any); ok {
							iType, _ := item["type"].(string)
							switch iType {
							case "mcp_tool_call":
								toolName, _ := item["tool"].(string)
								if toolName == "" {
									toolName, _ = item["name"].(string)
								}
								about := formatToolArgs(item["arguments"])
								w.steps <- port.Step{Kind: port.StepToolCall, Tool: toolName, About: about}
							case "command_execution":
								cmdStr, _ := item["command"].(string)
								w.steps <- port.Step{Kind: port.StepToolCall, Tool: "bash", About: cmdStr}
							case "file_read":
								pathStr, _ := item["path"].(string)
								w.steps <- port.Step{Kind: port.StepToolCall, Tool: "read", About: pathStr}
							}
						}
					case "item.updated":
						if item, ok := rawMap["item"].(map[string]any); ok {
							iType, _ := item["type"].(string)
							if iType == "agent_message" {
								if delta, ok := item["delta"].(string); ok && delta != "" {
									w.steps <- port.Step{Kind: port.StepSaying, Text: delta}
								}
							}
						}
					case "item.completed":
						if item, ok := rawMap["item"].(map[string]any); ok {
							iType, _ := item["type"].(string)
							switch iType {
							case "agent_message":
								if text, ok := item["text"].(string); ok && text != "" {
									w.steps <- port.Step{Kind: port.StepSaying, Text: text}
								}
							case "mcp_tool_call", "command_execution", "file_read":
								w.steps <- port.Step{Kind: port.StepAnswered}
							}
						}
					case "error", "turn.failed":
						if errObj, ok := rawMap["error"].(map[string]any); ok {
							msg, _ := errObj["message"].(string)
							w.steps <- port.Step{Kind: port.StepStopped, Detail: msg}
						} else if msg, ok := rawMap["message"].(string); ok && msg != "" {
							w.steps <- port.Step{Kind: port.StepStopped, Detail: msg}
						}
					case "GENERIC", "user":
						w.steps <- port.Step{Kind: port.StepAnswered}
					}
				}

				// Tool Calls (array or single)
				if toolCalls, ok := rawMap["tool_calls"].([]any); ok {
					for _, tc := range toolCalls {
						if tcMap, ok := tc.(map[string]any); ok {
							toolName, _ := tcMap["name"].(string)
							if toolName == "" {
								toolName, _ = tcMap["tool"].(string)
							}
							about := formatToolArgs(tcMap["args"])
							if toolName != "" {
								w.steps <- port.Step{Kind: port.StepToolCall, Tool: toolName, About: about}
							}
						}
					}
				}

				// Content / Message / Text Delta
				if content, ok := rawMap["content"].(string); ok && content != "" {
					w.steps <- port.Step{Kind: port.StepSaying, Text: content}
				}

				// Antigravity Step update wrapper
				if suMap, ok := rawMap["step_update"].(map[string]any); ok {
					stepType, _ := suMap["step_type"].(string)
					switch stepType {
					case "agent_response":
						if delta, ok := suMap["text_delta"].(string); ok && delta != "" {
							w.steps <- port.Step{Kind: port.StepSaying, Text: delta}
						}
					case "thinking":
						w.steps <- port.Step{Kind: port.StepThinking}
					case "tool_call":
						toolName, _ := suMap["tool_name"].(string)
						argsVal := suMap["arguments"]
						if toolName == "call_mcp_tool" {
							if mArgs, ok := argsVal.(map[string]any); ok {
								if tName, ok := mArgs["ToolName"].(string); ok && tName != "" {
									toolName = tName
									argsVal = mArgs["Arguments"]
								}
							}
						}
						about := formatToolArgs(argsVal)
						w.steps <- port.Step{Kind: port.StepToolCall, Tool: toolName, About: about}
					case "tool_result":
						w.steps <- port.Step{Kind: port.StepAnswered}
					}
				}

				// ACP JSON-RPC format
				if method, _ := rawMap["method"].(string); method == "session/update" {
					if params, ok := rawMap["params"].(map[string]any); ok {
						if update, ok := params["update"].(map[string]any); ok {
							uType, _ := update["type"].(string)
							switch uType {
							case "agent_message":
								if text, _ := update["text"].(string); text != "" {
									w.steps <- port.Step{Kind: port.StepSaying, Text: text}
								}
							case "agent_thought":
								w.steps <- port.Step{Kind: port.StepThinking}
							case "tool_call":
								toolName, _ := update["tool"].(string)
								about := formatToolArgs(update["arguments"])
								w.steps <- port.Step{Kind: port.StepToolCall, Tool: toolName, About: about}
							case "tool_result":
								w.steps <- port.Step{Kind: port.StepAnswered}
							}
						}
					}
				}

				// Result status
				if resMap, ok := rawMap["result"].(map[string]any); ok {
					if st, _ := resMap["status"].(string); st == "ERROR" {
						errMsg, _ := resMap["error"].(string)
						w.steps <- port.Step{Kind: port.StepStopped, Detail: errMsg}
					}
				}
			}
		}

		_ = cmd.Wait()
	}()

	return w, nil
}

func formatToolArgs(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]any); ok {
		for _, k := range []string{"Query", "query", "path", "file", "TargetFile", "SearchPath", "CommandLine"} {
			if val, ok := m[k].(string); ok && val != "" {
				return val
			}
		}
		if b, err := json.Marshal(m); err == nil {
			return string(b)
		}
	}
	return fmt.Sprint(v)
}
