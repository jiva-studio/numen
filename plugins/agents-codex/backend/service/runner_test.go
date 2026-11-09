package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
)

func TestResolveCommandWithEmptyCommand(t *testing.T) {
	withoutAgentsAnywhere(t)

	bin, args := resolveCommand(nil)
	if bin != "codex" || len(args) != 0 {
		t.Errorf("expected codex and empty args, got %s, %v", bin, args)
	}
}

func TestTheToolCallsOfCodexAreReported(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.started","item":{"type":"mcp_tool_call","name":"read_note","arguments":{"path":"a.md"}}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"mcp_tool_call"}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"agent_message"}}'`+"\n"+
			`echo '{"type":"turn.completed"}'`+"\n"+
			"echo plain text\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(&agentrun.Request{Command: []string{script}}))
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
	if len(got) != 3 {
		t.Fatalf("expected 3 steps, got %v", got)
	}
	if call := got[0].GetToolCall(); call.Tool != "read_note" || call.About != `{"path":"a.md"}` {
		t.Errorf("tool call is %v", got[0])
	}
	if !got[1].IsToolFinished() {
		t.Errorf("expected a finished tool, got %v", got[1])
	}
	if got[2].GetText() != "plain text\n" {
		t.Errorf("expected the plain line as text, got %v", got[2])
	}
}

func TestRunner_EmitsCodexMessages(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.completed","item":{"id":"i0","type":"agent_message","text":"The answer."}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"agent_message","text":""}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"reasoning","text":"hidden"}}'`+"\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(&agentrun.Request{Command: []string{script}}))
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 1 || got[0].GetText() != "The answer." {
		t.Errorf("got %v", got)
	}
}

func TestACodexToolCallNamesItsToolByTheToolField(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.started","item":{"type":"mcp_tool_call","server":"numen","tool":"search_notes","arguments":{"q":"x"}}}'`+"\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(&agentrun.Request{Command: []string{script}}))
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 1 || got[0].GetToolCall().Tool != "search_notes" {
		t.Errorf("got %v", got)
	}
}

func TestEventsAndJsonFieldEdgeCases(t *testing.T) {
	if got := getString(map[string]any{"k": 123}, "k"); got != "" {
		t.Errorf("getString non-string = %q, want empty", got)
	}
	if got := getMap(map[string]any{"k": "not-map"}, "k"); got != nil {
		t.Errorf("getMap non-map = %v, want nil", got)
	}
	if got := getToolArguments(map[string]any{"arguments": make(chan int)}); got != "" {
		t.Errorf("getToolArguments channel = %q, want empty", got)
	}

	w := agentrun.NewWork("c", 10, func() error { return nil })
	r := &reader{work: w}
	handle := newEventHandler(r)

	// Ignored events: nil item, wrong evType, or wrong item type.
	handle(map[string]any{"type": "item.completed"})
	handle(map[string]any{"type": "item.started", "item": map[string]any{"type": "agent_message", "text": "no"}})
	handle(map[string]any{"type": "other", "item": map[string]any{"type": "mcp_tool_call", "tool": "t"}})
	handle(map[string]any{"type": "item.started", "item": map[string]any{"type": "other"}})
	handle(map[string]any{"type": "item.completed", "item": map[string]any{"type": "other"}})
	if r.sentCount != 0 {
		t.Fatalf("expected 0 sent messages for ignored events, got %d", r.sentCount)
	}
}

func TestReaderLimitsAndConfigureCommandAndSpecs(t *testing.T) {
	wantDisabled := []string{
		"shell_tool", "unified_exec", "view_image", "multi_agent", "apps", "browser_use",
		"computer_use", "plugins", "image_generation", "hooks",
	}
	if got := getDisabledFeatures(); !slices.Equal(got, wantDisabled) {
		t.Errorf("getDisabledFeatures = %v, want %v", got, wantDisabled)
	}

	r := &reader{limit: 0, completed: 0}
	if r.isOverLimit() {
		t.Error("expected limit=0 completed=0 to not be over limit")
	}
	r.CountStep()
	if r.completed != 1 || r.isOverLimit() {
		t.Errorf("after CountStep with limit=0: completed=%d, isOverLimit=%v", r.completed, r.isOverLimit())
	}
	r.limit = 2
	if r.isOverLimit() {
		t.Error("expected completed=1 < limit=2 to not be over limit")
	}
	r.CountStep()
	if !r.isOverLimit() {
		t.Error("expected completed=2 == limit=2 to be over limit")
	}

	// Send on already-stopped Work returns immediately without incrementing sentCount or completed.
	stoppedWork := agentrun.NewWork("c", 0, func() error { return nil })
	if err := stoppedWork.Stop(); err != nil {
		t.Fatal(err)
	}
	stoppedReader := &reader{work: stoppedWork}
	stoppedReader.Send(&agentrun.Response{Step: &agentrun.ToolFinishedStep{}})
	if stoppedReader.sentCount != 0 || stoppedReader.completed != 0 {
		t.Errorf("Send on stopped work: sentCount=%d, completed=%d", stoppedReader.sentCount, stoppedReader.completed)
	}

	// configureCommand preserves non-empty Env and only appends NUMEN_TOKEN when Token is non-empty.
	cmd := exec.CommandContext(t.Context(), "true")
	cmd.Env = []string{"CUSTOM_ENV=1"}
	configureCommand(cmd, spec{Root: "/tmp/root", Token: ""})
	if cmd.Dir != "/tmp/root" || !slices.Equal(cmd.Env, []string{"CUSTOM_ENV=1"}) {
		t.Errorf("configureCommand with custom Env and empty Token: Dir=%q, Env=%v", cmd.Dir, cmd.Env)
	}
	cmd2 := exec.CommandContext(t.Context(), "true")
	configureCommand(cmd2, spec{Root: "", Token: "tok-1"})
	if cmd2.Dir != "" || !slices.Contains(cmd2.Env, "NUMEN_TOKEN=tok-1") {
		t.Errorf("configureCommand with default Env and Token: Dir=%q, hasToken=%v", cmd2.Dir, slices.Contains(cmd2.Env, "NUMEN_TOKEN=tok-1"))
	}

	// appendModelArguments omits --model when Model is "default", and makeCodexCommand omits -c when ToolsURL is empty.
	if got := appendModelArguments(nil, &agentrun.Request{Model: "default"}); len(got) != 0 {
		t.Errorf("appendModelArguments(default) = %v, want empty", got)
	}
	codexCmd := makeCodexCommand(t.Context(), &agentrun.Request{Prompt: "hi"}, "/custom/bin/codex")
	if sliceContainsSubstr(codexCmd.Args, "mcp_servers") {
		t.Errorf("makeCodexCommand without ToolsURL included mcp_servers: %v", codexCmd.Args)
	}

	// newSpec with customPath passes customPath through when runnable.
	shPath, err := exec.LookPath("sh")
	if err == nil {
		sWithTools := newSpec(&agentrun.Request{Prompt: "p", Tools: []string{"t"}, Root: "/r", ToolsToken: "tk"}, shPath)
		builtCmd, cleanup, err := sWithTools.Build(t.Context())
		if err != nil || cleanup != nil || builtCmd.Path != shPath {
			t.Errorf("newSpec with tools and customPath: Path=%q, hasCleanup=%v, err=%v", builtCmd.Path, cleanup != nil, err)
		}
		sNoTools := newSpec(&agentrun.Request{Prompt: "p"}, shPath)
		builtAnsCmd, ansCleanup, err := sNoTools.Build(t.Context())
		if err != nil || ansCleanup == nil || builtAnsCmd.Path != shPath {
			t.Errorf("newSpec without tools and customPath: Path=%q, err=%v", builtAnsCmd.Path, err)
		}
		if ansCleanup != nil {
			if err := ansCleanup(); err != nil {
				t.Error(err)
			}
		}
	}

	// makeAnswerCommand fails when TMPDIR is not a valid directory.
	t.Setenv("TMPDIR", "/dev/null/nonexistent")
	if _, _, err := makeAnswerCommand(t.Context(), &agentrun.Request{Prompt: "p"}); err == nil {
		t.Error("expected makeAnswerCommand error when TMPDIR is invalid")
	}
}

func sliceContainsSubstr(items []string, sub string) bool {
	for _, item := range items {
		if strings.Contains(item, sub) {
			return true
		}
	}
	return false
}

func TestStartErrorPathsAndReportExitAndScanOutput(t *testing.T) {
	var reg agentrun.Registry

	// 1. Build returns error.
	buildErr := errors.New("build failed")
	if _, err := start(t.Context(), &reg, spec{
		Build: func(context.Context) (*exec.Cmd, func() error, error) {
			return nil, nil, buildErr
		},
	}); !errors.Is(err, buildErr) {
		t.Errorf("start with Build error = %v, want %v", err, buildErr)
	}

	// 2. StdoutPipe fails when cmd.Stdout is already set, and calls cleanup.
	cleanedStdoutErr := false
	if _, err := start(t.Context(), &reg, spec{
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			c := exec.CommandContext(ctx, "true")
			c.Stdout = &bytes.Buffer{}
			return c, func() error {
				cleanedStdoutErr = true
				return nil
			}, nil
		},
	}); err == nil || !cleanedStdoutErr {
		t.Errorf("start with pre-set Stdout = %v, cleaned=%v", err, cleanedStdoutErr)
	}

	// 3. cmd.Start fails on nonexistent binary and calls cleanup.
	cleanedStartErr := false
	if _, err := start(t.Context(), &reg, spec{
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return exec.CommandContext(ctx, "/nonexistent-codex-bin"), func() error {
				cleanedStartErr = true
				return nil
			}, nil
		},
	}); err == nil || !cleanedStartErr {
		t.Errorf("start with nonexistent binary = %v, cleaned=%v", err, cleanedStartErr)
	}

	// 4. Closed registry rejects Add and calls cleanup.
	var closedReg agentrun.Registry
	if err := closedReg.Close(); err != nil {
		t.Fatal(err)
	}
	cleanedClosedReg := false
	if _, err := start(t.Context(), &closedReg, spec{
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return exec.CommandContext(ctx, "true"), func() error {
				cleanedClosedReg = true
				return nil
			}, nil
		},
	}); err == nil || !cleanedClosedReg {
		t.Errorf("start with closed registry = %v, cleaned=%v", err, cleanedClosedReg)
	}

	// 5. runReader reports cleanup error, and preserves pre-set cmd.Stderr.
	var customStderr bytes.Buffer
	wCleanup, err := start(t.Context(), &reg, spec{
		NewHandler: newEventHandler,
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			c := exec.CommandContext(ctx, "/bin/sh", "-c", "echo custom-err >&2")
			c.Stderr = &customStderr
			return c, func() error {
				return errors.New("cleanup boom")
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("start for cleanup error: %v", err)
	}
	var cleanupSteps []*agentrun.Response
	for resp := range wCleanup.Output {
		cleanupSteps = append(cleanupSteps, resp)
	}
	if len(cleanupSteps) != 1 || !strings.Contains(cleanupSteps[0].GetStopped(), "release the resources of the run: cleanup boom") {
		t.Errorf("cleanup error steps = %+v", cleanupSteps)
	}
	if !strings.Contains(customStderr.String(), "custom-err") {
		t.Errorf("expected customStderr to receive stderr, got %q", customStderr.String())
	}

	// 6. Step limit in scanOutput stops the run with limit message and skips reportExit.
	wLimit, err := start(t.Context(), &reg, spec{
		MaxSteps:   1,
		NewHandler: newEventHandler,
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return exec.CommandContext(ctx, "/bin/sh", "-c",
				`echo '{"type":"item.completed","item":{"type":"mcp_tool_call"}}'; echo 'after-limit' >&2; exit 2`), nil, nil
		},
	})
	if err != nil {
		t.Fatalf("start for limit: %v", err)
	}
	var limitSteps []*agentrun.Response
	for resp := range wLimit.Output {
		limitSteps = append(limitSteps, resp)
	}
	if len(limitSteps) != 2 || limitSteps[1].GetStopped() != "reached maximum limit of 1 steps" {
		t.Errorf("limit steps = %+v, want [ToolFinished, reached maximum limit of 1 steps]", limitSteps)
	}

	// 7. scanOutput returns false when work is stopped after a line.
	stopWork := agentrun.NewWork("c", 10, func() error { return nil })
	stopReader := &reader{work: stopWork}
	scanner := bufio.NewScanner(strings.NewReader("line1\nline2\n"))
	limited := scanOutput(scanner, func(map[string]any) {}, stopReader, stopWork, func() {})
	if limited || stopReader.sentCount != 2 {
		t.Errorf("scanOutput normal: limited=%v, sentCount=%d", limited, stopReader.sentCount)
	}
	scannerStopped := bufio.NewScanner(strings.NewReader("{\"k\":1}\nline2\n"))
	stopWork2 := agentrun.NewWork("c", 10, func() error { return nil })
	stopReader2 := &reader{work: stopWork2}
	limited2 := scanOutput(scannerStopped, func(map[string]any) {
		if err := stopWork2.Stop(); err != nil {
			t.Error(err)
		}
	}, stopReader2, stopWork2, func() {})
	if limited2 || stopReader2.sentCount != 0 {
		t.Errorf("scanOutput stopped mid-stream: limited=%v, sentCount=%d", limited2, stopReader2.sentCount)
	}

	// 8. reportExit cases: scanErr, waitErr, sentCount==0 with stderr, sentCount>0 with stderr, and sentCount==0 with whitespace stderr.
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
		t.Errorf("reportExit scanErr = %q, want 'scan err'", got)
	}
	if got := checkReportExit(0, nil, errors.New("wait err"), "stderr msg"); got != "stderr msg" {
		t.Errorf("reportExit waitErr = %q, want 'stderr msg'", got)
	}
	if got := checkReportExit(0, nil, nil, "only stderr"); got != "only stderr" {
		t.Errorf("reportExit sentCount=0 with stderr = %q, want 'only stderr'", got)
	}
	if got := checkReportExit(1, nil, nil, "only stderr"); got != "" {
		t.Errorf("reportExit sentCount=1 with stderr = %q, want empty", got)
	}
	if got := checkReportExit(0, nil, nil, "   "); got != "" {
		t.Errorf("reportExit sentCount=0 with blank stderr = %q, want empty", got)
	}
}
