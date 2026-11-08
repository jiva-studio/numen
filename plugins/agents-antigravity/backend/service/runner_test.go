package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
)

func TestResolveCommandWithEmptyCommand(t *testing.T) {
	withoutAgentsAnywhere(t)

	bin, args := resolveCommand(nil)
	if want := FindAntigravity(); bin != want || len(args) != 0 {
		t.Errorf("expected %s and empty args, got %s, %v", want, bin, args)
	}
}

func TestTheSessionOfTheConversationIsHandedToTheNextRun(t *testing.T) {
	var reg agentrun.Registry
	reg.SetSession("c-1", "s-1")

	cmd, _, err := makeAntigravityCommand(t.Context(), &agentrun.Request{Conversation: "c-1", Prompt: "q"}, &reg)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(cmd.Args, "--conversation") || !slices.Contains(cmd.Args, "s-1") {
		t.Errorf("the arguments are %v", cmd.Args)
	}
}

func TestTheSessionAndTheContentOfAntigravityAreReported(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"session_id":"s-1","content":"hello"}'`+"\n"+
			`echo '{"tool_calls":[{"name":"read_note","arguments":{"path":"a.md"}}]}'`+"\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(
		&agentrun.Request{Command: []string{script}, Conversation: "c-1"}, &reg))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Stop(); err != nil {
			t.Error(err)
		}
	}()

	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 2 || got[0].GetText() != "hello" || got[1].GetToolCall().Tool != "read_note" {
		t.Fatalf("the steps are %v", got)
	}
	if session := reg.GetSession("c-1"); session != "s-1" {
		t.Errorf("expected session s-1, got %q", session)
	}
}

func TestAFailedHomeSetupEndsTheRunBeforeItStarts(t *testing.T) {
	setUnusableTempDir(t)
	var reg agentrun.Registry

	_, err := start(t.Context(), &reg, newSpec(
		&agentrun.Request{Command: []string{"agy"}, ToolsURL: "http://127.0.0.1:1/mcp"}, &reg))

	if err == nil || !strings.Contains(err.Error(), "set up the Antigravity home") {
		t.Errorf("got %v", err)
	}
}

func TestTheToolCallsOfAntigravityAreCountedAgainstTheStepLimit(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`while true; do echo '{"tool_calls":[{"name":"read_note","arguments":{}}]}'; done`+"\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(
		&agentrun.Request{Command: []string{script}, MaxSteps: 2}, &reg))
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if last := got[len(got)-1]; last.GetStopped() != "reached maximum limit of 2 steps" {
		t.Errorf("the last step is %v", last)
	}
}

func TestAStreamedToolCallIsCountedAgainstTheStepLimit(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`while true; do
echo '{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"read_note"}}}'
echo '{"type":"stream_event","event":{"type":"content_block_stop"}}'
done`+"\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(
		&agentrun.Request{Command: []string{script}, MaxSteps: 1}, &reg))
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if last := got[len(got)-1]; last.GetStopped() != "reached maximum limit of 1 steps" {
		t.Errorf("the last step is %v", last)
	}
}

func setUnusableTempDir(t *testing.T) {
	t.Helper()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, filepath.Join(t.TempDir(), "absent"))
	}
}

func collect(t *testing.T, script string) ([]*agentrun.Response, *agentrun.Registry) {
	t.Helper()
	var reg agentrun.Registry
	w, err := start(t.Context(), &reg, newSpec(
		&agentrun.Request{Command: []string{script}, Conversation: "c-1", Prompt: "q"}, &reg))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Stop(); err != nil {
			t.Error(err)
		}
	}()
	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	return got, &reg
}

// The CLI answer arrives as the text deltas of its response steps. Its final
// result repeats them, so it is not emitted again.
func TestRunner_CLIAnswerEmittedOnce(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"event":"init","conversation_id":"s-1","init":{"model":"m"}}'`+"\n"+
			`echo '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":0,"state":"DONE","step_type":"user_input"}}'`+"\n"+
			`echo '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"ok"}}'`+"\n"+
			`printf '%s\n' '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":1,"state":"DONE","step_type":"agent_response","text_delta":"\n"}}'`+"\n"+
			`printf '%s\n' '{"event":"result","result":{"conversation_id":"s-1","status":"SUCCESS","response":"ok\n"}}'`+"\n")

	got, reg := collect(t, script)

	var response strings.Builder
	for _, resp := range got {
		if resp.GetStopped() != "" {
			t.Fatalf("stopped: %v", resp)
		}
		response.WriteString(resp.GetText())
	}
	if response.String() != "ok\n" {
		t.Errorf("said %q, want %q", response.String(), "ok\n")
	}
	if session := reg.GetSession("c-1"); session != "s-1" {
		t.Errorf("session %q, want s-1", session)
	}
}

// A result with a response and no streamed text emits the response.
func TestRunner_CLIResponseEmittedWhenNothingStreamed(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"event":"result","result":{"status":"SUCCESS","response":"fine"}}'`+"\n")

	got, _ := collect(t, script)

	if len(got) != 1 || got[0].GetText() != "fine" {
		t.Errorf("the steps are %v", got)
	}
}

// A result event ends the run even when a child process keeps the output pipe open.
func TestResultEventEndsRunWithOpenChildOutput(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		"(sleep 60) &\n"+
			`echo '{"event":"step_update","step_update":{"step_type":"agent_response","text_delta":"done"}}'`+"\n"+
			`echo '{"event":"result","result":{"status":"SUCCESS","response":"done"}}'`+"\n"+
			"wait\n")

	got, _ := collect(t, script)

	if len(got) != 1 || got[0].GetText() != "done" {
		t.Errorf("the steps are %v", got)
	}
}

// A result in error stops the run with the error the CLI gave.
func TestAnErrorResultOfTheCliStopsTheRunWithItsError(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"event":"result","result":{"status":"ERROR","response":"","error":"Eligibility check failed: 503"}}'`+"\n"+
			`echo 'error: Eligibility check failed: 503' >&2`+"\nexit 1\n")

	got, _ := collect(t, script)

	if len(got) == 0 || got[0].GetStopped() != "Eligibility check failed: 503" {
		t.Errorf("the steps are %v", got)
	}
}

// A model the CLI does not know ends it with the error and the list of the models on its error
// output: the run stops with the error line.
func TestAModelTheCliDoesNotKnowStopsTheRunWithTheErrorLine(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo 'error: invalid model selection (--model "m-old"): not recognized' >&2`+"\n"+
			`printf 'Available models:\n  Model One\n  Model Two\n' >&2`+"\nexit 1\n")

	got, _ := collect(t, script)

	want := `invalid model selection (--model "m-old"): not recognized`
	if len(got) != 1 || got[0].GetStopped() != want {
		t.Errorf("the steps are %v, want a stop of %q", got, want)
	}
}

func TestEventsAndJsonFieldCompleteCoverage(t *testing.T) {
	if got := getString(map[string]any{"k": 123}, "k"); got != "" {
		t.Errorf("getString non-string = %q, want empty", got)
	}
	if got := getMap(map[string]any{"k": "not-map"}, "k"); got != nil {
		t.Errorf("getMap non-map = %v, want nil", got)
	}

	var reg agentrun.Registry
	w := agentrun.NewWork("conv-1", 64, func() error { return nil })
	r := &reader{work: w, registry: &reg, conversation: "conv-1"}

	// SetSession ignores empty session or empty conversation.
	r.SetSession("")
	if got := reg.GetSession("conv-1"); got != "" {
		t.Errorf("expected empty session after SetSession(\"\"), got %q", got)
	}
	noConvReader := &reader{work: w, registry: &reg, conversation: ""}
	noConvReader.SetSession("ignored-sess")

	handle := newEventHandler(r)

	// Top-level thinking, content, delta, result, stopped, and tool_calls edge cases.
	handle(map[string]any{
		"conversation_id": "s-top",
		"thinking":        "pondering",
		"content":         "main content",
		"delta":           map[string]any{"text": "step delta"},
		"result":          "result text",
		"stopped":         "stopped reason",
		"tool_calls": []any{
			"not-a-map",
			map[string]any{"name": "bad_args_tool", "arguments": make(chan int)},
		},
	})
	// Empty/non-matching variants should emit nothing.
	countBefore := r.sentCount
	handle(map[string]any{
		"thinking":   "",
		"content":    "",
		"delta":      map[string]any{"text": ""},
		"result":     "",
		"stopped":    "",
		"tool_calls": "not-a-slice",
	})
	handle(map[string]any{"delta": "not-a-map"})
	if r.sentCount != countBefore {
		t.Errorf("expected no steps for empty fields, got %d new steps", r.sentCount-countBefore)
	}

	// Stream events: non-map event, non-map content_block, non-tool_use block, tool_use with non-map/unmarshalable/valid input, delta text & partial_json, and content_block_stop.
	handle(map[string]any{"type": "stream_event", "event": "not-map"})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_start", "content_block": "not-map"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_start", "content_block": map[string]any{"type": "text"}}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_stop"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "delta": "not-map"}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_delta", "delta": map[string]any{"text": ""}}})

	// Tool use with non-map input, then unmarshalable input, then valid input + partial_json + stop.
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type":          "content_block_start",
		"content_block": map[string]any{"type": "tool_use", "name": "t1", "input": "not-map"},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type":          "content_block_start",
		"content_block": map[string]any{"type": "tool_use", "name": "t2", "input": map[string]any{"ch": make(chan int)}},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type":          "content_block_start",
		"content_block": map[string]any{"type": "tool_use", "name": "t3", "input": map[string]any{"k": "v"}},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{
		"type":  "content_block_delta",
		"delta": map[string]any{"text": "stream-text", "partial_json": "_extra"},
	}})
	handle(map[string]any{"type": "stream_event", "event": map[string]any{"type": "content_block_stop"}})

	// CLI events: default without conversation_id, result with non-success status and empty error, result with empty response.
	handle(map[string]any{"event": "init"})
	handle(map[string]any{"event": "result", "result": map[string]any{"status": "CANCELLED", "error": ""}})
	handle(map[string]any{"event": "result", "result": map[string]any{"status": "SUCCESS", "response": ""}})

	close(w.Output)
	var steps []*agentrun.Response
	for resp := range w.Output {
		steps = append(steps, resp)
	}
	if len(steps) < 7 {
		t.Fatalf("expected at least 7 steps, got %d: %+v", len(steps), steps)
	}
	if !steps[0].IsModelStarted() {
		t.Errorf("step[0] = %+v, want ModelStarted", steps[0])
	}
	if steps[1].GetToolCall() == nil || steps[1].GetToolCall().Tool != "bad_args_tool" || steps[1].GetToolCall().About != "" {
		t.Errorf("step[1] = %+v, want bad_args_tool with empty About", steps[1])
	}
	if steps[len(steps)-2].GetToolCall() == nil || steps[len(steps)-2].GetToolCall().Tool != "t3" || steps[len(steps)-2].GetToolCall().About != `{"k":"v"}_extra` {
		t.Errorf("streamed tool call = %+v", steps[len(steps)-2])
	}
	if steps[len(steps)-1].GetStopped() != "the run ended with status CANCELLED" {
		t.Errorf("last step = %+v, want status CANCELLED", steps[len(steps)-1])
	}
}

func TestHomeSetupAndMCPConfigEdgeCases(t *testing.T) {
	if err := noRelease(); err != nil {
		t.Errorf("noRelease() = %v, want nil", err)
	}

	// readMCPServers with missing file and invalid JSON returns nil.
	if got := readMCPServers("/nonexistent/mcp_config.json"); got != nil {
		t.Errorf("readMCPServers missing = %v, want nil", got)
	}
	badJSONPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badJSONPath, []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readMCPServers(badJSONPath); got != nil {
		t.Errorf("readMCPServers bad JSON = %v, want nil", got)
	}

	// makeMCPConfig preserves existing non-numen servers and overwrites existing "numen" server.
	existingPath := filepath.Join(t.TempDir(), "mcp_config.json")
	if err := os.WriteFile(existingPath, []byte(`{"mcpServers":{"numen":{"serverUrl":"old"},"other":{"serverUrl":"http://other"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	merged := makeMCPConfig(Endpoint{URL: "http://new/mcp", Token: "tok"}, existingPath)
	servers, ok := merged["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("makeMCPConfig merged = %+v, want mcpServers map", merged)
	}
	numenSrv, ok := servers["numen"].(map[string]any)
	if !ok {
		t.Fatalf("makeMCPConfig merged = %+v, want numen server map", merged)
	}
	if numenSrv["serverUrl"] != "http://new/mcp" || servers["other"] == nil {
		t.Fatalf("makeMCPConfig merged = %+v", merged)
	}

	// setupAntigravityHome with tools != nil writes the numen server into tempConfig/mcp_config.json.
	home := t.TempDir()
	t.Setenv("HOME", home)
	tempHome, clean, err := setupAntigravityHome(&Endpoint{URL: "http://vault/mcp", Token: "vtok"})
	if err != nil {
		t.Fatalf("setupAntigravityHome with tools: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(tempHome, ".gemini", "config", "mcp_config.json"))
	if err != nil || !strings.Contains(string(written), "http://vault/mcp") || !strings.Contains(string(written), "Bearer vtok") {
		t.Errorf("written mcp_config.json = %q, %v", written, err)
	}
	if err := clean(); err != nil {
		t.Errorf("clean tempHome: %v", err)
	}

	// setupAntigravityHome error paths:
	// 1) HOME is a regular file -> LinkEntries(realHome, tempHome, ".gemini") fails.
	fileAsHome := filepath.Join(t.TempDir(), "file-home")
	if err := os.WriteFile(fileAsHome, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", fileAsHome)
	if _, _, err := setupAntigravityHome(nil); err == nil {
		t.Error("expected error when HOME is a regular file")
	}

	// 2) HOME/.gemini is a regular file -> LinkEntries(realGemini, tempGemini, "config") fails.
	homeWithGeminiFile := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeWithGeminiFile, ".gemini"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeWithGeminiFile)
	if _, _, err := setupAntigravityHome(nil); err == nil {
		t.Error("expected error when HOME/.gemini is a regular file")
	}

	// 3) HOME/.gemini/config is a regular file -> LinkEntries(realConfig, tempConfig, "mcp_config.json") fails.
	homeWithConfigFile := t.TempDir()
	if err := os.MkdirAll(filepath.Join(homeWithConfigFile, ".gemini"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homeWithConfigFile, ".gemini", "config"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeWithConfigFile)
	if _, _, err := setupAntigravityHome(nil); err == nil {
		t.Error("expected error when HOME/.gemini/config is a regular file")
	}

	// 4) No HOME environment variable -> os.UserHomeDir() fails.
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	if _, err := os.UserHomeDir(); err != nil {
		if _, _, err := setupAntigravityHome(nil); err == nil {
			t.Error("expected error when os.UserHomeDir fails")
		}
	}
}

func TestRunnerErrorPathsAndLimitsAndCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Reader limits, ToolFinishedStep increment, and Send on stopped work.
	w := agentrun.NewWork("c", 4, func() error { return nil })
	r := &reader{work: w, limit: 0}
	if r.isOverLimit() {
		t.Error("expected limit=0 to not be over limit")
	}
	r.Send(&agentrun.Response{Step: &agentrun.ToolFinishedStep{}})
	if r.completed != 1 || r.sentCount != 1 || r.isOverLimit() {
		t.Errorf("after ToolFinishedStep: completed=%d, sentCount=%d, isOverLimit=%v", r.completed, r.sentCount, r.isOverLimit())
	}
	r.limit = 2
	if r.isOverLimit() {
		t.Error("expected completed=1 < limit=2 to not be over limit")
	}
	r.CountStep()
	if !r.isOverLimit() {
		t.Error("expected completed=2 == limit=2 to be over limit")
	}

	stoppedWork := agentrun.NewWork("c", 0, func() error { return nil })
	if err := stoppedWork.Stop(); err != nil {
		t.Fatal(err)
	}
	stoppedReader := &reader{work: stoppedWork}
	stoppedReader.Send(&agentrun.Response{Step: &agentrun.ToolFinishedStep{}})
	if stoppedReader.sentCount != 0 || stoppedReader.completed != 0 {
		t.Errorf("Send on stopped work: sentCount=%d, completed=%d", stoppedReader.sentCount, stoppedReader.completed)
	}

	// configureCommand with pre-set Env vs empty Env and Token vs empty Token.
	cmd := exec.CommandContext(t.Context(), "true")
	cmd.Env = []string{"CUSTOM=1"}
	configureCommand(cmd, spec{Root: "/tmp/r", Token: ""})
	if cmd.Dir != "/tmp/r" || !slices.Equal(cmd.Env, []string{"CUSTOM=1"}) {
		t.Errorf("configureCommand custom Env: Dir=%q, Env=%v", cmd.Dir, cmd.Env)
	}
	cmd2 := exec.CommandContext(t.Context(), "true")
	configureCommand(cmd2, spec{Root: "", Token: "tok-x"})
	if cmd2.Dir != "" || !slices.Contains(cmd2.Env, "NUMEN_TOKEN=tok-x") {
		t.Errorf("configureCommand default Env with token: Dir=%q", cmd2.Dir)
	}

	// makeAntigravityCommand and makeAnswerCommand with defaultModelName vs custom model, session, ToolsURL, and customPath.
	var reg agentrun.Registry
	reg.SetSession("conv-a", "sess-a")
	shPath, err := exec.LookPath("sh")
	if err == nil {
		cmdWithTools, cleanTools, err := makeAntigravityCommand(t.Context(), &agentrun.Request{
			Prompt: "p", Tools: []string{"t"}, Model: defaultModelName, Conversation: "conv-a",
			ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "tk",
		}, &reg, shPath)
		if err != nil || cleanTools == nil || cmdWithTools.Path != shPath {
			t.Fatalf("makeAntigravityCommand with ToolsURL: Path=%q, err=%v", cmdWithTools.Path, err)
		}
		if slices.Contains(cmdWithTools.Args, "--model") || !slices.Contains(cmdWithTools.Args, "--conversation") {
			t.Errorf("makeAntigravityCommand Args = %v", cmdWithTools.Args)
		}
		if err := cleanTools(); err != nil {
			t.Fatal(err)
		}

		ansCmd, cleanAns, err := makeAnswerCommand(t.Context(), &agentrun.Request{
			Prompt: "p", Model: "m-ans", Conversation: "conv-a",
		}, &reg, shPath)
		if err != nil || cleanAns == nil || ansCmd.Path != shPath {
			t.Fatalf("makeAnswerCommand: Path=%q, err=%v", ansCmd.Path, err)
		}
		if !slices.Contains(ansCmd.Args, "--model") || !slices.Contains(ansCmd.Args, "m-ans") || !slices.Contains(ansCmd.Args, "--conversation") {
			t.Errorf("makeAnswerCommand Args = %v", ansCmd.Args)
		}
		if err := cleanAns(); err != nil {
			t.Fatal(err)
		}

		ansDefaultCmd, cleanAnsDef, err := makeAnswerCommand(t.Context(), &agentrun.Request{
			Prompt: "p", Model: defaultModelName,
		}, &reg, shPath)
		if err != nil || cleanAnsDef == nil || slices.Contains(ansDefaultCmd.Args, "--model") {
			t.Errorf("makeAnswerCommand with defaultModelName Args = %v, err = %v", ansDefaultCmd.Args, err)
		}
		if cleanAnsDef != nil {
			if err := cleanAnsDef(); err != nil {
				t.Error(err)
			}
		}
	}

	// start error paths: StdoutPipe error, cmd.Start error, closed registry, and cleanup error in runReader.
	cleanedStdout := false
	if _, err := start(t.Context(), &reg, spec{
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			c := exec.CommandContext(ctx, "true")
			c.Stdout = &bytes.Buffer{}
			return c, func() error {
				cleanedStdout = true
				return nil
			}, nil
		},
	}); err == nil || !cleanedStdout {
		t.Errorf("start with pre-set Stdout = %v, cleaned=%v", err, cleanedStdout)
	}

	cleanedStart := false
	if _, err := start(t.Context(), &reg, spec{
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return exec.CommandContext(ctx, "/nonexistent-agy-bin"), func() error {
				cleanedStart = true
				return nil
			}, nil
		},
	}); err == nil || !cleanedStart {
		t.Errorf("start with nonexistent binary = %v, cleaned=%v", err, cleanedStart)
	}

	var closedReg agentrun.Registry
	if err := closedReg.Close(); err != nil {
		t.Fatal(err)
	}
	cleanedClosed := false
	if _, err := start(t.Context(), &closedReg, spec{
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return exec.CommandContext(ctx, "true"), func() error {
				cleanedClosed = true
				return nil
			}, nil
		},
	}); err == nil || !cleanedClosed {
		t.Errorf("start with closed registry = %v, cleaned=%v", err, cleanedClosed)
	}

	var customStderr bytes.Buffer
	wCleanErr, err := start(t.Context(), &reg, spec{
		NewHandler: newEventHandler,
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			c := exec.CommandContext(ctx, "/bin/sh", "-c", "echo plain-line; echo custom-stderr >&2")
			c.Stderr = &customStderr
			return c, func() error {
				return errors.New("cleanup failed")
			}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var cleanSteps []*agentrun.Response
	for resp := range wCleanErr.Output {
		cleanSteps = append(cleanSteps, resp)
	}
	if len(cleanSteps) != 2 || cleanSteps[0].GetText() != "plain-line\n" || !strings.Contains(cleanSteps[1].GetStopped(), "release the resources of the run: cleanup failed") {
		t.Errorf("cleanSteps = %+v", cleanSteps)
	}
	if !strings.Contains(customStderr.String(), "custom-stderr") {
		t.Errorf("customStderr = %q", customStderr.String())
	}

	// scanOutput when work is stopped mid-stream returns false.
	stopWork := agentrun.NewWork("c", 10, func() error { return nil })
	stopReader := &reader{work: stopWork}
	ended := scanOutput(bufio.NewScanner(strings.NewReader("{\"k\":1}\nplain\n")), func(map[string]any) {
		if err := stopWork.Stop(); err != nil {
			t.Error(err)
		}
	}, stopReader, stopWork, func() {})
	if ended || stopReader.sentCount != 0 {
		t.Errorf("scanOutput stopped mid-stream: ended=%v, sentCount=%d", ended, stopReader.sentCount)
	}

	// reportExit cases: scanErr, waitErr, sentCount==0 with stderr, sentCount>0 with stderr, sentCount==0 with whitespace stderr.
	checkReportExit := func(sentCount int, scanErr, waitErr error, stderr string) string {
		rw := agentrun.NewWork("c", 4, func() error { return nil })
		rr := &reader{work: rw, sentCount: sentCount}
		reportExit(rr, scanErr, waitErr, stderr)
		close(rw.Output)
		for resp := range rw.Output {
			return resp.GetStopped()
		}
		return ""
	}
	if got := checkReportExit(0, errors.New("scan err"), errors.New("wait err"), "stderr"); got != "scan err" {
		t.Errorf("reportExit scanErr = %q", got)
	}
	if got := checkReportExit(0, nil, errors.New("wait err"), "stderr msg"); got != "stderr msg" {
		t.Errorf("reportExit waitErr = %q", got)
	}
	if got := checkReportExit(0, nil, nil, "only stderr"); got != "only stderr" {
		t.Errorf("reportExit sentCount=0 stderr = %q", got)
	}
	if got := checkReportExit(1, nil, nil, "only stderr"); got != "" {
		t.Errorf("reportExit sentCount=1 stderr = %q, want empty", got)
	}
	if got := checkReportExit(0, nil, nil, "   "); got != "" {
		t.Errorf("reportExit blank stderr = %q, want empty", got)
	}
}
