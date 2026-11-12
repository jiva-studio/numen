package container_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

func openSettingsAgent(t *testing.T, host *container.PluginRoles, content string) agentport.Agent {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	agent := (container.NewConfig(container.Config{
		PluginHost: container.AdaptPluginHost(host),
		Paths:      container.Paths{SettingsPath: path},
	})).GetSettingsAgent()
	if agent == nil {
		t.Fatal("a host with providers has no agent")
	}
	return agent
}

func runTask(t *testing.T, agent agentport.Agent, task agentdomain.Task) {
	t.Helper()
	run, err := agent.StartRun(t.Context(), task)
	if err != nil {
		t.Fatal(err)
	}
	for range run.GetSteps() {
	}
}

// A task naming no program goes to the one agent.use names, with the command line and the model of
// that program's section.
func TestSettingsAgent_UnnamedProgramUsesSettingsProgram(t *testing.T) {
	host, one, two := twoProviders(t)
	agent := openSettingsAgent(t, host,
		`{"app": {"agent": {"use": "alpha", "alpha": {"command": ["/bin/alpha"], "model": "section-model"}}}}`)

	runTask(t, agent, agentdomain.Task{Question: "q", Conversation: "c-1"})

	requests := one.getRequestedAsks()
	if len(requests) != 1 || len(two.getRequestedAsks()) != 0 {
		t.Fatalf("the first plugin was asked %d times and the second %d", len(requests), len(two.getRequestedAsks()))
	}
	if got := requests[0]; got.GetAgent() != "alpha" || got.GetModel() != "section-model" ||
		!slices.Equal(got.GetCommand(), []string{"/bin/alpha"}) || got.GetPrompt() != "q" {
		t.Errorf("the plugin was asked %v", got)
	}

	if err := agent.EndConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(one.getFinishedIDs(), []string{"c-1"}) {
		t.Errorf("the plugin finished %v", one.getFinishedIDs())
	}
}

// Settings that leave the choice open send a task naming no program to the default of the plugins.
func TestSettingsAgent_UnsetProgramUsesDefault(t *testing.T) {
	host, one, _ := twoProviders(t)
	agent := openSettingsAgent(t, host, `{}`)

	runTask(t, agent, agentdomain.Task{Question: "q"})

	if requests := one.getRequestedAsks(); len(requests) != 1 || requests[0].GetAgent() != "alpha" {
		t.Errorf("the default plugin was asked %v", requests)
	}
}

// A task naming its program and model goes to that program with them.
func TestASettingsAgentTaskNamingAProgramGoesToIt(t *testing.T) {
	host, one, two := twoProviders(t)
	agent := openSettingsAgent(t, host, `{"app": {"agent": {"use": "alpha"}}}`)

	runTask(t, agent, agentdomain.Task{Question: "q", Agent: "beta", Model: "asked-model"})

	if requests := two.getRequestedAsks(); len(requests) != 1 || len(one.getRequestedAsks()) != 0 ||
		requests[0].GetAgent() != "beta" || requests[0].GetModel() != "asked-model" {
		t.Errorf("the second plugin was asked %v, the first %v", requests, one.getRequestedAsks())
	}
}

// Settings that turn the agent off refuse a task naming no program.
func TestSettingsAgent_RejectsTaskWhenAgentDisabled(t *testing.T) {
	host, one, two := twoProviders(t)
	agent := openSettingsAgent(t, host, `{"app": {"agent": {"use": ""}}}`)

	if _, err := agent.StartRun(t.Context(), agentdomain.Task{Question: "q"}); err == nil {
		t.Error("a task was taken with the agent turned off")
	}
	if len(one.getRequestedAsks())+len(two.getRequestedAsks()) != 0 {
		t.Error("a plugin was asked")
	}
}

func TestTheSettingsAgentWithoutAHostRunsNothing(t *testing.T) {
	agent := container.NewConfig(container.Config{}).GetSettingsAgent()

	if _, err := agent.StartRun(t.Context(), agentdomain.Task{Question: "q"}); err == nil {
		t.Error("a task was taken where no plugin runs")
	}
}
