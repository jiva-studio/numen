package commandbridge

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func TestGetParts_ReadsTheManifestOfThePlugin(t *testing.T) {
	command := []manifest.Command{{ID: "tree.open", Title: "Open file tree"}}
	host := &commandHost{plugins: []pluginport.DiscoveredPlugin{
		{Manifest: manifest.Manifest{ID: "md.numen.tree", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{Commands: command}}},
		{Manifest: manifest.Manifest{ID: "md.numen.job", Backend: manifest.Backend{Entrypoint: "bin/job"}, Contributes: manifest.Contributes{Commands: command}}},
		{Manifest: manifest.Manifest{ID: "md.numen.empty", Contributes: manifest.Contributes{Commands: command}}},
	}}

	commands := Open(host)

	want := map[string]pluginport.PluginParts{
		"md.numen.tree":    {HasFrontend: true},
		"md.numen.job":     {HasBackend: true},
		"md.numen.empty":   {},
		"md.numen.missing": {},
	}
	for id, parts := range want {
		if got := commands.GetParts(id); got != parts {
			t.Errorf("plugin %s has parts %+v, want %+v", id, got, parts)
		}
	}
}

// failedHost reports the last start of every plugin it holds as failed.
type failedHost struct{ *commandHost }

func (h failedHost) GetPluginInfos() []pluginport.PluginInfo {
	infos := h.commandHost.GetPluginInfos()
	for i := range infos {
		infos[i].Failure = &pluginport.PluginFailure{PluginID: infos[i].ID, Reason: "Failed to start"}
	}
	return infos
}

func TestGetParts_KeepsThePartsOfAPluginWhoseLastStartFailed(t *testing.T) {
	host := failedHost{&commandHost{plugins: []pluginport.DiscoveredPlugin{
		{Manifest: manifest.Manifest{ID: "md.numen.cards", Backend: manifest.Backend{Entrypoint: "bin/cards"}, Frontend: &manifest.Frontend{}}},
	}}}

	parts := Open(host).GetParts("md.numen.cards")

	if !parts.HasBackend || !parts.HasFrontend {
		t.Fatalf("got %+v", parts)
	}
}
