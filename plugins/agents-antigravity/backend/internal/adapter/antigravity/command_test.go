package antigravity

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
)

func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
}

type plainRun struct{}

func (plainRun) Supervise(*exec.Cmd) {}

func TestResolveCommandWithEmptyCommand(t *testing.T) {
	withoutAgentsAnywhere(t)

	bin, args := resolveCommand(nil)
	if want := findProgram(); bin != want || len(args) != 0 {
		t.Errorf("expected %s and empty args, got %s, %v", want, bin, args)
	}
}

func TestTheSessionOfTheConversationIsHandedToTheNextRun(t *testing.T) {
	var reg agentrun.Registry
	reg.SetSession("c-1", "s-1")

	cmd, err := makeAntigravityCommand(t.Context(), &agentrun.Request{Conversation: "c-1", Prompt: "q"}, &reg)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Release != nil {
		defer func() {
			if err := cmd.Release(); err != nil {
				t.Error(err)
			}
		}()
	}

	if !slices.Contains(cmd.Args, "--conversation") || !slices.Contains(cmd.Args, "s-1") {
		t.Errorf("the arguments are %v", cmd.Args)
	}
}

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

func TestTheCommandsAreMadeFromTheRequest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var reg agentrun.Registry
	reg.SetSession("conv-a", "sess-a")
	shPath, err := exec.LookPath("sh")
	if err == nil {
		cmdWithTools, err := makeAntigravityCommand(t.Context(), &agentrun.Request{
			Prompt: "p", Tools: []string{"t"}, Model: model.DefaultName, Conversation: "conv-a",
			ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "tk",
		}, &reg, shPath)
		if err != nil || cmdWithTools.Release == nil || cmdWithTools.Path != shPath {
			t.Fatalf("makeAntigravityCommand with ToolsURL: Path=%q, err=%v", cmdWithTools.Path, err)
		}
		if slices.Contains(cmdWithTools.Args, "--model") || !slices.Contains(cmdWithTools.Args, "--conversation") {
			t.Errorf("makeAntigravityCommand Args = %v", cmdWithTools.Args)
		}
		if err := cmdWithTools.Release(); err != nil {
			t.Fatal(err)
		}

		ansCmd, err := makeAnswerCommand(t.Context(), &agentrun.Request{
			Prompt: "p", Model: "m-ans", Conversation: "conv-a",
		}, &reg, shPath)
		if err != nil || ansCmd.Release == nil || ansCmd.Path != shPath {
			t.Fatalf("makeAnswerCommand: Path=%q, err=%v", ansCmd.Path, err)
		}
		if !slices.Contains(ansCmd.Args, "--model") || !slices.Contains(ansCmd.Args, "m-ans") || !slices.Contains(ansCmd.Args, "--conversation") {
			t.Errorf("makeAnswerCommand Args = %v", ansCmd.Args)
		}
		if err := ansCmd.Release(); err != nil {
			t.Fatal(err)
		}

		ansDefaultCmd, err := makeAnswerCommand(t.Context(), &agentrun.Request{
			Prompt: "p", Model: model.DefaultName,
		}, &reg, shPath)
		if err != nil || ansDefaultCmd.Release == nil || slices.Contains(ansDefaultCmd.Args, "--model") {
			t.Errorf("makeAnswerCommand with model.DefaultName Args = %v, err = %v", ansDefaultCmd.Args, err)
		}
		if ansDefaultCmd.Release != nil {
			if err := ansDefaultCmd.Release(); err != nil {
				t.Error(err)
			}
		}
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
