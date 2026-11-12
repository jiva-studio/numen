//go:build unix

package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func runTestProgram(ctx context.Context, command []string, arguments ...string) ([]byte, error) {
	out, combined, err := (Runner{}).Run(ctx, command, nil, nil, arguments...)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, errors.New(combined)
	}
	return out, nil
}

func waitFor(t *testing.T, what string, holds func() bool) {
	t.Helper()
	const attempts = 3000
	for range attempts {
		if holds() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s did not happen", what)
}

func isAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func readPID(path string) (int, bool) {
	raw, err := os.ReadFile(path)
	if err != nil || !strings.HasSuffix(string(raw), "\n") {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	return pid, err == nil
}

func spawningScript(t *testing.T) []string {
	t.Helper()
	pos := filepath.Join(t.TempDir(), "tool")
	body := "#!/bin/sh\nsleep 600 >/dev/null 2>&1 &\necho $! > \"$1/child.tmp\"\n" +
		"mv \"$1/child.tmp\" \"$1/child\"\nwait\n"
	if err := os.WriteFile(pos, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return []string{pos}
}

// Canceling a run ends the process the tool started as well as the tool, and
// the call returns without waiting for either to finish by itself.
func TestCancelingARunEndsEveryProcessTheToolStarted(t *testing.T) {
	folder := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	returned := make(chan error, 1)
	go func() {
		_, err := runTestProgram(ctx, spawningScript(t), folder)
		returned <- err
	}()

	var child int
	waitFor(t, "the tool starting its child", func() bool {
		pid, ok := readPID(filepath.Join(folder, "child"))
		child = pid
		return ok
	})
	if !isAlive(child) {
		t.Fatal("the child was not running before the cancel")
	}
	cancel()

	select {
	case err := <-returned:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("a canceled run answered %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("a canceled run did not return")
	}
	waitFor(t, "the child ending", func() bool { return !isAlive(child) })
}

// A folder the tool's child was writing in can be removed once the run is
// canceled.
func TestTheFolderOfACanceledRunCanBeRemoved(t *testing.T) {
	folder := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	isDone := make(chan struct{})
	go func() {
		defer close(isDone)
		if _, err := runTestProgram(ctx, spawningScript(t), folder); err == nil {
			t.Error("a canceled run reported no error")
		}
	}()
	waitFor(t, "the tool starting its child", func() bool {
		_, ok := readPID(filepath.Join(folder, "child"))
		return ok
	})
	cancel()
	<-isDone
	if err := os.RemoveAll(folder); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(folder); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the folder is still there: %v", err)
	}
}

// A run that ends by itself is unchanged: its output is returned, and a failure
// carries the last line the tool printed.
func TestARunThatEndsByItselfAnswersAsBefore(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good")
	bad := filepath.Join(dir, "bad")
	if err := os.WriteFile(good, []byte("#!/bin/sh\necho fine\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("#!/bin/sh\necho no good >&2\nexit 3\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	out, err := runTestProgram(t.Context(), []string{good})
	if err != nil || strings.TrimSpace(string(out)) != "fine" {
		t.Errorf("a run that succeeded answered %q, %v", out, err)
	}
	_, err = runTestProgram(t.Context(), []string{bad})
	if err == nil || !strings.Contains(err.Error(), "no good") {
		t.Errorf("a run that failed answered %v", err)
	}
}

// A process group gives the command a wait delay of stopGrace.
func TestSetProcessGroupSetsWaitDelay(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "true")
	setProcessGroup(cmd)
	if cmd.WaitDelay != 5*time.Second {
		t.Errorf("WaitDelay is %v, want %v", cmd.WaitDelay, 5*time.Second)
	}
}

// Canceling a process that has already finished reports that it is done.
func TestSetProcessGroup_CancelFinishedProcess(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "true")
	setProcessGroup(cmd)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	err := cmd.Cancel()
	if !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("canceling finished process answered %v, want %v", err, os.ErrProcessDone)
	}
}
