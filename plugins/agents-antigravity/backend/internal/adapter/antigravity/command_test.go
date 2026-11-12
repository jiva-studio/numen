package antigravity

import (
	"os"
	"os/exec"
	"path/filepath"
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

type plainRun struct{}

func (plainRun) Supervise(*exec.Cmd) {}

func TestHomeSetupAndMCPConfigEdgeCases(t *testing.T) {
	if err := noRelease(); err != nil {
		t.Errorf("noRelease() = %v, want nil", err)
	}

	// readMCPServers with missing file and invalid JSON returns nil.
	if got := readMCPServers("/nonexistent/mcp_config.json"); got != nil {
		t.Errorf("readMCPServers missing = %v, want nil", got)
	}
	badJSONPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badJSONPath, []byte("{bad"), 0o600); err != nil {
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

	// setupAntigravityHome with tools != nil writes the numen server into tempConfig/mcp_config.json.
	home := t.TempDir()
	t.Setenv("HOME", home)
	tempHome, clean, err := setupAntigravityHome(&endpoint{URL: "http://vault/mcp", Token: "vtok"})
	if err != nil {
		t.Fatalf("setupAntigravityHome with tools: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(tempHome, ".gemini", "config", "mcp_config.json"))
	if err != nil || !strings.Contains(string(written), "http://vault/mcp") || !strings.Contains(string(written), "Bearer vtok") {
		t.Errorf("written mcp_config.json = %q, %v", written, err)
	}
	if err := clean(); err != nil {
		t.Errorf("clean tempHome: %v", err)
	}

	// setupAntigravityHome error paths:
	// 1) HOME is a regular file -> LinkEntries(realHome, tempHome, ".gemini") fails.
	fileAsHome := filepath.Join(t.TempDir(), "file-home")
	if err := os.WriteFile(fileAsHome, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", fileAsHome)
	if _, _, err := setupAntigravityHome(nil); err == nil {
		t.Error("expected error when HOME is a regular file")
	}

	// 2) HOME/.gemini is a regular file -> LinkEntries(realGemini, tempGemini, "config") fails.
	homeWithGeminiFile := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeWithGeminiFile, ".gemini"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeWithGeminiFile)
	if _, _, err := setupAntigravityHome(nil); err == nil {
		t.Error("expected error when HOME/.gemini is a regular file")
	}

	// 3) HOME/.gemini/config is a regular file -> LinkEntries(realConfig, tempConfig, "mcp_config.json") fails.
	homeWithConfigFile := t.TempDir()
	if err := os.MkdirAll(filepath.Join(homeWithConfigFile, ".gemini"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homeWithConfigFile, ".gemini", "config"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeWithConfigFile)
	if _, _, err := setupAntigravityHome(nil); err == nil {
		t.Error("expected error when HOME/.gemini/config is a regular file")
	}

	// 4) No HOME environment variable -> os.UserHomeDir() fails.
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	if _, err := os.UserHomeDir(); err != nil {
		if _, _, err := setupAntigravityHome(nil); err == nil {
			t.Error("expected error when os.UserHomeDir fails")
		}
	}
}

// A task with tools approves the permission requests of the program and gives it a home whose MCP
// configuration names the tools; the default model is not named. An answer approves nothing, and names
// the model and the conversation it continues. Each command releases the folders made for it.
func TestTheCommandsAreMadeFromTheRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	withTools, err := makeTaskCommand("/opt/agy", task.Run{
		Prompt: "p", Model: model.DefaultName, Session: "sess-a", ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "tk",
	})
	if err != nil || withTools.Release == nil || withTools.Path != "/opt/agy" {
		t.Fatalf("makeTaskCommand = %+v, %v", withTools, err)
	}
	want := []string{"-p", "p", "--output-format", "stream-json", "--dangerously-skip-permissions", "--conversation", "sess-a"}
	if !slices.Equal(withTools.Args, want) {
		t.Errorf("the arguments are %v, want %v", withTools.Args, want)
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
	if want := []string{"-p", "p", "--output-format", "stream-json", "--model", "m-ans", "--conversation", "sess-a"}; !slices.Equal(answer.Args, want) {
		t.Errorf("the arguments are %v, want %v", answer.Args, want)
	}
	if err := answer.Release(); err != nil {
		t.Fatal(err)
	}

	noTools, err := makeTaskCommand("/opt/agy", task.Run{Prompt: "p"})
	if err != nil || slices.Contains(noTools.Args, "--dangerously-skip-permissions") {
		t.Errorf("a task with no tools = %v, %v", noTools.Args, err)
	}
	if err := noTools.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestTheModelsAreListedAfterTheExtraArguments(t *testing.T) {
	cli := New(plainRun{})

	listed, err := cli.List(t.Context(), []string{
		"/bin/sh", "-c", `if [ "$1" != "--extra" ] || [ "$2" != "models" ]; then exit 2; fi; printf 'm1\tModel 1\n'`, "sh", "--extra",
	})

	if err != nil || len(listed) != 1 || listed[0].Name != "m1" {
		t.Fatalf("List with extra args = (%v, %v)", listed, err)
	}
}

func TestAProgramThatFailsListsNoModels(t *testing.T) {
	_, err := New(plainRun{}).List(t.Context(), []string{"/bin/sh", "-c", "exit 3"})

	if err == nil || !strings.Contains(err.Error(), "list the models of /bin/sh") {
		t.Errorf("List error = %v", err)
	}
}
