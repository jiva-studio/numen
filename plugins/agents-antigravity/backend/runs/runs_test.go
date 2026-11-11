package runs

import (
	"bufio"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/events"
)

type fakeProcess struct {
	stdout  string
	stderr  string
	waitErr error
}

func (p *fakeProcess) Output() io.ReadCloser { return io.NopCloser(strings.NewReader(p.stdout)) }
func (p *fakeProcess) Stderr() string        { return p.stderr }
func (p *fakeProcess) Wait() error           { return p.waitErr }

type launcherFunc func(ctx context.Context, cmd Command) (Process, error)

func (f launcherFunc) Start(ctx context.Context, cmd Command) (Process, error) {
	return f(ctx, cmd)
}

func TestRunnerErrorPathsAndLimits(t *testing.T) {
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
	cmd := Command{Path: "true", Env: []string{"CUSTOM=1"}}
	configureCommand(&cmd, spec{Root: "/tmp/r", Token: ""})
	if cmd.Dir != "/tmp/r" || !slices.Equal(cmd.Env, []string{"CUSTOM=1"}) || cmd.Token != "" {
		t.Errorf("configureCommand custom Env: Dir=%q, Env=%v, Token=%q", cmd.Dir, cmd.Env, cmd.Token)
	}
	cmd2 := Command{Path: "true"}
	configureCommand(&cmd2, spec{Root: "", Token: "tok-x"})
	if cmd2.Dir != "" || cmd2.Token != "tok-x" {
		t.Errorf("configureCommand with token: Dir=%q, Token=%q", cmd2.Dir, cmd2.Token)
	}

	var reg agentrun.Registry
	runner := New(&reg, nil, launcherFunc(func(_ context.Context, c Command) (Process, error) {
		if c.Path == "/nonexistent-agy-bin" {
			return nil, errors.New("no such file")
		}
		return &fakeProcess{stdout: "plain-line\n", stderr: "custom-stderr\n"}, nil
	}), nil)

	cleanedStart := false
	if _, err := runner.start(t.Context(), spec{
		Build: func(context.Context) (Command, error) {
			return Command{
				Path: "/nonexistent-agy-bin",
				Release: func() error {
					cleanedStart = true
					return nil
				},
			}, nil
		},
	}); err == nil || !cleanedStart {
		t.Errorf("start with nonexistent binary = %v, cleaned=%v", err, cleanedStart)
	}

	var closedReg agentrun.Registry
	if err := closedReg.Close(); err != nil {
		t.Fatal(err)
	}
	closedRunner := New(&closedReg, nil, launcherFunc(func(context.Context, Command) (Process, error) {
		return &fakeProcess{}, nil
	}), nil)
	cleanedClosed := false
	if _, err := closedRunner.start(t.Context(), spec{
		Build: func(context.Context) (Command, error) {
			return Command{
				Path: "true",
				Release: func() error {
					cleanedClosed = true
					return nil
				},
			}, nil
		},
	}); err == nil || !cleanedClosed {
		t.Errorf("start with closed registry = %v, cleaned=%v", err, cleanedClosed)
	}

	wCleanErr, err := runner.start(t.Context(), spec{
		NewHandler: events.NewHandler,
		Build: func(context.Context) (Command, error) {
			return Command{
				Path: "/bin/sh",
				Release: func() error {
					return errors.New("cleanup failed")
				},
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
