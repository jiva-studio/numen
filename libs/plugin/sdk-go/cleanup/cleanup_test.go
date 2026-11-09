package cleanup

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// recorder counts calls to Errorf, Helper, and Cleanup.
type recorder struct {
	testing.TB
	failures int
	helpers  int
	cleanups []func()
}

func (r *recorder) Errorf(string, ...any) { r.failures++ }
func (r *recorder) Helper()               { r.helpers++ }
func (r *recorder) Cleanup(fn func())     { r.cleanups = append(r.cleanups, fn) }

type closer struct{ err error }

func (c closer) Close() error { return c.err }

func TestClose_ReportsTheError(t *testing.T) {
	r := &recorder{TB: t}
	Close(r, closer{err: errors.New("busy")})
	if r.failures != 1 || r.helpers != 1 {
		t.Fatalf("a failed close reported %d failures and %d helpers, want 1 and 1", r.failures, r.helpers)
	}
}

func TestClose_AcceptsASuccessfulClose(t *testing.T) {
	r := &recorder{TB: t}
	Close(r, closer{})
	if r.failures != 0 || r.helpers != 1 {
		t.Fatalf("a successful close reported %d failures and %d helpers, want 0 and 1", r.failures, r.helpers)
	}
}

func TestCloseAfter_RegistersCleanupThatCloses(t *testing.T) {
	r := &recorder{TB: t}
	CloseAfter(r, closer{err: errors.New("later")})
	if r.helpers != 1 || len(r.cleanups) != 1 {
		t.Fatalf("got helpers=%d cleanups=%d, want 1 and 1", r.helpers, len(r.cleanups))
	}
	r.cleanups[0]()
	if r.failures != 1 || r.helpers != 2 {
		t.Fatalf("cleanup run got failures=%d helpers=%d, want 1 and 2", r.failures, r.helpers)
	}
}

func TestKill_HandlesNilExitedFailedAndRunningProcesses(t *testing.T) {
	r := &recorder{TB: t}
	Kill(r, &exec.Cmd{})
	if r.failures != 0 || r.helpers != 1 {
		t.Fatalf("nil process: got failures=%d helpers=%d", r.failures, r.helpers)
	}

	doneCmd := exec.CommandContext(t.Context(), "true")
	if err := doneCmd.Run(); err != nil {
		t.Fatal(err)
	}
	Kill(r, doneCmd)
	if r.failures != 0 || r.helpers != 2 {
		t.Fatalf("exited process: got failures=%d helpers=%d", r.failures, r.helpers)
	}

	sleepCmd := exec.CommandContext(t.Context(), "sleep", "60")
	if err := sleepCmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sleepCmd.Wait(); err == nil {
			t.Log("sleep exited cleanly")
		}
	})
	Kill(r, sleepCmd)
	if r.failures != 0 || r.helpers != 3 {
		t.Fatalf("running process: got failures=%d helpers=%d", r.failures, r.helpers)
	}

	Kill(r, &exec.Cmd{Process: &os.Process{Pid: -1}})
	if r.failures != 1 || r.helpers != 4 {
		t.Fatalf("invalid pid: got failures=%d helpers=%d, want 1 and 4", r.failures, r.helpers)
	}
}

func TestWait_IgnoresExitErrorAndReportsWaitFailure(t *testing.T) {
	r := &recorder{TB: t}
	okCmd := exec.CommandContext(t.Context(), "true")
	if err := okCmd.Start(); err != nil {
		t.Fatal(err)
	}
	Wait(r, okCmd)
	if r.failures != 0 || r.helpers != 1 {
		t.Fatalf("zero exit: got failures=%d helpers=%d", r.failures, r.helpers)
	}

	failCmd := exec.CommandContext(t.Context(), "false")
	if err := failCmd.Start(); err != nil {
		t.Fatal(err)
	}
	Wait(r, failCmd)
	if r.failures != 0 || r.helpers != 2 {
		t.Fatalf("non-zero exit: got failures=%d helpers=%d", r.failures, r.helpers)
	}

	Wait(r, exec.CommandContext(t.Context(), "true"))
	if r.failures != 1 || r.helpers != 3 {
		t.Fatalf("unstarted cmd: got failures=%d helpers=%d, want 1 and 3", r.failures, r.helpers)
	}
}
