package claudecode_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/agent/claudecode"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agent"
)

// started is an agent whose command line is a script printing what it was told
// to print. What is tested is the reading and the stopping: the tools are the
// server's business and the answering is the model's.
func started(t *testing.T, prints string) agent.Work {
	t.Helper()

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := "#!/bin/sh\ncat <<'SAID'\n" + prints + "\nSAID\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
		Words: map[string]claudecode.Words{
			claudecode.Tool("note_search"): {Title: "Search notes", About: "query"},
		},
	}
	work, err := claude.Take(t.Context(), agent.Task{Asked: "what is here?"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { work.Stop() })
	return work
}

// heard is every step of a piece of work, in order.
func heard(t *testing.T, work agent.Work) []agent.Step {
	t.Helper()

	var steps []agent.Step
	for step := range work.Steps() {
		steps = append(steps, step)
	}
	return steps
}

const connected = `{"type":"system","subtype":"init","session_id":"s1",` +
	`"mcp_servers":[{"name":"numen","status":"connected"}]}`

func TestSaysWhatTheAgentSaid(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Two notes."}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false,"result":"Two notes."}`)

	steps := heard(t, work)
	if len(steps) != 2 {
		t.Fatalf("expected saying and stopping, got %d steps: %+v", len(steps), steps)
	}
	if steps[0].Kind != agent.Saying || steps[0].Text != "Two notes." {
		t.Errorf("first step is %+v", steps[0])
	}
	if steps[1].Kind != agent.Stopped || steps[1].Failed != "" {
		t.Errorf("last step is %+v", steps[1])
	}
}

func TestNamesAToolAsItNamedItself(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__numen__note_search",`+
		`"input":{"query":"entropy"}}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if steps[0].Kind != agent.Calling || steps[0].Tool != "Search notes" || steps[0].About != "entropy" {
		t.Errorf("expected the tool's own title and what it was asked, got %+v", steps[0])
	}
}

// A tool this vault does not serve is named as the agent named it: there is
// nothing declared here to read it by.
func TestNamesAToolItWasNotToldAbout(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if steps[0].Tool != "Bash" || steps[0].About != "" {
		t.Errorf("steps are %+v", steps)
	}
}

// Words arrive a piece at a time, and a call is shown once it is whole.
func TestReadsWordsAndCallsAsTheyAreWritten(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Two "}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"notes."}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"mcp__numen__note_search"}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{\"query\":\"ent"}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"ropy\"}"}}}`+"\n"+
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Two notes."}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if len(steps) != 4 {
		t.Fatalf("the whole message was said again: %+v", steps)
	}
	if steps[0].Text != "Two " || steps[1].Text != "notes." {
		t.Errorf("words are %+v", steps[:2])
	}
	if steps[2].Kind != agent.Calling || steps[2].About != "entropy" {
		t.Errorf("call is %+v", steps[2])
	}
}

func TestPassesOverALineItCannotRead(t *testing.T) {
	work := started(t, connected+"\n"+
		"not json at all\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"still here"}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if steps[0].Kind != agent.Saying || steps[0].Text != "still here" {
		t.Errorf("a line it could not read stopped the work: %+v", steps)
	}
}

func TestSaysWhyItStopped(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"result","subtype":"error_max_turns","is_error":true,"result":"went round too many times"}`)

	steps := heard(t, work)
	last := steps[len(steps)-1]
	if last.Kind != agent.Stopped || last.Failed != "went round too many times" {
		t.Errorf("last step is %+v", last)
	}
}

func TestSaysWhenTheVaultDidNotReachTheAgent(t *testing.T) {
	work := started(t, `{"type":"system","subtype":"init","session_id":"s1","mcp_servers":[]}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	last := steps[len(steps)-1]
	if last.Kind != agent.Stopped || !strings.Contains(last.Failed, "without this vault") {
		t.Errorf("an agent that never got the tools answered anyway: %+v", last)
	}
}

// A message of somebody else's shape must not stop the reading: what a user
// message carries is not what an assistant message carries.
func TestReadsPastAMessageOfAnotherShape(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"user","message":{"role":"user","content":"a string, not blocks"}}`+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"after"}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	if steps[0].Kind != agent.Saying || steps[0].Text != "after" {
		t.Errorf("steps are %+v", steps)
	}
}

func TestStoppingLeavesNothingRunning(t *testing.T) {
	work := started(t, connected+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"a word"}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	if err := work.Stop(); err != nil {
		t.Fatal(err)
	}
	// Stopping twice is what a panel closed twice does.
	if err := work.Stop(); err != nil {
		t.Fatal(err)
	}
	for range work.Steps() {
		// Draining what was already read is fine; the channel must close.
	}
}

// A version that says nothing about servers is not a version that says the
// vault never arrived.
func TestSaysNothingWhenTheLineSaysNothingAboutServers(t *testing.T) {
	work := started(t, `{"type":"system","subtype":"init","session_id":"s1"}`+"\n"+
		`{"type":"assistant","message":{"content":[{"type":"text","text":"here"}]}}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false}`)

	steps := heard(t, work)
	last := steps[len(steps)-1]
	if last.Kind != agent.Stopped || last.Failed != "" {
		t.Errorf("last step is %+v", last)
	}
}
