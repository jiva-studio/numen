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

func writeFakeClaude(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCompleteAnswersWithTheConfiguredKeyAndNoTools(t *testing.T) {
	t.Setenv(anthropicKeyVar, "env-key")
	record := filepath.Join(t.TempDir(), "record")
	script := writeFakeClaude(t, `{ echo "key=$ANTHROPIC_API_KEY"; echo "args=$*"; cat; } > '`+record+`'; echo "the answer"`)
	svc := &agentServer{command: []string{script}, apiKey: "host-key", modelName: "m1"}

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
	for _, want := range []string{"key=host-key", "--max-turns 1", "--strict-mcp-config", "--model m1", "why?"} {
		if !strings.Contains(string(seen), want) {
			t.Errorf("command did not see %q in:\n%s", want, seen)
		}
	}
	if strings.Contains(string(seen), "env-key") || strings.Contains(string(seen), "--mcp-config") {
		t.Errorf("command saw what it must not:\n%s", seen)
	}
}

func TestCompleteIsUnavailableWithoutAKey(t *testing.T) {
	t.Setenv(anthropicKeyVar, "env-key")
	ran := filepath.Join(t.TempDir(), "ran")
	script := writeFakeClaude(t, `touch '`+ran+`'; echo invented`)
	svc := &agentServer{command: []string{script}}

	res, err := svc.Complete(context.Background(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE || res.Msg.GetAnswer() != nil {
		t.Errorf("got %+v", res.Msg)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("the command ran without a key")
	}
}

func TestCompleteStopsTheCommandWhenCancelled(t *testing.T) {
	script := writeFakeClaude(t, `exec sleep 30`)
	svc := &agentServer{command: []string{script}, apiKey: "host-key"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := svc.Complete(ctx, askCompletion("why?"))
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("got %v", err)
	}
}

func TestConfigureKeepsTheKeyTheHostSends(t *testing.T) {
	svc := &agentServer{}
	svc.applyConfig(context.Background(), map[string]string{"apiKey": "k1", "baseUrl": "http://127.0.0.1:1"})
	if svc.apiKey != "k1" || svc.baseURL != "http://127.0.0.1:1" {
		t.Errorf("got %q %q", svc.apiKey, svc.baseURL)
	}
	svc.applyConfig(context.Background(), map[string]string{"apiKey": ""})
	if svc.apiKey != "" {
		t.Errorf("the key was kept after the host cleared it: %q", svc.apiKey)
	}
}

func TestCompletionEnvironmentReplacesTheHeldKey(t *testing.T) {
	got := buildCompletionEnvironment([]string{"A=1", "ANTHROPIC_API_KEY=old", "ANTHROPIC_BASE_URL=old", "CLAUDECODE=1"}, "new", "")
	want := []string{"A=1", "ANTHROPIC_API_KEY=new"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v", got)
	}
}
