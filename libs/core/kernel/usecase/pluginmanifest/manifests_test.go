package pluginmanifest

import (
	"testing"

	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// manifestHost holds the plugins it is given, enabled or not, and counts the reads of its listing.
type manifestHost struct {
	plugins   []catalogmanifest.DiscoveredPlugin
	disabled  map[string]bool
	infoReads int
}

func (h *manifestHost) IsPluginEnabled(id string) bool { return !h.disabled[id] }

func (h *manifestHost) IsPluginSuspended(string) bool { return false }

func (h *manifestHost) GetPluginInfos() []plugin.Info {
	h.infoReads++
	infos := make([]plugin.Info, 0, len(h.plugins))
	for _, one := range h.plugins {
		infos = append(infos, plugin.Info{ID: one.Manifest.ID})
	}
	return infos
}

func (h *manifestHost) GetPlugin(id string) (catalogmanifest.DiscoveredPlugin, bool) {
	for _, one := range h.plugins {
		if one.Manifest.ID == id && !h.disabled[id] {
			return one, true
		}
	}
	return catalogmanifest.DiscoveredPlugin{}, false
}

func (h *manifestHost) ResolveCapability(string) (catalogmanifest.DiscoveredPlugin, error) {
	return catalogmanifest.DiscoveredPlugin{}, &plugin.UnavailableError{Reason: plugin.NotInstalled}
}

func (h *manifestHost) ResolveService(string) (catalogmanifest.DiscoveredPlugin, bool) {
	return catalogmanifest.DiscoveredPlugin{}, false
}

func (h *manifestHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}

func declaring(id string, commands ...string) catalogmanifest.DiscoveredPlugin {
	manifest := catalogmanifest.Manifest{ID: id}
	for _, one := range commands {
		manifest.Contributes.Commands = append(manifest.Contributes.Commands, catalogmanifest.Command{ID: one, Title: one})
	}
	return catalogmanifest.DiscoveredPlugin{Manifest: manifest}
}

func TestListCommands_ReadsManifestsWithoutStartingPlugins(t *testing.T) {
	host := &manifestHost{
		plugins: []catalogmanifest.DiscoveredPlugin{
			declaring("md.numen.a", "first", "second"),
			declaring("md.numen.b", "third"),
			declaring("md.numen.c", "fourth"),
		},
		disabled: map[string]bool{"md.numen.b": true},
	}

	entries := New(host).ListCommands()

	var ids []string
	for _, one := range entries {
		ids = append(ids, one.PluginID+"/"+one.Command.ID)
	}
	want := []string{"md.numen.a/first", "md.numen.a/second", "md.numen.c/fourth"}
	if len(ids) != len(want) || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Errorf("got %v, want %v", ids, want)
	}
}
