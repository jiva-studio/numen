package container_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// The window requests its agent on every call, so a conversation started through one call ends on the
// plugin it was started in when another call finishes it.
func TestEditorAgent_EndsConversationStartedInEarlierCall(t *testing.T) {
	host, one, _ := twoProviders(t)
	assembly := newEditorServices(container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost((*container.PluginRoles)(host))}))

	runTask(t, assembly.GetAgent(), agent.Task{Question: "q", Agent: "alpha", Conversation: "c-1"})
	if err := assembly.GetAgent().EndConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(one.getFinished(), []string{"c-1"}) {
		t.Errorf("the plugin finished %v", one.getFinished())
	}
}

// Listing models for the settings populates the editor's agent router so a task started afterward
// goes straight to its plugin without listing the programs a second time.
func TestEditorSettingsShareAgentRouter(t *testing.T) {
	host, one, _ := twoProviders(t)
	assembly := newEditorServices(container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost((*container.PluginRoles)(host))}))

	if _, err := assembly.OpenGetSettings().Execute(); err != nil {
		t.Fatal(err)
	}
	listedBefore := len(one.getCommands())

	runTask(t, assembly.GetAgent(), agent.Task{Question: "q", Agent: "alpha", Conversation: "c-1"})

	if got := len(one.getCommands()); got != listedBefore {
		t.Errorf("GetAgentPrograms called %d times after settings, want %d", got, listedBefore)
	}
}

// When the first read of the settings returns before a slow agent provider lists its programs,
// watching the settings reports a change as soon as the listing finishes, and the next read
// includes the listed programs.
func TestEditorSettings_ReportsLatePrograms(t *testing.T) {
	host, one, _ := twoProviders(t)
	release := make(chan struct{})
	one.gate = release
	settingsFile := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(settingsFile, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	assembly := newEditorServices(
		container.NewConfig(container.Config{
			PluginHost: container.AdaptPluginHost((*container.PluginRoles)(host)),
			Paths:      container.Paths{SettingsPath: settingsFile},
		}),
	)

	changes, stop := assembly.OpenWatchSettings().Execute()
	defer stop()
	first, err := assembly.OpenGetSettings().Execute()
	if err != nil {
		t.Fatal(err)
	}
	if got := listAgentProgramNames(first.Models); !slices.Equal(got, []string{""}) {
		t.Fatalf("first read programs = %q, want [\"\"] before slow listing ends", got)
	}

	close(release)
	select {
	case batch := <-changes:
		if len(batch) == 0 || !slices.Equal(batch[0].Path, []string{"app", "agent", "use"}) {
			t.Fatalf("watch settings delivered %+v, want app.agent.use", batch)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("watch settings did not report when the slow program listing ended")
	}

	second, err := assembly.OpenGetSettings().Execute()
	if err != nil {
		t.Fatal(err)
	}
	if got := listAgentProgramNames(second.Models); !slices.Equal(got, []string{"alpha", "beta", ""}) {
		t.Fatalf("second read programs = %q, want [\"alpha\" \"beta\" \"\"]", got)
	}
}

func listAgentProgramNames(models []settings.Model) []string {
	var names []string
	for _, one := range models {
		if strings.Join(one.Path, ".") == "app.agent.use" {
			names = append(names, one.Name)
		}
	}
	return names
}
