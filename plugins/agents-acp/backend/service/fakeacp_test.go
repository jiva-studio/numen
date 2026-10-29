package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain runs the test binary as a fake ACP agent when it is started as one.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "fake-acp" {
		os.Exit(runFakeACPAgent(os.Args[2:], os.Stdin, os.Stdout))
	}
	os.Exit(m.Run())
}

// fakeACPAgent returns an agent served by this test binary. The behaviour is one of: "config"
// reports models m1 and m2 as a config option, "thought" does so and adds the depths low, medium and high of
// the category thought_level, "legacy" reports them the older way, "plain"
// reports none, "tools" reports none, accepts MCP servers over HTTP and calls tools as callTools specifies, "fail" exits immediately, "refuse" ends its turn declining to answer, and "hang" never ends it.
// Every line the agent reads is appended to the record.
func fakeACPAgent(t *testing.T, program, behaviour string) (acpAgent, string) {
	t.Helper()
	record := filepath.Join(t.TempDir(), "record")
	return acpAgent{
		Program: program,
		Title:   "Fake " + program,
		Command: []string{os.Args[0], "fake-acp", behaviour, record},
	}, record
}

// readRecord returns the messages the fake agent read, by method; a reply to the agent is kept
// under "reply".
func readRecord(t *testing.T, record string) map[string][]map[string]any {
	t.Helper()
	raw, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	observed := map[string][]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var message map[string]any
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			t.Fatalf("the agent read %q: %v", line, err)
		}
		method, _ := message["method"].(string)
		if method == "" {
			method = "reply"
		}
		observed[method] = append(observed[method], message)
	}
	return observed
}

func runFakeACPAgent(args []string, in io.Reader, out io.Writer) int {
	behaviour, record := args[0], args[1]
	if behaviour == "fail" {
		if started, err := os.OpenFile(record, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
			fmt.Fprintln(started, `{"method":"started"}`)
			_ = started.Close()
		}
		fmt.Fprintln(os.Stderr, "not signed in")
		return 1
	}
	recording, err := os.OpenFile(record, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 2
	}
	defer recording.Close()
	agent := &fakeAgent{behaviour: behaviour, out: out, lines: bufio.NewScanner(in), record: recording}
	agent.lines.Buffer(nil, 4<<20)
	for agent.lines.Scan() {
		agent.handle()
	}
	return 0
}

type fakeAgent struct {
	behaviour string
	out       io.Writer
	lines     *bufio.Scanner
	record    io.Writer
}

func (a *fakeAgent) send(message map[string]any) {
	message["jsonrpc"] = "2.0"
	raw, _ := json.Marshal(message)
	fmt.Fprintf(a.out, "%s\n", raw)
}

// read returns the next message from the client.
func (a *fakeAgent) read() map[string]any {
	if !a.lines.Scan() {
		return nil
	}
	return a.decode()
}

func (a *fakeAgent) decode() map[string]any {
	fmt.Fprintf(a.record, "%s\n", a.lines.Bytes())
	var message map[string]any
	_ = json.Unmarshal(a.lines.Bytes(), &message)
	return message
}

func (a *fakeAgent) handle() {
	message := a.decode()
	id := message["id"]
	switch message["method"] {
	case "initialize":
		capabilities := map[string]any{"sessionCapabilities": map[string]any{"close": map[string]any{}}}
		if a.behaviour == "tools" {
			capabilities["mcpCapabilities"] = map[string]any{"http": true}
		}
		a.send(map[string]any{"id": id, "result": map[string]any{"protocolVersion": 1, "agentCapabilities": capabilities}})
	case "session/new":
		a.send(map[string]any{"id": id, "result": a.getSession()})
	case "session/prompt":
		a.answerPrompt(id)
	default:
		a.send(map[string]any{"id": id, "result": map[string]any{}})
	}
}

func (a *fakeAgent) getSession() map[string]any {
	session := map[string]any{"sessionId": "s1"}
	models := []map[string]any{{"value": "m1", "name": "Model One"}, {"value": "m2", "name": "Model Two"}}
	switch a.behaviour {
	case "config", "thought":
		session["configOptions"] = []map[string]any{
			{"id": "mode", "category": "mode", "type": "select", "currentValue": "ask", "options": []map[string]any{{"value": "ask"}}},
			{"id": "model", "category": "model", "type": "select", "currentValue": "m1", "options": []map[string]any{
				{"group": "g", "name": "Group", "options": models},
			}},
		}
		if a.behaviour == "thought" {
			session["configOptions"] = append(session["configOptions"].([]map[string]any), map[string]any{
				"id": "effort", "category": "thought_level", "type": "select", "currentValue": "medium", "options": []map[string]any{
					{"value": "low", "name": "Low"}, {"value": "medium", "name": "Medium"}, {"value": "high", "name": "High"},
				},
			})
		}
	case "legacy":
		session["models"] = map[string]any{"currentModelId": "m1", "availableModels": []map[string]any{
			{"modelId": "m1", "name": "Model One"}, {"modelId": "m2", "name": "Model Two"},
		}}
	}
	return session
}

// answerPrompt requests a permission and a file, then sends the answer in chunks
// among updates the client ignores.
func (a *fakeAgent) answerPrompt(id any) {
	if a.behaviour == "hang" {
		return
	}
	if a.behaviour == "tools" {
		a.callTools()
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
	if a.behaviour == "refuse" {
		stopReason = "rejected"
	}
	a.send(map[string]any{"id": id, "result": map[string]any{"stopReason": stopReason}})
}

// callTools reports a call of note_read whose title holds its arguments, as Gemini writes one, requests
// to run it by its id alone and reports it completed. It then requests to run a shell command titled as
// a run tool, a call with no name titled as a run tool, and a call named as the MCP server's
// note_read. The client's replies are recorded under "reply" in that order.
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
}

func (a *fakeAgent) update(update map[string]any) {
	a.send(map[string]any{"method": "session/update", "params": map[string]any{"sessionId": "s1", "update": update}})
}

// askPermission requests to run the tool call and reads the reply.
func (a *fakeAgent) askPermission(id string, toolCall map[string]any) {
	a.send(map[string]any{"id": id, "method": "session/request_permission", "params": map[string]any{
		"sessionId": "s1", "toolCall": toolCall,
		"options": []map[string]any{
			{"optionId": "always", "name": "Always allow", "kind": "allow_always"},
			{"optionId": "once", "name": "Allow", "kind": "allow_once"},
			{"optionId": "no", "name": "Reject", "kind": "reject_once"},
		},
	}})
	a.read()
}
