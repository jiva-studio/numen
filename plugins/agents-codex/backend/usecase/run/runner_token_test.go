package run_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
)

const (
	orchardToolsURL   = "http://127.0.0.1:4410/mcp"
	orchardToolsToken = "not-a-secret"
)

func runCodexWithTools(t *testing.T) (arguments []string, token string) {
	t.Helper()
	record := t.TempDir()
	argumentsFile, tokenFile := filepath.Join(record, "arguments"), filepath.Join(record, "token")
	script := agenttest.WriteScript(t, "codex",
		`printf '%s\n' "$@" > `+agenttest.QuoteShell(argumentsFile)+"\n"+
			`printf '%s' "$NUMEN_TOKEN" > `+agenttest.QuoteShell(tokenFile)+"\n"+
			`echo '{"type":"turn.completed"}'`+"\n")
	var reg agentrun.Registry

	w, err := newTestRunner().Start(t.Context(), &reg, &agentrun.Request{
		Command: []string{script}, Prompt: "Shelve the atlas of rivers.", Tools: []string{"atlas_open"},
		Root: t.TempDir(), ToolsURL: orchardToolsURL, ToolsToken: orchardToolsToken,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	for range w.Output {
	}
	if err := w.Stop(); err != nil {
		t.Error(err)
	}

	recorded, err := os.ReadFile(argumentsFile)
	if err != nil {
		t.Fatal(err)
	}
	environment, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(recorded), "\n"), "\n"), string(environment)
}

func TestTheCommandOfACodexToolsRunCarriesNoToken(t *testing.T) {
	arguments, _ := runCodexWithTools(t)

	for _, argument := range arguments {
		if strings.Contains(argument, orchardToolsToken) {
			t.Errorf("the Codex command line carries the token in %q", argument)
		}
	}
}

func TestACodexToolsRunReadsTheTokenOfTheMCPServerFromNumenToken(t *testing.T) {
	arguments, token := runCodexWithTools(t)

	for _, want := range []string{
		`mcp_servers.numen.url="` + orchardToolsURL + `"`,
		`mcp_servers.numen.bearer_token_env_var="NUMEN_TOKEN"`,
	} {
		index := slices.Index(arguments, want)
		if index < 1 || arguments[index-1] != "-c" {
			t.Errorf("the Codex command line %q does not set -c %s", arguments, want)
		}
	}
	if token != orchardToolsToken {
		t.Errorf("the Codex process has NUMEN_TOKEN=%q, want %q", token, orchardToolsToken)
	}
}
