package tabkind_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/tabkind"
)

// declared is the tab kinds of the enabled plugins, in the order of their ids.
type declared []pluginport.PluginTabKindEntry

func (d declared) ListTabKinds() []pluginport.PluginTabKindEntry { return d }

// notes is a content type table that knows notes only.
type notes struct{}

func (notes) GetContentType(string) domain.ContentType   { return "text/markdown" }
func (notes) ListExtensions(domain.ContentType) []string { return []string{"md"} }

func entry(pluginID, kind, when string, rank domain.HandlerRank) pluginport.PluginTabKindEntry {
	return pluginport.PluginTabKindEntry{
		PluginID: pluginID,
		TabKind:  manifest.TabKind{Kind: kind, Title: kind, Icon: "", When: when, Rank: rank},
	}
}

var kinds = declared{
	entry("org.example.cards", "deck", "target.noteType == 'deck'", domain.RankDefault),
	entry("org.example.editor", "note", "target.isNote", domain.RankAlternate),
	entry("org.example.viewer", "media", "has(target.properties.media) || target.noteType == 'link'", domain.RankDefault),
	entry("org.example.zines", "zine", "target.noteType == 'deck'", domain.RankDefault),
	entry("org.example.files", "files", "", domain.RankDefault),
}

func TestChoose_OpensANoteInTheKindWhoseConditionHolds(t *testing.T) {
	choose := tabkind.NewChoose(kinds, notes{})

	cases := []struct {
		name       string
		path       string
		noteType   domain.NoteType
		properties map[string]any
		want       string
	}{
		{"an ordinary note", "plain.md", domain.TypeNote, map[string]any{}, "note"},
		{"a link note without media", "links/tide.md", "link", map[string]any{"url": "https://tides.example/mill"}, "media"},
		{"an ordinary note with media", "talk.md", domain.TypeNote, map[string]any{"media": "plugin://org.example.links/talk.mp4"}, "media"},
		{"a deck that two kinds take, chosen by the lower plugin id", "decks/verbs.md", "deck", map[string]any{}, "deck"},
	}
	for _, one := range cases {
		if got := choose.Execute(one.path, one.noteType, one.properties); got != one.want {
			t.Errorf("%s: kind = %q, want %q", one.name, got, one.want)
		}
	}
}

func TestChoose_GivesNoKindWhereNoConditionHolds(t *testing.T) {
	choose := tabkind.NewChoose(declared{entry("org.example.cards", "deck", "target.noteType == 'deck'", domain.RankDefault)}, notes{})

	if got := choose.Execute("plain.md", domain.TypeNote, nil); got != "" {
		t.Errorf("kind = %q, want none", got)
	}
}

func TestChoose_GivesNoKindWithoutPlugins(t *testing.T) {
	if got := tabkind.NewChoose(nil, nil).Execute("plain.md", domain.TypeNote, nil); got != "" {
		t.Errorf("kind = %q, want none", got)
	}
}
