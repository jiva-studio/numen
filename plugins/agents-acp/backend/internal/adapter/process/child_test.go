package process

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func findProgram(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func launchChild(t *testing.T) *Child {
	t.Helper()
	launched, err := Launcher{}.Launch(t.Context(), "echo", []string{findProgram(t, "cat")})
	if err != nil {
		t.Fatal(err)
	}
	child, ok := launched.(*Child)
	if !ok {
		t.Fatalf("launched %T", launched)
	}
	return child
}

func TestTheLaunchedProgramIsSupervisedInAFolderOfItsOwn(t *testing.T) {
	if maxLineBytes != 4*1024*1024 || stderrTailBytes != 8*1024 {
		t.Fatalf("buffer constants = (%d, %d), want (4194304, 8192)", maxLineBytes, stderrTailBytes)
	}
	t.Setenv("NUMEN_PLUGIN_TOKEN", "secret-token")
	child := launchChild(t)

	if child.cmd.Dir != child.Dir() || child.cmd.Cancel == nil || child.cmd.WaitDelay != 2*time.Second {
		t.Errorf("not supervised or in another folder: dir=%q want=%q waitDelay=%v", child.cmd.Dir, child.Dir(), child.cmd.WaitDelay)
	}
	if len(child.cmd.Env) == 0 || slices.Contains(child.cmd.Env, "NUMEN_PLUGIN_TOKEN=secret-token") {
		t.Errorf("child env = %v, want filtered child environment", child.cmd.Env)
	}
	if child.cmd.Stderr != child.stderr {
		t.Error("child stderr was not attached to tailBuffer")
	}
	if err := child.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
	if child.cmd.ProcessState == nil {
		t.Error("process was not waited for on Close")
	}
}

func TestTheLinesSentAreTheLinesReceivedUntilTheOutputEnds(t *testing.T) {
	child := launchChild(t)

	longLine := strings.Repeat("a", 70000)
	if err := child.Send([]byte(longLine)); err != nil {
		t.Fatal(err)
	}
	line, err := child.Receive()
	if err != nil || string(line) != longLine {
		t.Errorf("received len=%d, %v; want len=%d", len(line), err, len(longLine))
	}
	if err := child.stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := child.Receive(); !errors.Is(err, io.EOF) {
		t.Errorf("received after the end: %v, want %v", err, io.EOF)
	}
	if err := child.Close(); !errors.Is(err, os.ErrClosed) {
		t.Errorf("close after stdin closed = %v, want %v", err, os.ErrClosed)
	}

	tooLongScanner := bufio.NewScanner(strings.NewReader("0123456789abcdef\n"))
	tooLongScanner.Buffer(nil, 8)
	shortChild := &Child{lines: tooLongScanner}
	if _, err := shortChild.Receive(); !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("Receive on overlong line = %v, want %v", err, bufio.ErrTooLong)
	}
}

func TestNoProgramStartsWithoutACommand(t *testing.T) {
	if _, err := (Launcher{}).Launch(t.Context(), "empty", nil); err == nil || !strings.Contains(err.Error(), "no command starts empty") {
		t.Errorf("launch with no command = %v", err)
	}
}

func TestAMissingProgramFailsToStart(t *testing.T) {
	_, err := (Launcher{}).Launch(t.Context(), "missing", []string{"/nonexistent-acp-bin"})
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "start missing") {
		t.Errorf("launch of a missing binary = %v, want wrapped os.ErrNotExist", err)
	}
}

func TestAFolderThatCannotBeCreatedFailsTheLaunch(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "nonexistent-parent"))

	_, err := Launcher{}.Launch(t.Context(), "echo", []string{findProgram(t, "cat")})

	if err == nil || !strings.Contains(err.Error(), "create the session folder") || errors.Unwrap(err) == nil {
		t.Errorf("launch with an invalid TMPDIR = %v", err)
	}
}

func TestAnInputThatIsAlreadySetFailsToStart(t *testing.T) {
	child := &Child{cmd: exec.CommandContext(t.Context(), findProgram(t, "true")), program: "stdin-prog"}
	child.cmd.Stdin = strings.NewReader("")

	if err := child.start(); err == nil || !strings.Contains(err.Error(), "open the input of stdin-prog") || errors.Unwrap(err) == nil {
		t.Errorf("start with Stdin set = %v", err)
	}
}

func TestAnOutputThatIsAlreadySetFailsToStart(t *testing.T) {
	child := &Child{cmd: exec.CommandContext(t.Context(), findProgram(t, "true")), program: "stdout-prog"}
	child.cmd.Stdout = io.Discard

	if err := child.start(); err == nil || !strings.Contains(err.Error(), "open the output of stdout-prog") || errors.Unwrap(err) == nil {
		t.Errorf("start with Stdout set = %v", err)
	}
}

func TestTheTailBufferKeepsTheLastBytes(t *testing.T) {
	var tail tailBuffer
	if n, err := tail.Write([]byte("abc")); err != nil || n != 3 || tail.String() != "abc" {
		t.Fatalf("short Write = (%d, %v, %q), want (3, nil, \"abc\")", n, err, tail.String())
	}

	tailText := strings.Repeat("k", 8192)
	input := []byte("dropped-" + tailText)
	if n, err := tail.Write(input); err != nil || n != len(input) {
		t.Fatalf("Write = (%d, %v), want (%d, nil)", n, err, len(input))
	}

	if tail.String() != tailText {
		t.Errorf("tail len = %d, want %d", len(tail.String()), len(tailText))
	}
}

func TestCloseAndIsStoppedHandleExitErrorsWaitErrorsAndCancel(t *testing.T) {
	for _, err := range []error{nil, &exec.ExitError{}, context.Canceled, os.ErrProcessDone, syscall.ESRCH} {
		if !isStopped(err) {
			t.Errorf("isStopped(%v) = false, want true", err)
		}
	}
	if isStopped(errors.New("unexpected wait failure")) {
		t.Error("isStopped(unexpected) = true, want false")
	}

	falseLaunched, err := (Launcher{}).Launch(t.Context(), "false", []string{findProgram(t, "false")})
	if err != nil {
		t.Fatal(err)
	}
	if err := falseLaunched.Close(); err != nil {
		t.Errorf("Close on non-zero exit process = %v, want nil", err)
	}

	alreadyWaited := launchChild(t)
	if err := alreadyWaited.stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := alreadyWaited.cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := alreadyWaited.Close(); err == nil {
		t.Error("Close when Wait was already called succeeded, want error")
	}

	stopCtx, stop := context.WithCancel(t.Context())
	unstarted := &Child{cmd: exec.CommandContext(t.Context(), findProgram(t, "true")), stop: stop, dir: filepath.Join(t.TempDir(), "clean")}
	if err := unstarted.Close(); err != nil {
		t.Errorf("Close on unstarted child = %v, want nil", err)
	}
	if !errors.Is(stopCtx.Err(), context.Canceled) {
		t.Error("Close did not call stop()")
	}
}
