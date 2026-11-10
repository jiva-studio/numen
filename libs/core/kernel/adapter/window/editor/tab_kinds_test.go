package editor_test

import (
	"maps"
	"strings"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
)

// tabKinds is the tab kinds of the enabled plugins of a window.
type tabKinds []plugin.TabKindEntry

func (k tabKinds) ListTabKinds() []plugin.TabKindEntry { return k }

func TestAPathNamesTheKindOfTabItsNoteOpensIn(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Entropy.md":    "# Entropy\n",
		"links/tide.md": "---\ntype: link\nurl: https://tides.example/mill\nmedia: plugin://org.example.links/tide.mp4\n---\n",
		"Physics.epub":  "an epub\n",
	})
	f.read(t)
	choose, err := tabkind.NewChoose(tabKinds{
		{PluginID: "org.example.editor", TabKind: manifest.TabKind{Kind: "note", Title: "Note", Icon: "", When: "target.isNote", Rank: domain.RankAlternate}},
		{PluginID: "org.example.viewer", TabKind: manifest.TabKind{Kind: "media", Title: "Media", Icon: "", When: "has(target.properties.media)", Rank: domain.RankDefault}},
	}, notesOnly{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	editor.ServicesOf(f.installation).SetTabKindChoice(choose)

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

// notesOnly gives a note its content type and no other file one.
type notesOnly struct{}

func (notesOnly) GetContentType(path string) domain.ContentType {
	if strings.HasSuffix(path, ".md") {
		return "text/markdown"
	}
	return domain.UnknownContentType
}

func (notesOnly) ListExtensions(domain.ContentType) []string { return []string{"md"} }

func TestAListingOfAFolderNamesTheKindOfTabEachNoteOpensIn(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"links/tide.md":  "---\ntype: link\nurl: https://tides.example/mill\nmedia: plugin://org.example.links/tide.mp4\n---\n",
		"links/plain.md": "# Plain\n",
		"links/map.png":  "pixels",
	})
	f.read(t)
	choose, err := tabkind.NewChoose(tabKinds{
		{PluginID: "org.example.viewer", TabKind: manifest.TabKind{Kind: "media", Title: "Media", When: "has(target.properties.media)", Rank: domain.RankDefault}},
	}, notesOnly{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	editor.ServicesOf(f.installation).SetTabKindChoice(choose)

	answer, err := f.client.ListFiles(t.Context(), connect.NewRequest(&v1.ListFilesRequest{Path: "links"}))
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, one := range answer.Msg.GetEntries() {
		got[one.GetPath()] = one.GetTabKind()
	}
	if want := map[string]string{"links/tide.md": "media", "links/plain.md": "", "links/map.png": ""}; !maps.Equal(got, want) {
		t.Errorf("tab kinds %v, want %v", got, want)
	}
}
