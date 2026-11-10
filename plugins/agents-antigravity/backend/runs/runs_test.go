package runs

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
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/events"
)

type noSupervisor struct{}

func (noSupervisor) Supervise(*exec.Cmd) {}

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

	var reg agentrun.Registry
	runner := New(&reg, nil, noSupervisor{}, nil)

	// start error paths: StdoutPipe error, cmd.Start error, closed registry, and cleanup error in runReader.
	cleanedStdout := false
	if _, err := runner.start(t.Context(), spec{
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
	if _, err := runner.start(t.Context(), spec{
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
	closedRunner := New(&closedReg, nil, noSupervisor{}, nil)
	cleanedClosed := false
	if _, err := closedRunner.start(t.Context(), spec{
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
	wCleanErr, err := runner.start(t.Context(), spec{
		NewHandler: events.NewHandler,
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
