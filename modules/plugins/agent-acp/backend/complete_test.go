package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func askCompletion(text string) *connect.Request[numenv1.CompleteRequest] {
	return connect.NewRequest(&numenv1.CompleteRequest{Question: &numenv1.CompletionQuestion{
		Messages: []*numenv1.CompletionMessage{
			{Role: "system", Text: "be brief"},
			{Role: "user", Text: text},
		},
	}})
}

func writeFakeAgent(t *testing.T, body string) string {
	t.Helper()
	needsShell(t)
	path := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCompleteAnswersWithNoToolsAndNoMcpServers(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	script := writeFakeAgent(t, `{ echo "args=$*"; cat; } > '`+record+`'; echo "the answer"`)
	svc := &agentServer{command: []string{script}}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetAnswer().GetText() != "the answer" {
		t.Fatalf("got %+v", res.Msg)
	}
	seen, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--max-turns 1", "--strict-mcp-config", "--append-system-prompt be brief", "why?"} {
		if !strings.Contains(string(seen), want) {
			t.Errorf("command did not see %q in:\n%s", want, seen)
		}
	}
}

func TestCompleteIsUnavailableWhenNoAgentCanAnswer(t *testing.T) {
	withoutAgentsAnywhere(t)
	svc := &agentServer{}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE || res.Msg.GetAnswer() != nil {
		t.Errorf("got %+v", res.Msg)
	}
}

func TestCompleteIsUnavailableWhenTheAgentFails(t *testing.T) {
	script := writeFakeAgent(t, `echo "not signed in" >&2; exit 1`)
	svc := &agentServer{command: []string{script}}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE {
		t.Errorf("got %+v", res.Msg)
	}
}

func TestCompleteStopsTheAgentWhenCancelled(t *testing.T) {
	script := writeFakeAgent(t, `exec sleep 30`)
	svc := &agentServer{command: []string{script}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := svc.Complete(ctx, askCompletion("why?"))
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("got %v", err)
	}
}

// Claude Code, found where it is installed, answers when no command line is set.
func TestCompleteUsesTheInstalledClaudeCodeWhenNoCommandIsSet(t *testing.T) {
	needsShell(t)
	withoutAgentsAnywhere(t)
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\ncat > /dev/null\necho from the cli\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+"/bin:/usr/bin")
	svc := &agentServer{}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetAnswer().GetText() != "from the cli" {
		t.Errorf("got %+v", res.Msg)
	}
}
