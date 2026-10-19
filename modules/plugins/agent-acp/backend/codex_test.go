package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCodexModelCacheClearsFailedKeyOnSuccess(t *testing.T) {
	needsShell(t)
	t.Setenv("HOME", t.TempDir())
	failing := writeScript(t, "codex", "exit 1\n")
	now := time.Unix(1000, 0)
	cache := codexModelCache{now: func() time.Time { return now }}

	if got := cache.get(t.Context(), []string{failing}); got != nil {
		t.Fatalf("expected nil from failed script, got %v", got)
	}
	if cache.failedKey == "" {
		t.Fatal("expected failedKey to be set")
	}

	if err := os.WriteFile(failing, []byte("#!/bin/sh\n"+
		`echo '{"id":1,"result":{}}'`+"\n"+
		`echo '{"id":2,"result":{"data":[{"model":"gpt-ok"}]}}'`+"\n"+
		"while read -r _; do :; done\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	now = now.Add(codexFailureBackoff)
	got := cache.get(t.Context(), []string{failing})
	if len(got) != 1 || got[0].Name != "gpt-ok" {
		t.Fatalf("expected 1 model, got %v", got)
	}
	if cache.failedKey != "" {
		t.Errorf("expected failedKey to be cleared, got %q", cache.failedKey)
	}
}

func TestGetCodexModelsFallbackToCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cacheContent := `{"models": [{"slug": "fallback-model", "display_name": "Fallback Model", "visibility": "show"}]}`
	if err := os.WriteFile(filepath.Join(codexDir, "models_cache.json"), []byte(cacheContent), 0o600); err != nil {
		t.Fatal(err)
	}

	models, err := GetCodexModels(t.Context(), []string{"/nonexistent/bin"})
	if err != nil {
		t.Fatalf("expected no error falling back to cache, got %v", err)
	}
	if len(models) != 1 || models[0].Name != "fallback-model" {
		t.Fatalf("expected fallback-model from cache, got %v", models)
	}
}

func TestAskCodexModelsWithEmptyCommand(t *testing.T) {
	withoutAgentsAnywhere(t)
	models, err := askCodexModels(t.Context(), nil)
	if err == nil && len(models) > 0 {
		t.Fatalf("expected error or no models, got %v", models)
	}
}

func TestAskCodexModelsCommandArguments(t *testing.T) {
	needsShell(t)
	out := filepath.Join(t.TempDir(), "args")
	script := writeScript(t, "codex",
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+quoteShell(out)+"\n"+
			`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"m1"}]}}'`+"\n"+
			"while read -r _; do :; done\n")

	models, err := askCodexModels(t.Context(), []string{script, "--custom-arg"})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %v", models)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(args) != 2 || args[0] != "--custom-arg" || args[1] != "app-server" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestAskCodexModelsStartFailure(t *testing.T) {
	_, err := askCodexModels(t.Context(), []string{"/nonexistent/path/to/binary"})
	if err == nil {
		t.Fatal("expected error starting non-existent command, got nil")
	}
}

func TestAskCodexModelsStopsCommand(t *testing.T) {
	needsShell(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	script := writeScript(t, "codex",
		fmt.Sprintf("echo $$ > %s\n", quoteShell(pidFile))+
			`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"m1"}]}}'`+"\n"+
			"while :; do sleep 0.05; done\n")

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	start := time.Now()
	models, err := askCodexModels(ctx, []string{script})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %v", models)
	}
	if elapsed > 2*time.Second {
		t.Errorf("expected prompt return, took %v", elapsed)
	}

	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		t.Fatal(err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("expected process %d to be stopped and reaped, got %v", pid, err)
	}
}

func TestListCodexModelsIdFallbackAndEmptyFiltering(t *testing.T) {
	in := &bytes.Buffer{}
	response := `{"id":1,"result":{}}` + "\n" +
		`{"id":2,"result":{"data":[` +
		`{"id":"id-only","displayName":"ID Only"},` +
		`{"id":"","model":"","displayName":"No Name"},` +
		`{"id":"hidden-model","model":"hidden-model","hidden":true},` +
		`{"model":"normal-model","displayName":"Normal"}` +
		`]}}` + "\n"
	out := strings.NewReader(response)

	models, err := listCodexModels(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d: %v", len(models), models)
	}
	if models[0].Name != "id-only" {
		t.Errorf("expected id-only fallback, got %s", models[0].Name)
	}
	if models[1].Name != "normal-model" {
		t.Errorf("expected normal-model, got %s", models[1].Name)
	}
}

func TestReadCodexAnswerFiltersInterleavedMessages(t *testing.T) {
	lines := bufio.NewScanner(strings.NewReader(
		"not-json-line\n" +
			`{"id":999,"result":{"ignored":true}}` + "\n" +
			`{"id":1,"result":{"target":true}}` + "\n",
	))

	res, err := readCodexAnswer(lines, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res), `"target":true`) {
		t.Errorf("expected target result, got %s", string(res))
	}
}

func TestReadCodexAnswerScannerError(t *testing.T) {
	lines := bufio.NewScanner(strings.NewReader(strings.Repeat("x", 70000) + "\n"))
	_, err := readCodexAnswer(lines, 1)
	if !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("expected bufio.ErrTooLong, got %v", err)
	}
}

func TestWriteCodexLineWritesContent(t *testing.T) {
	var buf bytes.Buffer
	msg := map[string]any{"id": 42, "hello": "world"}
	if err := writeCodexLine(&buf, msg); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"hello":"world"`) || !strings.HasSuffix(buf.String(), "\n") {
		t.Errorf("unexpected written content: %q", buf.String())
	}
}

func TestListCodexModelsLargeLine(t *testing.T) {
	in := &bytes.Buffer{}
	longTitle := strings.Repeat("A", 70000)
	response := `{"id":1,"result":{}}` + "\n" +
		fmt.Sprintf(`{"id":2,"result":{"data":[{"model":"m1","displayName":%q}]}}`+"\n", longTitle)
	out := strings.NewReader(response)

	models, err := listCodexModels(in, out)
	if err != nil {
		t.Fatalf("expected successful list with large line, got %v", err)
	}
	if len(models) != 1 || models[0].Title != longTitle {
		t.Fatalf("expected model with large title, got %v", models)
	}
}
