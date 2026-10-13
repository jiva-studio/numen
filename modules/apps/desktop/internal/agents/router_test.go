//go:build !nomcp

package agents

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// makeProgram is a program that writes its arguments down, one to a line, and answers nothing.
func makeProgram(t *testing.T, name string) (command []string, argv func() []string) {
	t.Helper()

	dir := t.TempDir()
	script := filepath.Join(dir, name)
	written := filepath.Join(dir, "argv")
	body := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + written + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return []string{script}, func() []string {
		raw, err := os.ReadFile(written)
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	}
}

// makeRouter is a router over three programs that write their arguments down.
func makeRouter(t *testing.T, tune func(*agent.Config)) (*Router, map[string]func() []string) {
	t.Helper()

	settings := agent.Defaults()
	argv := map[string]func() []string{}
	settings.Claude.Command, argv[agent.UseClaude] = makeProgram(t, "claude")
	settings.Codex.Command, argv[agent.UseCodex] = makeProgram(t, "codex")
	settings.Antigravity.Command, argv[agent.UseAntigravity] = makeProgram(t, "agy")
	if tune != nil {
		tune(&settings)
	}

	vocabulary := map[string]mcp.Tool{"note_read": {}, "note_edit": {}}
	router := NewRouter(Options{
		Config: makeInstallation(t), Root: t.TempDir(), Out: io.Discard,
	}.withAgent(settings), "http://127.0.0.1:7717/mcp", "secret", vocabulary)
	t.Cleanup(func() { _ = router.Close() })
	return router, argv
}

// withAgent is the options with the agent section of the settings replaced.
func (o Options) withAgent(settings agent.Config) Options {
	o.Config.Agent = settings
	return o
}

func drain(t *testing.T, run port.Run) {
	t.Helper()
	for range run.GetSteps() {
	}
}

// getFlag is what the flag was given on a command line.
func getFlag(argv []string, flag string) string {
	at := slices.Index(argv, flag)
	if at < 0 || at+1 >= len(argv) {
		return ""
	}
	return argv[at+1]
}

// The router is built with an agent for each name the settings can give, and a name it does not
// know is refused.
func TestARouterAnswersToEachNameAndRefusesOthers(t *testing.T) {
	router, _ := makeRouter(t, nil)

	for _, name := range []string{agent.UseClaude, agent.UseCodex, agent.UseAntigravity} {
		if _, ok := router.agents[name]; !ok {
			t.Errorf("no agent is built for %q", name)
		}
	}
	if _, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "gemini"}); err == nil {
		t.Error("a task naming an agent nobody built was taken")
	}
}

// A task naming no agent goes to the configured one, and one naming another goes there.
func TestATaskGoesToTheAgentItNamesOrTheConfiguredOne(t *testing.T) {
	router, argv := makeRouter(t, func(c *agent.Config) { c.Use = agent.UseCodex })

	run, err := router.StartRun(t.Context(), port.Task{Question: "to the configured"})
	if err != nil {
		t.Fatal(err)
	}
	drain(t, run)
	if got := argv[agent.UseCodex](); got[len(got)-1] != "to the configured" {
		t.Errorf("codex was asked %q", got)
	}

	run, err = router.StartRun(t.Context(), port.Task{Question: "to claude", Agent: agent.UseClaude})
	if err != nil {
		t.Fatal(err)
	}
	drain(t, run)
	if got := argv[agent.UseClaude](); !slices.Contains(got, "--allowedTools") {
		t.Errorf("claude was started with %q", got)
	}
}

// The step limit of the settings is the task's where the task names none, and the task's own
// is kept.
func TestTheStepLimitComesFromTheSettingsUnlessTheTaskNamesOne(t *testing.T) {
	router, argv := makeRouter(t, func(c *agent.Config) { c.Claude.MaxSteps = 7 })

	run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: agent.UseClaude})
	if err != nil {
		t.Fatal(err)
	}
	drain(t, run)
	if got := getFlag(argv[agent.UseClaude](), "--max-turns"); got != "7" {
		t.Errorf("--max-turns is %q, want 7", got)
	}

	run, err = router.StartRun(t.Context(), port.Task{Question: "q", Agent: agent.UseClaude, MaxSteps: 3})
	if err != nil {
		t.Fatal(err)
	}
	drain(t, run)
	if got := getFlag(argv[agent.UseClaude](), "--max-turns"); got != "3" {
		t.Errorf("--max-turns is %q, want 3", got)
	}
}

// Closing the router closes every agent, and none of them takes work afterwards.
func TestNoAgentTakesWorkAfterTheRouterIsClosed(t *testing.T) {
	router, _ := makeRouter(t, nil)
	if err := router.Close(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{agent.UseClaude, agent.UseCodex, agent.UseAntigravity} {
		if _, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: name}); err == nil {
			t.Errorf("%s took work after the router was closed", name)
		}
	}
}
