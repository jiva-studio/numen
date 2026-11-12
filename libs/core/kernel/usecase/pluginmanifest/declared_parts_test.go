package pluginmanifest

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestGetParts_ReadsTheManifestOfThePlugin(t *testing.T) {
	command := []manifest.Command{{ID: "tree.open", Title: "Open file tree"}}
	host := &manifestHost{plugins: []manifest.Discovered{
		{Manifest: manifest.Manifest{ID: "md.numen.tree", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{Commands: command}}},
		{Manifest: manifest.Manifest{ID: "md.numen.job", Backend: manifest.Backend{Entrypoint: "bin/job"}, Contributes: manifest.Contributes{Commands: command}}},
		{Manifest: manifest.Manifest{ID: "md.numen.empty", Contributes: manifest.Contributes{Commands: command}}},
	}}

	commands := New(host)

	want := map[string]plugin.Parts{
		"md.numen.tree":    {HasFrontend: true},
		"md.numen.job":     {HasBackend: true},
		"md.numen.empty":   {},
		"md.numen.missing": {},
	}
	for id, parts := range want {
		if got := commands.GetParts(id); !reflect.DeepEqual(got, parts) {
			t.Errorf("plugin %s has parts %+v, want %+v", id, got, parts)
		}
	}
}

// suspendedHost reports every plugin it holds as suspended after a failed start.
type suspendedHost struct{ *manifestHost }

func (h suspendedHost) IsPluginSuspended(string) bool { return true }

func TestGetParts_ReportsASuspendedPlugin(t *testing.T) {
	host := suspendedHost{&manifestHost{plugins: []manifest.Discovered{
		{Manifest: manifest.Manifest{ID: "md.numen.cards", Backend: manifest.Backend{Entrypoint: "bin/cards"}, Frontend: &manifest.Frontend{}}},
	}}}

	parts := New(host).GetParts("md.numen.cards")

	if !parts.IsSuspended || !parts.HasBackend || !parts.HasFrontend {
		t.Fatalf("got %+v", parts)
	}
}

func TestGetParts_ReadsNoListingOfThePlugins(t *testing.T) {
	host := &manifestHost{plugins: []manifest.Discovered{
		{Manifest: manifest.Manifest{ID: "md.numen.cards", Backend: manifest.Backend{Entrypoint: "bin/cards"}}},
		{Manifest: manifest.Manifest{ID: "md.numen.tree", Frontend: &manifest.Frontend{}}},
	}}

	parts := New(host).GetParts("md.numen.cards")

	if !parts.HasBackend || parts.IsSuspended {
		t.Errorf("got %+v", parts)
	}
	if host.infoReads != 0 {
		t.Errorf("the parts of one plugin read the listing of all plugins %d times", host.infoReads)
	}
}

func TestGetParts_ReadsTheNameAndTheTextKindsOfThePlugin(t *testing.T) {
	host := &manifestHost{plugins: []manifest.Discovered{
		{Manifest: manifest.Manifest{
			ID: "md.numen.ears", Name: "Transcription", Backend: manifest.Backend{Entrypoint: "bin/ears"},
			Contributes: manifest.Contributes{TextKinds: []manifest.TextKind{{Kind: "transcript", Title: "Transcript"}}},
		}},
	}}

	parts := New(host).GetParts("md.numen.ears")

	if parts.Name != "Transcription" || parts.TextKinds["transcript"] != "Transcript" {
		t.Fatalf("got %+v", parts)
	}
}
