package completion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunCommandReturnsStdoutFedByStdin(t *testing.T) {
	got, err := RunCommand(context.Background(), writeScript(t, `cat`), nil, os.Environ(), "hello\n")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestRunCommandReportsTheLastLineOfStderr(t *testing.T) {
	_, err := RunCommand(context.Background(), writeScript(t, "echo warming >&2\necho boom >&2\nexit 3"), nil, os.Environ(), "")
	if err == nil || err.Error() != "boom" {
		t.Errorf("got %v", err)
	}
}

func TestRunCommandStopsWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := RunCommand(ctx, writeScript(t, "exec sleep 30"), nil, os.Environ(), "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v", err)
	}
}

func TestRunCommandRefusesNoCommand(t *testing.T) {
	if _, err := RunCommand(context.Background(), "", nil, nil, ""); !errors.Is(err, ErrNoCommand) {
		t.Errorf("got %v", err)
	}
}

func TestFailureMessageFallsBackToTheError(t *testing.T) {
	if got := FailureMessage(errors.New("exit 1"), "warming up\n\nboom: no key\n"); got != "boom: no key" {
		t.Errorf("got %q", got)
	}
	if got := FailureMessage(errors.New("exit 1"), "  \n"); got != "exit 1" {
		t.Errorf("got %q", got)
	}
}

func TestClaudeArgumentsGiveNoToolsAndOneTurn(t *testing.T) {
	got := strings.Join(ClaudeArguments("be brief", "m1"), " ")
	for _, want := range []string{"-p", "--output-format text", "--max-turns 1", "--tools  ", "--strict-mcp-config", "--append-system-prompt be brief", "--model m1"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	bare := strings.Join(ClaudeArguments("", ""), " ")
	if strings.Contains(bare, "--append-system-prompt") || strings.Contains(bare, "--model") {
		t.Errorf("got %q", bare)
	}
}

func TestRunCommandStopsACommandThatWritesTooMuch(t *testing.T) {
	for _, stream := range []string{"", ">&2"} {
		script := writeScript(t, "yes | head -c 5000000 "+stream)
		_, err := RunCommand(context.Background(), script, nil, os.Environ(), "")
		if !errors.Is(err, ErrOutputTooLarge) {
			t.Errorf("stream %q: got %v", stream, err)
		}
	}
}
