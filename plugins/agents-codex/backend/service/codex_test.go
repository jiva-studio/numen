package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
)

func TestCodexModelCacheClearsFailedKeyOnSuccess(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	failing := agenttest.WriteScript(t, "codex", "exit 1\n")
	now := clock.NewFake(time.Unix(1000, 0))
	cache := codexModelCache{clock: now}

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

	now.Advance(codexFailureBackoff)
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
	agenttest.NeedsShell(t)
	out := filepath.Join(t.TempDir(), "args")
	script := agenttest.WriteScript(t, "codex",
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+agenttest.QuoteShell(out)+"\n"+
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

// A program that exits as soon as it has listed its models is not reported as killed: on macOS the
// kill of its exited process group fails.
func TestACodexThatExitsAfterListingItsModelsListsThem(t *testing.T) {
	agenttest.NeedsShell(t)
	script := agenttest.WriteScript(t, "codex",
		"read -r _\n"+
			`echo '{"id":1,"result":{}}'`+"\n"+
			"read -r _\nread -r _\n"+
			`echo '{"id":2,"result":{"data":[{"model":"m1"}]}}'`+"\n"+
			"exit 0\n")

	models, err := askCodexModels(t.Context(), []string{script})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].Name != "m1" {
		t.Errorf("the models are %v", models)
	}
}

func TestAskCodexModelsStartFailure(t *testing.T) {
	_, err := askCodexModels(t.Context(), []string{"/nonexistent/path/to/binary"})
	if err == nil {
		t.Fatal("expected error starting non-existent command, got nil")
	}
}

func TestAskCodexModelsStopsCommand(t *testing.T) {
	agenttest.NeedsShell(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	script := agenttest.WriteScript(t, "codex",
		fmt.Sprintf("echo $$ > %s\n", agenttest.QuoteShell(pidFile))+
			`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"m1"}]}}'`+"\n"+
			"while :; do sleep 0.05; done\n")

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	models, err := askCodexModels(ctx, []string{script})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %v", models)
	}
	if ctx.Err() != nil {
		t.Errorf("expected return before the deadline, got %v", ctx.Err())
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

type failOnWriteWriter struct {
	failAt int
	writes int
}

func (w *failOnWriteWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.failAt {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

func TestReadCodexCacheAndListEdgeCases(t *testing.T) {
	if codexListLimit != 15*time.Second {
		t.Errorf("codexListLimit = %v, want 15s", codexListLimit)
	}
	if codexFailureBackoff != 30*time.Second {
		t.Errorf("codexFailureBackoff = %v, want 30s", codexFailureBackoff)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	// Missing models_cache.json returns nil.
	if got := readCodexCache(); got != nil {
		t.Errorf("readCodexCache missing file = %v, want nil", got)
	}

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(codexDir, "models_cache.json")
	// Invalid JSON returns nil.
	if err := os.WriteFile(cacheFile, []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readCodexCache(); got != nil {
		t.Errorf("readCodexCache invalid JSON = %v, want nil", got)
	}

	// Hidden and empty-slug models are skipped.
	content := `{"models":[` +
		`{"slug":"hidden-one","display_name":"Hidden","visibility":"hide"},` +
		`{"slug":"","display_name":"Empty","visibility":"show"},` +
		`{"slug":"visible-one","display_name":"Visible","visibility":"show","default_reasoning_level":"high","supported_reasoning_levels":[{"effort":""},{"effort":"high"},{"effort":"low"}]}` +
		`]}`
	if err := os.WriteFile(cacheFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cached := readCodexCache()
	if len(cached) != 1 || cached[0].Name != "visible-one" || cached[0].DefaultReasoning != "high" {
		t.Fatalf("readCodexCache filtering = %+v", cached)
	}
	if got := getLevelNames(cached[0]); !slices.Equal(got, []string{"low", "high"}) {
		t.Errorf("cached levels = %v, want [low high]", got)
	}

	// writeCodexLine fails on unmarshalable value.
	if err := writeCodexLine(&bytes.Buffer{}, map[string]any{"bad": make(chan int)}); err == nil {
		t.Error("expected writeCodexLine error on channel value")
	}

	// listCodexModels fails on each of the 3 writes.
	validOut := `{"id":1,"result":{}}` + "\n" + `{"id":2,"result":{"data":[{"model":"m1"}]}}` + "\n"
	for failAt := 1; failAt <= 3; failAt++ {
		if _, err := listCodexModels(&failOnWriteWriter{failAt: failAt}, strings.NewReader(validOut)); err == nil {
			t.Errorf("expected error when write %d fails", failAt)
		}
	}

	// listCodexModels fails when answer 1 has error or EOF, when answer 2 has error, or when result 2 is bad JSON.
	for _, outStr := range []string{
		"",
		`{"id":1,"error":{"message":"init failed"}}` + "\n",
		`{"id":1,"result":{}}` + "\n" + `{"id":2,"error":{"message":"list failed"}}` + "\n",
		`{"id":1,"result":{}}` + "\n" + `{"id":2,"result":"not-an-object"}` + "\n",
	} {
		if _, err := listCodexModels(&bytes.Buffer{}, strings.NewReader(outStr)); err == nil {
			t.Errorf("expected listCodexModels error for output %q", outStr)
		}
	}

	// newAgentModel ignores empty effort names.
	model := newAgentModel("m", "M", true, []string{"", "high", "low"}, "high")
	if got := getLevelNames(model); !slices.Equal(got, []string{"low", "high"}) {
		t.Errorf("newAgentModel levels = %v, want [low high]", got)
	}
}

func TestIsStoppedAndCacheAsyncAndServerEdgeCases(t *testing.T) {
	if !isStopped(nil) || !isStopped(&exec.ExitError{}) || !isStopped(context.Canceled) ||
		!isStopped(os.ErrProcessDone) || !isStopped(syscall.ESRCH) {
		t.Error("expected standard stop errors to report isStopped = true")
	}
	if isStopped(errors.New("other")) {
		t.Error("expected arbitrary error to report isStopped = false")
	}

	// GetCodexModels returns error when both command and cache fail.
	t.Setenv("HOME", t.TempDir())
	if models, err := GetCodexModels(t.Context(), []string{"/nonexistent-codex"}); err == nil || models != nil {
		t.Errorf("GetCodexModels with no cache and bad command = (%v, %v), want error", models, err)
	}

	// codexModelCache key separation, backoff boundary, and async fetch.
	now := clock.NewFake(time.Unix(1000, 0))
	c := codexModelCache{
		clock:     now,
		key:       "a b",
		models:    []AgentModel{{Name: "cached-ab"}},
		isCached:  true,
		failedKey: "fail-cmd",
		failedAt:  now.Now(),
	}
	if got := c.get(t.Context(), []string{"a", "b"}); len(got) != 1 || got[0].Name != "cached-ab" {
		t.Fatalf("expected cached-ab for key 'a b', got %v", got)
	}
	now.Advance(codexFailureBackoff - time.Nanosecond)
	if got := c.get(t.Context(), []string{"fail-cmd"}); got != nil {
		t.Fatalf("expected nil before codexFailureBackoff elapsed, got %v", got)
	}

	// Async cache returns fallback from disk while background fetch runs, and ignores duplicate inFlight starts.
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(`{"models":[{"slug":"disk-m","display_name":"Disk"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	asyncCache := codexModelCache{
		clock:    now,
		isAsync:  true,
		inFlight: map[string]bool{"already": true},
	}
	asyncCache.startBackgroundFetch(t.Context(), "already", []string{"/nonexistent"})
	cmd := []string{"/bin/sh", "-c", `read -r _; echo '{"id":1,"result":{}}'; read -r _; read -r _; echo '{"id":2,"result":{"data":[{"model":"bg-m"}]}}'`, "sh"}
	initial := asyncCache.get(t.Context(), cmd)
	if len(initial) != 1 || initial[0].Name != "disk-m" {
		t.Fatalf("async initial get = %v, want disk-m", initial)
	}
	wall := clock.System{}
	deadline := wall.Now().Add(25 * time.Second)
	for wall.Now().Before(deadline) {
		asyncCache.mu.Lock()
		cached := asyncCache.isCached
		asyncCache.mu.Unlock()
		if cached {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	bgGot := asyncCache.get(t.Context(), cmd)
	if len(bgGot) != 1 || bgGot[0].Name != "bg-m" {
		t.Fatalf("async cached get = %v, want bg-m", bgGot)
	}

	// Server applyConfig, GetPrograms with customPath, and Ask error paths.
	s := &agentServer{codexModels: codexModelCache{
		clock:    now,
		key:      "/custom/codex",
		models:   []AgentModel{{Name: "m-custom", Title: "Custom", IsDefault: true}},
		isCached: true,
	}}
	s.applyConfig(map[string]string{"executable_path": " /custom/codex "})
	s.applyConfig(map[string]string{"unrelated": "val"})
	if got := s.getExecutablePath(); got != "/custom/codex" {
		t.Fatalf("getExecutablePath = %q, want /custom/codex", got)
	}
	progs, err := s.GetPrograms(t.Context(), nil)
	if err != nil || len(progs) != 1 || !progs[0].IsInstalled || len(progs[0].Models) != 1 || progs[0].Models[0].Name != "m-custom" {
		t.Fatalf("GetPrograms with customPath = (%+v, %v)", progs, err)
	}

	sendErr := errors.New("send failed")
	if err := s.Ask(t.Context(), &agentrun.Request{Prompt: "q"}, func(*agentrun.Response) error {
		return sendErr
	}); !errors.Is(err, sendErr) {
		t.Errorf("Ask when first send fails = %v, want %v", err, sendErr)
	}

	var steps []*agentrun.Response
	if err := s.Ask(t.Context(), &agentrun.Request{Prompt: "q", Command: []string{"/nonexistent-codex-bin"}}, func(r *agentrun.Response) error {
		steps = append(steps, r)
		return nil
	}); err != nil {
		t.Errorf("Ask when start fails returned error: %v", err)
	}
	if len(steps) != 2 || steps[1].GetStopped() == "" {
		t.Errorf("Ask when start fails steps = %+v, want ModelStarted and Stopped", steps)
	}

	isolateWithoutHome(t)
	if got := readCodexCache(); got != nil {
		t.Errorf("readCodexCache without home = %v, want nil", got)
	}
}
