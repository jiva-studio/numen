package pluginmanifest

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestListTabKinds_ReadsTheManifestsOfEnabledPluginsInOrder(t *testing.T) {
	deck := manifest.TabKind{Kind: "deck", Title: "Deck", Icon: "", When: "target.noteType == 'deck'", Rank: domain.RankDefault}
	note := manifest.TabKind{Kind: "note", Title: "Note", Icon: "", When: "target.isNote", Rank: domain.RankAlternate}
	host := &manifestHost{
		plugins: []manifest.Discovered{
			{Manifest: manifest.Manifest{ID: "org.example.cards", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{TabKinds: []manifest.TabKind{deck}}}},
			{Manifest: manifest.Manifest{ID: "org.example.off", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{TabKinds: []manifest.TabKind{deck}}}},
			{Manifest: manifest.Manifest{ID: "org.example.editor", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{TabKinds: []manifest.TabKind{note}}}},
		},
		disabled: map[string]bool{"org.example.off": true},
	}

	entries := New(host).ListTabKinds()

	if len(entries) != 2 || entries[0].PluginID != "org.example.cards" || entries[0].TabKind != deck ||
		entries[1].PluginID != "org.example.editor" || entries[1].TabKind != note {
		t.Errorf("got %+v", entries)
	}
}

type listedHost struct {
	*manifestHost
	infos []plugin.Info
}

func (h listedHost) GetPluginInfos() []plugin.Info { return h.infos }

func TestListFrontendPluginIDs_ListsThePluginsWithAFrontendInOrder(t *testing.T) {
	host := listedHost{manifestHost: &manifestHost{}, infos: []plugin.Info{
		{ID: "org.example.cards", HasFrontend: true},
		{ID: "org.example.ocr"},
		{ID: "org.example.editor", HasFrontend: true},
	}}

	if got := New(host).ListFrontendPluginIDs(); !slices.Equal(got, []string{"org.example.cards", "org.example.editor"}) {
		t.Errorf("got %v", got)
	}
	if got := New(listedHost{manifestHost: &manifestHost{}}).ListFrontendPluginIDs(); got == nil || len(got) != 0 {
		t.Errorf("a host with no plugin lists %#v, want an empty list", got)
	}
}
