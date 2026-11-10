package run

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
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/codexcli"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/process"
)

func TestTheToolCallsOfCodexAreReported(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.started","item":{"type":"mcp_tool_call","name":"read_note","arguments":{"path":"a.md"}}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"mcp_tool_call"}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"agent_message"}}'`+"\n"+
			`echo '{"type":"turn.completed"}'`+"\n"+
			"echo plain text\n")
	var reg agentrun.Registry

	w, err := newTestRunner().Start(t.Context(), &reg, &agentrun.Request{Command: []string{script}}, "")
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

	w, err := newTestRunner().Start(t.Context(), &reg, &agentrun.Request{Command: []string{script}}, "")
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

	w, err := newTestRunner().Start(t.Context(), &reg, &agentrun.Request{Command: []string{script}}, "")
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

func TestReaderLimitsAndConfigureCommand(t *testing.T) {
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

}

func TestStartErrorPathsAndReportExitAndScanOutput(t *testing.T) {
	runner := newTestRunner()
	var reg agentrun.Registry

	// 1. Build returns error.
	buildErr := errors.New("build failed")
	if _, err := runner.start(t.Context(), &reg, spec{
		Build: func(context.Context) (*exec.Cmd, func() error, error) {
			return nil, nil, buildErr
		},
	}); !errors.Is(err, buildErr) {
		t.Errorf("start with Build error = %v, want %v", err, buildErr)
	}

	// 2. StdoutPipe fails when cmd.Stdout is already set, and calls cleanup.
	cleanedStdoutErr := false
	if _, err := runner.start(t.Context(), &reg, spec{
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
	if _, err := runner.start(t.Context(), &reg, spec{
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
	if _, err := runner.start(t.Context(), &closedReg, spec{
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
	wCleanup, err := runner.start(t.Context(), &reg, spec{
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
	wLimit, err := runner.start(t.Context(), &reg, spec{
		MaxSteps: 1,
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

func newTestRunner() *Runner {
	return NewRunner(codexcli.Commands{}, process.Group{})
}
