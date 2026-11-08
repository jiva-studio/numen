package manifestlist

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

func TestListTabKinds_ReadsTheManifestsOfEnabledPluginsInOrder(t *testing.T) {
	deck := manifest.TabKind{Kind: "deck", Title: "Deck", Icon: "", When: "target.noteType == 'deck'", Rank: domain.RankDefault}
	note := manifest.TabKind{Kind: "note", Title: "Note", Icon: "", When: "target.isNote", Rank: domain.RankAlternate}
	host := &manifestHost{
		plugins: []pluginport.DiscoveredPlugin{
			{Manifest: manifest.Manifest{ID: "org.example.cards", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{TabKinds: []manifest.TabKind{deck}}}},
			{Manifest: manifest.Manifest{ID: "org.example.off", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{TabKinds: []manifest.TabKind{deck}}}},
			{Manifest: manifest.Manifest{ID: "org.example.editor", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{TabKinds: []manifest.TabKind{note}}}},
		},
		disabled: map[string]bool{"org.example.off": true},
	}

	entries := Open(host).ListTabKinds()

	if len(entries) != 2 || entries[0].PluginID != "org.example.cards" || entries[0].TabKind != deck ||
		entries[1].PluginID != "org.example.editor" || entries[1].TabKind != note {
		t.Errorf("got %+v", entries)
	}
}
