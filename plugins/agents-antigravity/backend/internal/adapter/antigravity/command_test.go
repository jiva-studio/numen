package antigravity

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/task"
)

func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
}

type plainRun struct {
	supervised int
}

func (p *plainRun) Supervise(*exec.Cmd) {
	p.supervised++
}

func TestHomeSetupAndMCPConfigEdgeCases(t *testing.T) {
	if err := releaseNothing(); err != nil {
		t.Errorf("noRelease() = %v, want nil", err)
	}

	// readMCPServers with missing file and invalid JSON returns nil.
	if got := readMCPServers("/nonexistent/mcp_config.json"); got != nil {
		t.Errorf("readMCPServers missing = %v, want nil", got)
	}
	badJSONPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badJSONPath, []byte(`{"mcpServers":{"s1":{}},"broken":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readMCPServers(badJSONPath); got != nil {
		t.Errorf("readMCPServers bad JSON = %v, want nil", got)
	}

	// makeMCPConfig preserves existing non-numen servers and overwrites existing "numen" server.
	existingPath := filepath.Join(t.TempDir(), "mcp_config.json")
	if err := os.WriteFile(existingPath, []byte(`{"mcpServers":{"numen":{"serverUrl":"old"},"other":{"serverUrl":"http://other"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	merged := makeMCPConfig(endpoint{URL: "http://new/mcp", Token: "tok"}, existingPath)
	servers, ok := merged["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("makeMCPConfig merged = %+v, want mcpServers map", merged)
	}
	numenSrv, ok := servers["numen"].(map[string]any)
	if !ok {
		t.Fatalf("makeMCPConfig merged = %+v, want numen server map", merged)
	}
	if numenSrv["serverUrl"] != "http://new/mcp" || servers["other"] == nil {
		t.Fatalf("makeMCPConfig merged = %+v", merged)
	}

	// setupAntigravityHome with tools != nil writes the numen server into tempConfig/mcp_config.json
	// and links entries before and after the excluded name.
	home := t.TempDir()
	linked := []string{
		"a-before", "z-after", filepath.Join(".gemini", "sub-a"), filepath.Join(".gemini", "sub-z"),
		filepath.Join(".gemini", "config", "cfg-a"), filepath.Join(".gemini", "config", "cfg-z"),
	}
	for _, rel := range linked {
		full := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(rel), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	tempHome, clean, err := setupAntigravityHome(&endpoint{URL: "http://vault/mcp", Token: "vtok"})
	if err != nil || !filepath.IsAbs(tempHome) {
		t.Fatalf("setupAntigravityHome with tools = %q, %v", tempHome, err)
	}
	for _, rel := range linked {
		if got, err := os.ReadFile(filepath.Join(tempHome, rel)); err != nil || string(got) != rel {
			t.Errorf("linked %s = %q, %v", rel, got, err)
		}
	}
	configFile := filepath.Join(tempHome, ".gemini", "config", "mcp_config.json")
	written, err := os.ReadFile(configFile)
	if err != nil || !strings.Contains(string(written), "http://vault/mcp") || !strings.Contains(string(written), "Bearer vtok") {
		t.Errorf("written mcp_config.json = %q, %v", written, err)
	}
	if runtime.GOOS != "windows" {
		if dirInfo, err := os.Stat(filepath.Dir(configFile)); err != nil || dirInfo.Mode().Perm() != 0o700 {
			t.Errorf("tempConfig mode = %v, %v, want 0700", dirInfo.Mode().Perm(), err)
		}
		if fileInfo, err := os.Stat(configFile); err != nil || fileInfo.Mode().Perm() != 0o600 {
			t.Errorf("mcp_config.json mode = %v, %v, want 0600", fileInfo.Mode().Perm(), err)
		}
	}
	if err := clean(); err != nil {
		t.Errorf("clean tempHome: %v", err)
	}

	if err := linkEntries(home, filepath.Join(t.TempDir(), "missing-into"), ".gemini"); err == nil {
		t.Error("linkEntries into missing directory succeeded, want error")
	}

	// setupAntigravityHome error paths:
	// 1) HOME is a regular file -> linkEntries(realHome, tempHome, ".gemini") fails and wraps *os.PathError.
	fileAsHome := filepath.Join(t.TempDir(), "file-home")
	if err := os.WriteFile(fileAsHome, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", fileAsHome)
	if _, clean, err := setupAntigravityHome(nil); err == nil || clean == nil {
		t.Errorf("expected error and non-nil clean when HOME is a regular file: clean is nil %v, err %v", clean == nil, err)
	} else if pathErr, ok := errors.AsType[*os.PathError](err); !ok || pathErr.Path == "" {
		t.Errorf("expected wrapped *os.PathError with a path, got %T (%v)", err, err)
	}

	// 2) HOME/.gemini is a regular file -> linkEntries(realGemini, tempGemini, "config") fails.
	homeWithGeminiFile := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeWithGeminiFile, ".gemini"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeWithGeminiFile)
	if _, clean, err := setupAntigravityHome(nil); err == nil || clean == nil {
		t.Errorf("expected error and non-nil clean when HOME/.gemini is a regular file: clean is nil %v, err %v", clean == nil, err)
	}

	// 3) HOME/.gemini/config is a regular file -> linkEntries(realConfig, tempConfig, "mcp_config.json") fails.
	homeWithConfigFile := t.TempDir()
	if err := os.MkdirAll(filepath.Join(homeWithConfigFile, ".gemini"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homeWithConfigFile, ".gemini", "config"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeWithConfigFile)
	if _, clean, err := setupAntigravityHome(nil); err == nil || clean == nil {
		t.Errorf("expected error and non-nil clean when HOME/.gemini/config is a regular file: clean is nil %v, err %v", clean == nil, err)
	}

	// 4) No HOME environment variable -> os.UserHomeDir() fails.
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	if _, err := os.UserHomeDir(); err != nil {
		if _, clean, err := setupAntigravityHome(nil); err == nil || clean == nil {
			t.Errorf("expected error and non-nil clean when os.UserHomeDir fails: clean is nil %v, err %v", clean == nil, err)
		}
	}

	// 5) Unwritable TMPDIR -> createTempDir fails.
	t.Setenv("HOME", t.TempDir())
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, filepath.Join(t.TempDir(), "missing-tmp"))
	}
	if _, clean, err := setupAntigravityHome(nil); err == nil || clean == nil {
		t.Errorf("expected error and non-nil clean when TMPDIR is missing: clean is nil %v, err %v", clean == nil, err)
	}
	if _, err := makeAnswerCommand("/opt/agy", task.Run{Prompt: "p"}); err == nil {
		t.Error("expected error from makeAnswerCommand when TMPDIR is missing")
	}
}

// A task with tools approves the permission requests of the program and gives it a home whose MCP
// configuration names the tools; the default model is not named. An answer approves nothing, and names
// the model and the conversation it continues. The prompt is the message on the standard input, never an
// argument. Each command releases the folders made for it.
func TestTheCommandsAreMadeFromTheRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	withTools, err := makeTaskCommand("/opt/agy", task.Run{
		Prompt: "Summarize \"Orchard\"", Model: model.DefaultName, Session: "sess-a", ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "tk",
	})
	if err != nil || withTools.Release == nil || withTools.Path != "/opt/agy" {
		t.Fatalf("makeTaskCommand = %+v, %v", withTools, err)
	}
	want := []string{
		"--print=", "--input-format", "stream-json", "--output-format", "stream-json", "--dangerously-skip-permissions", "--conversation", "sess-a",
	}
	if !slices.Equal(withTools.Args, want) {
		t.Errorf("the arguments are %v, want %v", withTools.Args, want)
	}
	if wantInput := "{\"event\":\"user\",\"message\":{\"content\":\"Summarize \\\"Orchard\\\"\"}}\n"; withTools.Stdin != wantInput {
		t.Errorf("the input is %q, want %q", withTools.Stdin, wantInput)
	}
	home := ""
	for _, one := range withTools.Env {
		if value, isHome := strings.CutPrefix(one, "HOME="); isHome {
			home = value
		}
	}
	written, err := os.ReadFile(filepath.Join(home, ".gemini", "config", "mcp_config.json"))
	if err != nil || !strings.Contains(string(written), "http://127.0.0.1:1/mcp") || !strings.Contains(string(written), "Bearer tk") {
		t.Errorf("the home names the servers %s, %v", written, err)
	}
	if err := withTools.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Errorf("the home is still there: %v", err)
	}

	answer, err := makeAnswerCommand("/opt/agy", task.Run{Prompt: "p", Model: "m-ans", Session: "sess-a"})
	if err != nil || answer.Release == nil {
		t.Fatalf("makeAnswerCommand = %+v, %v", answer, err)
	}
	if want := []string{
		"--print=", "--input-format", "stream-json", "--output-format", "stream-json", "--model", "m-ans", "--conversation", "sess-a",
	}; !slices.Equal(answer.Args, want) || !strings.Contains(answer.Stdin, "\"content\":\"p\"") {
		t.Errorf("the arguments are %v and the input %q, want %v and the prompt", answer.Args, answer.Stdin, want)
	}
	if err := answer.Release(); err != nil {
		t.Fatal(err)
	}

	noTools, err := makeTaskCommand("/opt/agy", task.Run{Prompt: "p"})
	if err != nil || slices.Contains(noTools.Args, "--dangerously-skip-permissions") || slices.Contains(noTools.Args, "--model") {
		t.Errorf("a task with no tools = %v, %v", noTools.Args, err)
	}
	if err := noTools.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestTheModelsAreListedAfterTheExtraArguments(t *testing.T) {
	sup := &plainRun{}
	cli := New(sup)

	listed, err := cli.List(t.Context(), []string{
		"/bin/sh", "-c", `if [ "$1" != "--extra" ] || [ "$2" != "models" ]; then exit 2; fi; printf 'm1\tModel 1\n'`, "sh", "--extra",
	})

	if err != nil || len(listed) != 1 || listed[0].Name != "m1" || sup.supervised != 1 {
		t.Fatalf("List with extra args = (%v, %v), supervised = %d", listed, err, sup.supervised)
	}

	if runtime.GOOS != "windows" {
		binDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(binDir, "agy"), []byte("#!/bin/sh\nif [ \"$1\" = models ]; then printf 'm2\\tModel 2\\n'; fi\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", binDir)
		defaultListed, err := cli.List(t.Context(), nil)
		if err != nil || len(defaultListed) != 1 || defaultListed[0].Name != "m2" || sup.supervised != 2 {
			t.Fatalf("List with default command = (%v, %v), supervised = %d", defaultListed, err, sup.supervised)
		}
	}
}

func TestAProgramThatFailsListsNoModels(t *testing.T) {
	_, err := New(&plainRun{}).List(t.Context(), []string{"/bin/sh", "-c", "exit 3"})

	if err == nil || !strings.Contains(err.Error(), "list the models of /bin/sh") {
		t.Errorf("List error = %v", err)
	} else if exitErr, ok := errors.AsType[*exec.ExitError](err); !ok || exitErr.ExitCode() != 3 {
		t.Errorf("expected wrapped *exec.ExitError with code 3, got %T (%v)", err, err)
	}
}
