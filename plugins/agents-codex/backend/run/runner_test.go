package run

import (
	"bufio"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

type fakeCommands struct{}

func (fakeCommands) MakeToolsCommand(_ context.Context, req *agentrun.Request, _ string) Command {
	return Command{Path: req.Command[0], Args: req.Command[1:]}
}

func (fakeCommands) MakeAnswerCommand(_ context.Context, req *agentrun.Request, _ string) (Command, error) {
	return Command{Path: req.Command[0], Args: req.Command[1:]}, nil
}

type fakeProcess struct {
	stdout  io.ReadCloser
	stderr  string
	waitErr error
}

func (p *fakeProcess) Output() io.ReadCloser { return p.stdout }
func (p *fakeProcess) Stderr() string        { return p.stderr }
func (p *fakeProcess) Wait() error           { return p.waitErr }

type fakeLauncher struct{}

func (fakeLauncher) Start(_ context.Context, cmd Command) (Process, error) {
	if cmd.Path == "/nonexistent-codex-bin" {
		return nil, errors.New("binary not found")
	}
	if len(cmd.Args) == 2 && cmd.Args[0] == "-c" && strings.Contains(cmd.Args[1], "after-limit") {
		return &fakeProcess{
			stdout:  io.NopCloser(strings.NewReader(`{"type":"item.completed","item":{"type":"mcp_tool_call"}}` + "\n")),
			stderr:  "after-limit\n",
			waitErr: errors.New("exit 2"),
		}, nil
	}
	return &fakeProcess{stdout: io.NopCloser(strings.NewReader(""))}, nil
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

	// configureCommand preserves non-empty Env and sets Token when Token is non-empty.
	cmd := Command{Path: "true", Env: []string{"CUSTOM_ENV=1"}}
	configureCommand(&cmd, spec{Root: "/tmp/root", Token: ""})
	if cmd.Dir != "/tmp/root" || !slices.Equal(cmd.Env, []string{"CUSTOM_ENV=1"}) || cmd.Token != "" {
		t.Errorf("configureCommand with custom Env and empty Token: Dir=%q, Env=%v, Token=%q", cmd.Dir, cmd.Env, cmd.Token)
	}
	cmd2 := Command{Path: "true"}
	configureCommand(&cmd2, spec{Root: "", Token: "tok-1"})
	if cmd2.Dir != "" || cmd2.Token != "tok-1" {
		t.Errorf("configureCommand with Token: Dir=%q, Token=%q", cmd2.Dir, cmd2.Token)
	}
}

func TestStartErrorPathsAndReportExitAndScanOutput(t *testing.T) {
	runner := NewRunner(fakeCommands{}, fakeLauncher{})
	var reg agentrun.Registry

	// 1. Build returns error.
	buildErr := errors.New("build failed")
	if _, err := runner.start(t.Context(), &reg, spec{
		Build: func(context.Context) (Command, error) {
			return Command{}, buildErr
		},
	}); !errors.Is(err, buildErr) {
		t.Errorf("start with Build error = %v, want %v", err, buildErr)
	}

	// 2. Launcher fails on nonexistent binary and calls cleanup.
	cleanedStartErr := false
	if _, err := runner.start(t.Context(), &reg, spec{
		Build: func(context.Context) (Command, error) {
			return Command{
				Path: "/nonexistent-codex-bin",
				Release: func() error {
					cleanedStartErr = true
					return nil
				},
			}, nil
		},
	}); err == nil || !cleanedStartErr {
		t.Errorf("start with nonexistent binary = %v, cleaned=%v", err, cleanedStartErr)
	}

	// 3. Closed registry rejects Add and calls cleanup.
	var closedReg agentrun.Registry
	if err := closedReg.Close(); err != nil {
		t.Fatal(err)
	}
	cleanedClosedReg := false
	if _, err := runner.start(t.Context(), &closedReg, spec{
		Build: func(context.Context) (Command, error) {
			return Command{
				Path: "true",
				Release: func() error {
					cleanedClosedReg = true
					return nil
				},
			}, nil
		},
	}); err == nil || !cleanedClosedReg {
		t.Errorf("start with closed registry = %v, cleaned=%v", err, cleanedClosedReg)
	}

	// 4. runReader reports cleanup error.
	wCleanup, err := runner.start(t.Context(), &reg, spec{
		Build: func(context.Context) (Command, error) {
			return Command{
				Path: "/bin/sh",
				Args: []string{"-c", "echo custom-err >&2"},
				Release: func() error {
					return errors.New("cleanup boom")
				},
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

	// 5. Step limit in scanOutput stops the run with limit message and skips reportExit.
	wLimit, err := runner.start(t.Context(), &reg, spec{
		MaxSteps: 1,
		Build: func(context.Context) (Command, error) {
			return Command{
				Path: "/bin/sh",
				Args: []string{"-c", `echo '{"type":"item.completed","item":{"type":"mcp_tool_call"}}'; echo 'after-limit' >&2; exit 2`},
			}, nil
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
