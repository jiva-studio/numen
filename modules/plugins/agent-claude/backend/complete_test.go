package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
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
	t.Setenv("NUMEN_CLAUDE_COMMAND", script)
	svc := &agentServer{}
	svc.applyConfig(map[string]string{"apiKey": "host-key", "model": "m1"})

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
	t.Setenv("NUMEN_CLAUDE_COMMAND", script)
	svc := &agentServer{}

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
	t.Setenv("NUMEN_CLAUDE_COMMAND", script)
	svc := &agentServer{}
	svc.applyConfig(map[string]string{"apiKey": "host-key"})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := svc.Complete(ctx, askCompletion("why?"))
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("got %v", err)
	}
}

func TestConfigureKeepsTheKeyTheHostSends(t *testing.T) {
	svc := &agentServer{}
	svc.applyConfig(map[string]string{"apiKey": "k1", "baseUrl": "http://127.0.0.1:1"})
	if key, base := svc.catalogue.credentials(); key != "k1" || base != "http://127.0.0.1:1" {
		t.Errorf("got %q %q", key, base)
	}
	svc.applyConfig(map[string]string{"apiKey": ""})
	if key, _ := svc.catalogue.credentials(); key != "" {
		t.Errorf("the key was kept after the host cleared it: %q", key)
	}
}

func TestCompletionEnvironmentReplacesTheHeldKey(t *testing.T) {
	got := buildCompletionEnvironment([]string{"A=1", "ANTHROPIC_API_KEY=old", "ANTHROPIC_BASE_URL=old", "CLAUDECODE=1"}, "new", "")
	want := []string{"A=1", "ANTHROPIC_API_KEY=new"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v", got)
	}
}

func TestCompleteAnswersWithTheModelTheHostNames(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	script := writeFakeClaude(t, `{ echo "args=$*"; cat; } > '`+record+`'; echo "the answer"`)
	t.Setenv("NUMEN_CLAUDE_COMMAND", script)
	svc := &agentServer{}
	svc.applyConfig(map[string]string{"apiKey": "host-key", "model": "m1"})
	request := askCompletion("why?")
	request.Msg.Model = "m2"

	if _, err := svc.Complete(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	seen, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(seen), "--model m2") || strings.Contains(string(seen), "--model m1") {
		t.Errorf("command did not see the host's model in:\n%s", seen)
	}
}

// The models Complete answers with are those the endpoint lists, the configured one the default.
func TestTheCompletionModelsAreTheEndpoints(t *testing.T) {
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data": [{"id": "opus", "display_name": "Opus"}, {"id": "haiku", "display_name": "Haiku"}]}`))
	}))
	defer endpoint.Close()
	svc := &agentServer{}
	svc.applyConfig(map[string]string{"apiKey": "host-key", "baseUrl": endpoint.URL, "model": "haiku"})

	res, err := svc.ListCompletionModels(context.Background(), connect.NewRequest(&numenv1.ListCompletionModelsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	models := res.Msg.GetModels()
	if len(models) != 2 || models[0].GetName() != "opus" || models[0].GetIsDefault() ||
		models[1].GetName() != "haiku" || !models[1].GetIsDefault() {
		t.Errorf("listed %v", models)
	}
}

func TestCompletionArgumentsGiveNoToolsAndOneTurn(t *testing.T) {
	got := buildCompletionArguments("be brief", "m1")
	for _, pair := range [][2]string{
		{"--output-format", "text"}, {"--max-turns", "1"}, {"--tools", ""}, {"--setting-sources", ""},
		{"--append-system-prompt", "be brief"}, {"--model", "m1"},
	} {
		assertPair(t, got, pair[0], pair[1])
	}
	for _, flag := range []string{"-p", "--strict-mcp-config"} {
		if !slices.Contains(got, flag) {
			t.Errorf("%s absent from %q", flag, got)
		}
	}
	bare := buildCompletionArguments("", "")
	if slices.Contains(bare, "--append-system-prompt") || slices.Contains(bare, "--model") {
		t.Errorf("got %q", bare)
	}
}
