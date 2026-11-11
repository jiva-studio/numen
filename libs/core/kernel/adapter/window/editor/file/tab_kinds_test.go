package file_test

import (
	"maps"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	pluginentries "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
)

// window is a window showing one vault, with nothing writing and no plugins to tell.
type window struct{ vault domain.Vault }

func (w window) GetCurrentVault() domain.Vault { return w.vault }
func (window) PublishChange(...string)         {}
func (window) EmitFileEvent(string)            {}

// tabKinds is the tab kinds of the enabled plugins of a window.
type tabKinds []pluginentries.TabKindEntry

func (k tabKinds) ListTabKinds() []pluginentries.TabKindEntry { return k }

// notesOnly gives a note its content type and no other file one.
type notesOnly struct{}

func (notesOnly) GetContentType(path string) domain.ContentType {
	if strings.HasSuffix(path, ".md") {
		return "text/markdown"
	}
	return domain.UnknownContentType
}

func (notesOnly) ListExtensions(domain.ContentType) []string { return []string{"md"} }

func newHandler(t *testing.T, notes map[string]string, kinds tabKinds) *filewindow.Handler {
	t.Helper()
	open := window{vault: testsupport.NewVault(t, notes)}
	choose, err := tabkind.NewChoose(kinds, notesOnly{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	readers := vaultreader.Readers{}
	fileKinds := file.NewGetFileKinds(readers, nil)
	return filewindow.NewHandler(
		filewindow.NewFolders(
			open, open,
			file.NewListFolder(file.NewGetFolderEntries(readers, nil), file.NewListTabKinds(fileKinds, choose)),
			file.NewListFileKinds(fileKinds, choose), nil,
		),
		filewindow.NewChanges(open, open, nil, nil, nil),
	)
}

func TestAPathNamesTheKindOfTabItsNoteOpensIn(t *testing.T) {
	handler := newHandler(t, map[string]string{
		"Entropy.md":    "# Entropy\n",
		"links/tide.md": "---\ntype: link\nurl: https://tides.example/mill\nmedia: plugin://org.example.links/tide.mp4\n---\n",
		"Physics.epub":  "an epub\n",
	}, tabKinds{
		{PluginID: "org.example.editor", TabKind: manifest.TabKind{Kind: "note", Title: "Note", Icon: "", When: "target.isNote", Rank: domain.RankAlternate}},
		{PluginID: "org.example.viewer", TabKind: manifest.TabKind{Kind: "media", Title: "Media", Icon: "", When: "has(target.properties.media)", Rank: domain.RankDefault}},
	})

	answer, err := handler.ListFileKinds(t.Context(), connect.NewRequest(&v1.ListFileKindsRequest{
		Paths: []string{"Entropy.md", "links/tide.md", "Physics.epub"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]*v1.FileKind{}
	for _, one := range answer.Msg.GetKinds() {
		kinds[one.GetPath()] = one
	}

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

func TestAListingOfAFolderNamesTheKindOfTabEachNoteOpensIn(t *testing.T) {
	handler := newHandler(t, map[string]string{
		"links/tide.md":  "---\ntype: link\nurl: https://tides.example/mill\nmedia: plugin://org.example.links/tide.mp4\n---\n",
		"links/plain.md": "# Plain\n",
		"links/map.png":  "pixels",
	}, tabKinds{
		{PluginID: "org.example.viewer", TabKind: manifest.TabKind{Kind: "media", Title: "Media", When: "has(target.properties.media)", Rank: domain.RankDefault}},
	})

	answer, err := handler.ListFiles(t.Context(), connect.NewRequest(&v1.ListFilesRequest{Path: "links"}))
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
