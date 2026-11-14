package connection_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

const agentPlugin = "org.example.agent"

func (f *fixture) getGrantRevision(t *testing.T, pluginID string) (string, bool) {
	t.Helper()
	revision, isGranted, err := connection.NewGetGrantRevision(connection.NewDeps(f.repository, f.types, f.keys, f.uses, none.Detected{})).Execute(pluginID)
	if err != nil {
		t.Fatal(err)
	}
	return revision, isGranted
}

func TestTheGrantRevisionChangesWithTheConnectionsTheGrantReaches(t *testing.T) {
	f := newFixture(t)
	f.uses.grants = []connectiondomain.Grant{{PluginID: agentPlugin, PluginName: "Agent", Operations: []string{"llm.complete"}}}
	work := f.createOpenRouter(t, "Work")
	first, isGranted := f.getGrantRevision(t, agentPlugin)
	if !isGranted || first == "" {
		t.Fatalf("revision = %q, granted %v", first, isGranted)
	}

	if _, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: work.ID, Revision: work.Revision, Name: "Work 2"}); err != nil {
		t.Fatal(err)
	}
	updated, _ := f.getGrantRevision(t, agentPlugin)
	f.createCompatible(t, "Ollama", "http://localhost:11434/v1", "")
	added, _ := f.getGrantRevision(t, agentPlugin)

	if updated == first || added == updated {
		t.Errorf("revisions %q, %q, %q, want each change to give another", first, updated, added)
	}
}

func TestAPluginWithoutAGrantHasNoGrantRevision(t *testing.T) {
	f := newFixture(t)
	f.createOpenRouter(t, "Work")

	if revision, isGranted := f.getGrantRevision(t, agentPlugin); isGranted || revision != "" {
		t.Errorf("revision = %q, granted %v, want none", revision, isGranted)
	}
}
