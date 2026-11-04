package commandbridge

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func TestListCommands_CarriesThePartsOfTheDeclaringPlugin(t *testing.T) {
	command := []manifest.Command{{ID: "tree.open", Title: "Open file tree"}}
	host := &commandHost{plugins: []pluginport.DiscoveredPlugin{
		{Manifest: manifest.Manifest{ID: "md.numen.tree", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{Commands: command}}},
		{Manifest: manifest.Manifest{ID: "md.numen.job", Backend: manifest.Backend{Entrypoint: "bin/job"}, Contributes: manifest.Contributes{Commands: command}}},
		{Manifest: manifest.Manifest{ID: "md.numen.empty", Contributes: manifest.Contributes{Commands: command}}},
	}}

	entries := Open(host).ListCommands()

	if len(entries) != 3 {
		t.Fatalf("got %+v", entries)
	}
	parts := map[string][2]bool{}
	for _, one := range entries {
		parts[one.PluginID] = [2]bool{one.HasBackend, one.HasFrontend}
	}
	want := map[string][2]bool{"md.numen.tree": {false, true}, "md.numen.job": {true, false}, "md.numen.empty": {false, false}}
	for id, got := range parts {
		if got != want[id] {
			t.Errorf("plugin %s has parts %v, want %v", id, got, want[id])
		}
	}
}

// suspendedHost reports every plugin it holds as suspended after a failed start.
type suspendedHost struct{ *commandHost }

func (h suspendedHost) GetPluginInfos() []pluginport.PluginInfo {
	infos := h.commandHost.GetPluginInfos()
	for i := range infos {
		infos[i].Failure = &pluginport.PluginFailure{PluginID: infos[i].ID, Reason: "Failed to start"}
	}
	return infos
}

func TestListCommands_MarksTheCommandsOfASuspendedPlugin(t *testing.T) {
	command := []manifest.Command{{ID: "deck.create", Title: "Create deck"}}
	host := suspendedHost{&commandHost{plugins: []pluginport.DiscoveredPlugin{
		{Manifest: manifest.Manifest{ID: "md.numen.cards", Backend: manifest.Backend{Entrypoint: "bin/cards"}, Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{Commands: command}}},
	}}}

	entries := Open(host).ListCommands()

	if len(entries) != 1 || !entries[0].IsSuspended || !entries[0].HasBackend {
		t.Fatalf("got %+v", entries)
	}
}
