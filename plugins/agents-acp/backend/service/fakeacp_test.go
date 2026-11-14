package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMain runs the test binary as a fake ACP agent when it is started as one, and the tests of m
// otherwise.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "fake-acp" {
		time.AfterFunc(20*time.Second, func() { os.Exit(0) })
		os.Exit(runFakeACPAgent(os.Args[2:], os.Stdin, os.Stdout))
	}
	if os.Getenv("NUMEN_ACP_TEST_CHILD") == "1" {
		os.Exit(2)
	}
	if err := os.Setenv("NUMEN_ACP_TEST_CHILD", "1"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func fakeProgram(t *testing.T, behavior string) (program, record string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake program is a shell script")
	}
	record = filepath.Join(t.TempDir(), "record")
	program = filepath.Join(t.TempDir(), "fake-agent")
	script := "#!/bin/sh\nexec " + quoteShell(os.Args[0]) + " fake-acp " + behavior + " " + quoteShell(record) + "\n"
	if err := os.WriteFile(program, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return program, record
}

func quoteShell(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

func readRecord(t *testing.T, record string) map[string][]map[string]any {
	t.Helper()
	raw, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	observed := map[string][]map[string]any{}
	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		var message map[string]any
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			t.Fatalf("the agent read %q: %v", line, err)
		}
		method := "reply"
		if name, ok := message["method"].(string); ok && name != "" {
			method = name
		}
		observed[method] = append(observed[method], message)
	}
	return observed
}

func runFakeACPAgent(args []string, in io.Reader, out io.Writer) int {
	behavior, record := args[0], args[1]
	if behavior == "fail" {
		started, err := os.OpenFile(record, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		_, err = fmt.Fprintln(started, `{"method":"started"}`)
		if err = errors.Join(err, started.Close()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		fmt.Fprintln(os.Stderr, "not signed in")
		return 1
	}
	recording, err := os.OpenFile(record, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 2
	}
	agent := &fakeAgent{behavior: behavior, out: out, lines: bufio.NewScanner(in), record: recording}
	agent.recordFolder()
	agent.lines.Buffer(nil, 4<<20)
	for agent.lines.Scan() {
		agent.handle()
	}
	if err := errors.Join(agent.err, recording.Close()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return 0
}

type fakeAgent struct {
	behavior string
	out      io.Writer
	lines    *bufio.Scanner
	record   io.Writer
	err      error
	pending  any
	mcpURL   string
	mcpToken string
}

func (a *fakeAgent) recordFolder() {
	folder, err := os.Getwd()
	a.fail(err)
	entries, err := os.ReadDir(folder)
	a.fail(err)
	raw, err := json.Marshal(map[string]any{"method": "folder", "path": folder, "entries": len(entries)})
	a.fail(err)
	_, err = fmt.Fprintf(a.record, "%s\n", raw)
	a.fail(err)
}

func (a *fakeAgent) fail(err error) {
	if a.err == nil {
		a.err = err
	}
}

func (a *fakeAgent) send(message map[string]any) {
	message["jsonrpc"] = "2.0"
	raw, err := json.Marshal(message)
	if err != nil {
		a.fail(err)
		return
	}
	_, err = fmt.Fprintf(a.out, "%s\n", raw)
	a.fail(err)
}

func (a *fakeAgent) read() map[string]any {
	if !a.lines.Scan() {
		return nil
	}
	return a.decode()
}

func (a *fakeAgent) decode() map[string]any {
	_, err := fmt.Fprintf(a.record, "%s\n", a.lines.Bytes())
	a.fail(err)
	var message map[string]any
	a.fail(json.Unmarshal(a.lines.Bytes(), &message))
	return message
}

func (a *fakeAgent) handle() {
	message := a.decode()
	id := message["id"]
	switch message["method"] {
	case "initialize":
		capabilities := map[string]any{"sessionCapabilities": map[string]any{"close": map[string]any{}}}
		if a.behavior != "no-http" {
			capabilities["mcpCapabilities"] = map[string]any{"http": true}
		}
		a.send(map[string]any{"id": id, "result": map[string]any{"protocolVersion": 1, "agentCapabilities": capabilities}})
	case "session/new":
		a.keepServer(message)
		a.send(map[string]any{"id": id, "result": a.getSession()})
	case "session/cancel":
		if a.pending != nil {
			a.send(map[string]any{"id": a.pending, "result": map[string]any{"stopReason": "canceled"}})
			a.pending = nil
		}
	case "session/prompt":
		a.answerPrompt(id)
	case "session/set_config_option":
		if a.behavior == "fail-reasoning" {
			a.send(map[string]any{"id": id, "error": map[string]any{"code": -32000, "message": "reasoning rejected"}})
			return
		}
		a.send(map[string]any{"id": id, "result": map[string]any{}})
	case "session/close":
		if a.behavior == "fail-close" {
			a.send(map[string]any{"id": id, "error": map[string]any{"code": -32000, "message": "close rejected"}})
			return
		}
		a.send(map[string]any{"id": id, "result": map[string]any{}})
	default:
		a.send(map[string]any{"id": id, "result": map[string]any{}})
	}
}

func (a *fakeAgent) getSession() map[string]any {
	session := map[string]any{"sessionId": "s1"}
	models := []map[string]any{{"value": "m1", "name": "Model One"}, {"value": "m2", "name": "Model Two"}}
	switch a.behavior {
	case "config", "thought", "fail-reasoning":
		options := []map[string]any{
			{"id": "mode", "category": "mode", "type": "select", "currentValue": "ask", "options": []map[string]any{{"value": "ask"}}},
			{"id": "model", "category": "model", "type": "select", "currentValue": "m1", "options": []map[string]any{
				{"group": "g", "name": "Group", "options": models},
			}},
		}
		if a.behavior == "thought" || a.behavior == "fail-reasoning" {
			options = append(options, map[string]any{
				"id": "effort", "category": "thought_level", "type": "select", "currentValue": "medium", "options": []map[string]any{
					{"value": "low", "name": "Low"}, {"value": "medium", "name": "Medium"}, {"value": "high", "name": "High"},
				},
			})
		}
		session["configOptions"] = options
	case "legacy":
		session["models"] = map[string]any{"currentModelId": "m1", "availableModels": []map[string]any{
			{"modelId": "m1", "name": "Model One"}, {"modelId": "m2", "name": "Model Two"},
		}}
	}
	return session
}

func (a *fakeAgent) answerPrompt(id any) {
	switch a.behavior {
	case "hang":
		a.pending = id
		return
	case "stream":
		a.streamAnswer()
		a.send(map[string]any{"id": id, "result": map[string]any{"stopReason": "end_turn"}})
		return
	case "edit":
		a.editFile()
		a.send(map[string]any{"id": id, "result": map[string]any{"stopReason": "end_turn"}})
		return
	case "mcp":
		a.callServedTool()
		a.send(map[string]any{"id": id, "result": map[string]any{"stopReason": "end_turn"}})
		return
	case "tools":
		a.callTools()
	}
	if a.behavior == "write" {
		a.requestWrites()
	}
	a.send(map[string]any{"id": "p1", "method": "session/request_permission", "params": map[string]any{
		"sessionId": "s1", "toolCall": map[string]any{"toolCallId": "t1"},
		"options": []map[string]any{{"optionId": "yes", "name": "Allow", "kind": "allow_once"}},
	}})
	a.read()
	a.send(map[string]any{"id": "p2", "method": "fs/read_text_file", "params": map[string]any{"sessionId": "s1", "path": "/etc/hosts"}})
	a.read()
	for _, update := range []map[string]any{
		{"sessionId": "s1", "update": map[string]any{"sessionUpdate": "agent_thought_chunk", "content": map[string]any{"type": "text", "text": "thinking "}}},
		{"sessionId": "s1", "update": map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": "the "}}},
		{"sessionId": "other", "update": map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": "elsewhere "}}},
		{"sessionId": "s1", "update": map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": "answer"}}},
	} {
		a.send(map[string]any{"method": "session/update", "params": update})
	}
	stopReason := "end_turn"
	if a.behavior == "refuse" {
		stopReason = "refusal"
	}
	a.send(map[string]any{"id": id, "result": map[string]any{"stopReason": stopReason}})
}

func (a *fakeAgent) requestWrites() {
	a.askPermission("w1", map[string]any{"toolCallId": "w1", "name": "write_file", "title": "Write note.md", "kind": "edit"})
	a.send(map[string]any{"id": "w2", "method": "fs/write_text_file", "params": map[string]any{"sessionId": "s1", "path": "note.md", "content": "x"}})
	a.read()
}

func (a *fakeAgent) callTools() {
	a.update(map[string]any{
		"sessionUpdate": "tool_call", "toolCallId": "c0", "name": "note_read", "title": `{"path":"Garden.md"}`,
		"kind": "read", "status": "pending", "rawInput": map[string]any{"path": "Garden.md"},
	})
	a.askPermission("t0", map[string]any{"toolCallId": "c0", "title": `{"path":"Garden.md"}`})
	a.update(map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "c0", "status": "in_progress"})
	a.update(map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "c0", "status": "completed"})
	a.update(map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "c0", "status": "completed"})
	a.askPermission("t1", map[string]any{"toolCallId": "c1", "name": "run_shell_command", "title": "mcp__numen__note_read"})
	a.askPermission("t2", map[string]any{"toolCallId": "c2", "title": "mcp__numen__note_read"})
	a.askPermission("t3", map[string]any{"toolCallId": "c3", "name": "mcp__numen__note_read", "title": "Read"})
	a.askPermission("t4", map[string]any{"toolCallId": "c4", "title": "mcp.numen.note_read", "rawInput": map[string]any{
		"server": "numen", "tool": "note_read", "arguments": map[string]any{"path": "Garden.md"},
	}})
	a.askPermission("t5", map[string]any{"toolCallId": "c5", "title": "mcp.orchard.note_read", "rawInput": map[string]any{
		"server": "orchard", "tool": "note_read", "arguments": map[string]any{"path": "Garden.md"},
	}})
}

func (a *fakeAgent) keepServer(message map[string]any) {
	if address, isText := dig(message, "params", "mcpServers", 0, "url").(string); isText {
		a.mcpURL = address
	}
	if value, isText := dig(message, "params", "mcpServers", 0, "headers", 0, "value").(string); isText {
		a.mcpToken = strings.TrimPrefix(value, "Bearer ")
	}
}

func dig(value any, path ...any) any {
	for _, step := range path {
		switch at := step.(type) {
		case string:
			object, isObject := value.(map[string]any)
			if !isObject {
				return nil
			}
			value = object[at]
		case int:
			list, isList := value.([]any)
			if !isList || at >= len(list) {
				return nil
			}
			value = list[at]
		}
	}
	return value
}

func (a *fakeAgent) streamAnswer() {
	a.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": "Raised "}})
	a.update(map[string]any{"sessionUpdate": "plan", "entries": []map[string]any{
		{"content": "Read the garden note", "priority": "high", "status": "pending"},
	}})
	a.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": "beds."}})
}

func (a *fakeAgent) editFile() {
	a.update(map[string]any{"sessionUpdate": "agent_thought_chunk", "content": map[string]any{"type": "text", "text": "The beds "}})
	a.update(map[string]any{"sessionUpdate": "agent_thought_chunk", "content": map[string]any{"type": "text", "text": "need a list."}})
	a.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": "Writing it."}})
	a.update(map[string]any{"sessionUpdate": "agent_thought_chunk", "content": map[string]any{"type": "text", "text": "Start north."}})
	a.update(map[string]any{
		"sessionUpdate": "tool_call", "toolCallId": "e1", "title": "Write beds.md", "kind": "edit", "status": "pending",
		"content": []map[string]any{
			{"type": "content", "content": map[string]any{"type": "text", "text": "A new note"}},
			{"type": "diff", "path": "/garden/beds.md", "oldText": nil, "newText": "# Beds"},
		},
	})
	a.update(map[string]any{
		"sessionUpdate": "tool_call_update", "toolCallId": "e1", "status": "in_progress",
		"content": []map[string]any{{"type": "diff", "path": "/garden/beds.md", "oldText": "# Beds", "newText": "# Beds\nNorth"}},
	})
	a.update(map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "e1", "status": "completed"})
}

func (a *fakeAgent) callServedTool() {
	meta := map[string]any{"claudeCode": map[string]any{"toolName": "mcp__numen__note_read"}}
	a.update(map[string]any{
		"sessionUpdate": "tool_call", "toolCallId": "c1", "title": "mcp__numen__note_read", "kind": "other",
		"status": "pending", "rawInput": map[string]any{"path": "Garden.md"}, "_meta": meta,
	})
	reply := a.askPermission("t1", map[string]any{"toolCallId": "c1", "_meta": meta})
	if dig(reply, "result", "outcome", "optionId") != "once" {
		a.update(map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "c1", "status": "failed"})
		return
	}
	text, err := a.callMCPTool("note_read", map[string]any{"path": "Garden.md"})
	a.fail(err)
	a.update(map[string]any{
		"sessionUpdate": "tool_call_update", "toolCallId": "c1", "status": "completed", "rawOutput": map[string]any{"text": text},
	})
	a.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]any{"type": "text", "text": text}})
}

func (a *fakeAgent) callMCPTool(name string, arguments map[string]any) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	token := a.mcpToken
	client := &http.Client{Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Authorization", "Bearer "+token)
		return http.DefaultTransport.RoundTrip(req)
	})}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "fake-agent", Version: "1"}, nil).
		Connect(ctx, &sdk.StreamableClientTransport{Endpoint: a.mcpURL, HTTPClient: client, DisableStandaloneSSE: true}, nil)
	if err != nil {
		return "", err
	}
	defer func() { a.fail(session.Close()) }()
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return "", err
	}
	var names []string
	for _, one := range listed.Tools {
		names = append(names, one.Name)
	}
	raw, err := json.Marshal(map[string]any{"method": "mcp/tools", "names": names})
	if err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(a.record, "%s\n", raw); err != nil {
		return "", err
	}
	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return "", err
	}
	var text string
	for _, content := range result.Content {
		if one, isText := content.(*sdk.TextContent); isText {
			text += one.Text
		}
	}
	return text, nil
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func (a *fakeAgent) update(update map[string]any) {
	a.send(map[string]any{"method": "session/update", "params": map[string]any{"sessionId": "s1", "update": update}})
}

func (a *fakeAgent) askPermission(id string, toolCall map[string]any) map[string]any {
	a.send(map[string]any{"id": id, "method": "session/request_permission", "params": map[string]any{
		"sessionId": "s1", "toolCall": toolCall,
		"options": []map[string]any{
			{"optionId": "always", "name": "Always allow", "kind": "allow_always"},
			{"optionId": "once", "name": "Allow", "kind": "allow_once"},
			{"optionId": "no", "name": "Reject", "kind": "reject_once"},
		},
	}})
	return a.read()
}

func getField[T any](t *testing.T, object any, key string) T {
	t.Helper()
	var zero T
	fields, ok := object.(map[string]any)
	if !ok {
		t.Fatalf("%T is not a JSON object", object)
		return zero
	}
	value, ok := fields[key].(T)
	if !ok {
		t.Fatalf("%q holds %T, want %T", key, fields[key], zero)
		return zero
	}
	return value
}
