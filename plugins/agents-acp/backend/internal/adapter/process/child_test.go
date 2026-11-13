package process

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	child := launchChild(t)

	if child.cmd.Dir != child.Dir() || child.cmd.Cancel == nil || child.cmd.WaitDelay <= 0 {
		t.Errorf("not supervised or in another folder: dir=%q want=%q waitDelay=%v", child.cmd.Dir, child.Dir(), child.cmd.WaitDelay)
	}
	if err := child.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}

func TestTheLinesSentAreTheLinesReceivedUntilTheOutputEnds(t *testing.T) {
	child := launchChild(t)

	if err := child.Send([]byte("first")); err != nil {
		t.Fatal(err)
	}
	line, err := child.Receive()
	if err != nil || string(line) != "first" {
		t.Errorf("received %q, %v", line, err)
	}
	if err := child.stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := child.Receive(); !errors.Is(err, io.EOF) {
		t.Errorf("received after the end: %v, want %v", err, io.EOF)
	}
	if err := child.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		t.Errorf("close: %v", err)
	}
}

func TestNoProgramStartsWithoutACommand(t *testing.T) {
	if _, err := (Launcher{}).Launch(t.Context(), "empty", nil); err == nil || !strings.Contains(err.Error(), "no command starts empty") {
		t.Errorf("launch with no command = %v", err)
	}
}

func TestAMissingProgramFailsToStart(t *testing.T) {
	if _, err := (Launcher{}).Launch(t.Context(), "missing", []string{"/nonexistent-acp-bin"}); err == nil || !strings.Contains(err.Error(), "start missing") {
		t.Errorf("launch of a missing binary = %v", err)
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
	tailText := strings.Repeat("k", stderrTailBytes)

	if _, err := tail.Write([]byte("dropped-" + tailText)); err != nil {
		t.Fatal(err)
	}

	if tail.String() != tailText {
		t.Errorf("tail len = %d, want %d", len(tail.String()), len(tailText))
	}
}
