package codexcli

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
)

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

// A program that exits as soon as it has listed its models is not reported as killed: on macOS the
// kill of its exited process group fails.
func TestACodexThatExitsAfterListingItsModelsListsThem(t *testing.T) {
	needsShell(t)
	script := writeScript(t, "codex",
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
	needsShell(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	script := writeScript(t, "codex",
		fmt.Sprintf("echo $$ > %s\n", quoteShell(pidFile))+
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
}

func TestAStoppedProgramIsNotAFailure(t *testing.T) {
	if !isStopped(nil) || !isStopped(&exec.ExitError{}) || !isStopped(context.Canceled) ||
		!isStopped(os.ErrProcessDone) || !isStopped(syscall.ESRCH) {
		t.Error("expected standard stop errors to report isStopped = true")
	}
	if isStopped(errors.New("other")) {
		t.Error("expected arbitrary error to report isStopped = false")
	}
}

func TestStoredModelsAreNoneWithoutAHome(t *testing.T) {
	isolateWithoutHome(t)
	if got := readCodexCache(); got != nil {
		t.Errorf("readCodexCache without home = %v, want nil", got)
	}
}

func isolateWithoutHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	if _, err := os.UserHomeDir(); err == nil {
		t.Skip("this platform finds a home without any environment")
	}
}
