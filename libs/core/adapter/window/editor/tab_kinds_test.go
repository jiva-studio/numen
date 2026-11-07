package editor_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/tabkind"
)

// tabKinds is the tab kinds of the enabled plugins of a window.
type tabKinds []pluginport.PluginTabKindEntry

func (k tabKinds) ListTabKinds() []pluginport.PluginTabKindEntry { return k }

func TestAPathNamesTheKindOfTabItsNoteOpensIn(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Entropy.md":    "# Entropy\n",
		"links/tide.md": "---\ntype: link\nurl: https://tides.example/mill\nmedia: plugin://org.example.links/tide.mp4\n---\n",
		"Physics.epub":  "an epub\n",
	})
	f.read(t)
	f.installation.Services.SetTabKindChoice(tabkind.NewChoose(tabKinds{
		{PluginID: "org.example.editor", TabKind: manifest.TabKind{Kind: "note", Title: "Note", Icon: "", When: "target.isNote", Rank: domain.RankAlternate}},
		{PluginID: "org.example.viewer", TabKind: manifest.TabKind{Kind: "media", Title: "Media", Icon: "", When: "has(target.properties.media)", Rank: domain.RankDefault}},
	}, nil))

	kinds := getKindsByPath(t, f, "Entropy.md", "links/tide.md", "Physics.epub")

	if got := kinds["Entropy.md"].GetTabKind(); got != "note" {
		t.Errorf("the ordinary note opens in %q, want note", got)
	}
	if got := kinds["links/tide.md"].GetTabKind(); got != "media" {
		t.Errorf("the note with media opens in %q, want media", got)
	}
	if got := kinds["Physics.epub"].GetTabKind(); got != "" {
		t.Errorf("the book opens in %q, want no tab kind", got)
	}
}
