package tabkind_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
)

// declared is the tab kinds of the enabled plugins, in the order of their ids.
type declared []plugin.TabKindEntry

func (d declared) ListTabKinds() []plugin.TabKindEntry { return d }

// notes is a content type table that knows notes only.
type notes struct{}

func (notes) GetContentType(string) domain.ContentType   { return "text/markdown" }
func (notes) ListExtensions(domain.ContentType) []string { return []string{"md"} }

func entry(pluginID, kind, when string, rank domain.HandlerRank) plugin.TabKindEntry {
	return plugin.TabKindEntry{
		PluginID: pluginID,
		TabKind:  manifest.TabKind{Kind: kind, Title: kind, Icon: "", When: when, Rank: rank},
	}
}

func newChoose(t *testing.T, kinds declared, config pluginport.WhenConfig) *tabkind.Choose {
	t.Helper()
	choose := tabkind.NewChoose(kinds, notes{}, config)
	return choose
}

func getKinds() declared {
	return declared{
		entry("org.example.cards", "deck", "target.noteType == 'deck'", domain.RankDefault),
		entry("org.example.editor", "note", "target.isNote", domain.RankAlternate),
		entry("org.example.viewer", "media", "has(target.properties.media) || target.noteType == 'link'", domain.RankDefault),
		entry("org.example.zines", "zine", "target.noteType == 'deck'", domain.RankDefault),
		entry("org.example.files", "files", "", domain.RankDefault),
	}
}

func TestChoose_OpensANoteInTheKindWhoseConditionMatches(t *testing.T) {
	choose := newChoose(t, getKinds(), nil)

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

func TestChoose_ReturnsNoKindWhenNoConditionMatches(t *testing.T) {
	choose := newChoose(t, declared{entry("org.example.cards", "deck", "target.noteType == 'deck'", domain.RankDefault)}, nil)

	if got := choose.Execute("plain.md", domain.TypeNote, nil); got != "" {
		t.Errorf("kind = %q, want none", got)
	}
}

// settingsOf is the config of when conditions it holds.
type settingsOf map[string]any

func (s settingsOf) GetWhenConfig() map[string]any { return s }

func TestChoose_ReadsTheSettingsAsConfig(t *testing.T) {
	kinds := declared{entry("org.example.viewer", "media", "target.isNote && config.org.example.viewer.isOn == true", domain.RankDefault)}
	on := settingsOf{"org": map[string]any{"example": map[string]any{"viewer": map[string]any{"isOn": true}}}}

	if got := newChoose(t, kinds, on).Execute("talk.md", domain.TypeNote, nil); got != "media" {
		t.Errorf("kind = %q, want media while its setting holds", got)
	}
}
