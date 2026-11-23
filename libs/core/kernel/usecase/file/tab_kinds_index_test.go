package file_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	pluginentries "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/markdown"
)

// countingReaders counts the files its readers read or stat.
type countingReaders struct {
	vaultport.VaultReaders
	reads, stats *atomic.Int64
}

func (c countingReaders) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := c.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return countingReader{VaultReader: reader, reads: c.reads, stats: c.stats}, nil
}

type countingReader struct {
	vaultport.VaultReader
	reads, stats *atomic.Int64
}

func (c countingReader) Read(ctx context.Context, path string) ([]byte, error) {
	c.reads.Add(1)
	return c.VaultReader.Read(ctx, path)
}

func (c countingReader) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	c.stats.Add(1)
	return c.VaultReader.StatFile(ctx, path)
}

// countingHeads counts the questions put to the index.
type countingHeads struct {
	heads interface {
		GetHeads(context.Context, domain.VaultID, []string) (map[string]frontmatter.Head, error)
	}
	calls atomic.Int64
}

func (c *countingHeads) GetHeads(ctx context.Context, id domain.VaultID, paths []string) (map[string]frontmatter.Head, error) {
	c.calls.Add(1)
	return c.heads.GetHeads(ctx, id, paths)
}

// shipped lists the tab kinds the shipped plugins declare, by the conditions they use.
type shipped []pluginentries.TabKindEntry

func (s shipped) ListTabKinds() []pluginentries.TabKindEntry { return s }

type markdownTypes struct{}

func (markdownTypes) GetContentType(path string) domain.ContentType {
	if strings.HasSuffix(path, ".md") {
		return "text/markdown"
	}
	return domain.UnknownContentType
}

func (markdownTypes) ListExtensions(domain.ContentType) []string { return []string{"md"} }

func newShippedChoice() *pluginentries.TabKindChoice {
	entry := func(plugin, kind, when string, rank domain.HandlerRank) pluginentries.TabKindEntry {
		return pluginentries.TabKindEntry{PluginID: plugin, TabKind: manifest.TabKind{Kind: kind, Title: kind, When: when, Rank: rank}}
	}
	return pluginentries.NewTabKindChoice(shipped{
		entry("org.example.editor", "note", "target.isNote", domain.RankAlternate),
		entry("org.example.decks", "deck", "target.noteType == 'deck'", domain.RankDefault),
		entry("org.example.search", "search", "target.noteType == 'search'", domain.RankDefault),
		entry("org.example.links", "link", "target.noteType == 'link' && has(target.properties.url)", domain.RankDefault),
		entry("org.example.media", "media", "has(target.properties.media)", domain.RankDefault),
		entry("org.example.rated", "rated", "has(target.properties.rank) && target.properties.rank > 2", domain.RankDefault),
	}.ListTabKinds, markdownTypes{}.GetContentType, nil)
}

func getTabKindNotes() map[string]string {
	return map[string]string{
		"plain.md":        "# plain\n",
		"deck.md":         "---\ntype: deck\n---\n# deck\n",
		"search.md":       "---\ntype: search\nquery: tide\n---\n",
		"link.md":         "---\ntype: link\nurl: https://tides.example/mill\nmedia: plugin://org.example.links/tide.mp4\n---\n",
		"linkless.md":     "---\ntype: link\n---\n",
		"rated.md":        "---\nrank: 3\n---\n",
		"low.md":          "---\nrank: 1\n---\n",
		"empty.md":        "---\n---\n# empty\n",
		"broken.md":       "---\ntype: [deck\nurl: x: y\n---\n# broken\n",
		"folder/inner.md": "---\ntype: deck\nrank: 2.5\n---\n",
		"picture.png":     "pixels",
	}
}

type tabKindWorld struct {
	db     *index.DB
	vault  domain.Vault
	reads  *atomic.Int64
	stats  *atomic.Int64
	heads  *countingHeads
	choose *pluginentries.TabKindChoice
}

func newTabKindWorld(t *testing.T, notes map[string]string) tabKindWorld {
	t.Helper()
	db, err := index.Open(t.Context(), indexfile.Path(t), index.Unsynchronized())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	v := vaulttest.NewVault(t, notes)
	scan := indexing.NewScan(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), testsupport.EmptySourceBacklog{})
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return tabKindWorld{
		db: db, vault: v, reads: new(atomic.Int64), stats: new(atomic.Int64),
		heads: &countingHeads{heads: db.NoteQueries()}, choose: newShippedChoice(),
	}
}

func (w tabKindWorld) fromFiles(t *testing.T, paths []string) map[string]string {
	t.Helper()
	reader, err := vaultreader.Readers{}.Open(w.vault)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, path := range paths {
		if !strings.HasSuffix(path, ".md") {
			continue
		}
		raw, err := reader.Read(t.Context(), path)
		if err != nil {
			continue
		}
		parsed := markdown.Parse(domain.FileStat{Path: path, Kind: domain.KindNote}, raw)
		if kind := w.choose.Choose(path, parsed.Type, parsed.Frontmatter); kind != "" {
			out[path] = kind
		}
	}
	return out
}

func (w tabKindWorld) listTabKinds(t *testing.T, paths []string) map[string]string {
	t.Helper()
	kinds, err := indexnote.NewTabKinds(w.heads, w.choose).ListTabKinds(t.Context(), w.vault, paths)
	if err != nil {
		t.Fatal(err)
	}
	return kinds
}

func sameKinds(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tab kinds %v, want %v", got, want)
	}
	for path, kind := range want {
		if got[path] != kind {
			t.Errorf("%s opens in %q, want %q (all: %v)", path, got[path], kind, got)
		}
	}
}

func TestTheTabKindsOfIndexedNotesAreTheOnesTheirFilesGive(t *testing.T) {
	notes := getTabKindNotes()
	w := newTabKindWorld(t, notes)
	paths := make([]string, 0, len(notes))
	for path := range notes {
		paths = append(paths, path)
	}

	got := w.listTabKinds(t, paths)

	want := w.fromFiles(t, paths)
	if want["deck.md"] != "deck" || want["search.md"] != "search" || want["link.md"] != "link" ||
		want["rated.md"] != "rated" || want["broken.md"] != "note" || want["plain.md"] != "note" {
		t.Fatalf("the files give %v, which does not cover the shipped kinds", want)
	}
	sameKinds(t, got, want)
}

func TestTheTabKindsOfAFolderAreChosenWithoutReadingAFile(t *testing.T) {
	w := newTabKindWorld(t, getTabKindNotes())
	readers := countingReaders{VaultReaders: vaultreader.Readers{}, reads: w.reads, stats: w.stats}
	folder := file.NewListFolder(file.NewGetFolderEntries(readers, w.db.NoteQueries()), indexnote.NewTabKinds(w.heads, w.choose))

	listing, err := folder.Execute(t.Context(), w.vault, "")
	if err != nil {
		t.Fatal(err)
	}

	if listing.TabKinds["deck.md"] != "deck" || listing.TabKinds["link.md"] != "link" {
		t.Errorf("tab kinds %v", listing.TabKinds)
	}
	if w.reads.Load() != 0 || w.stats.Load() != 0 {
		t.Errorf("%d files read and %d stat, want none", w.reads.Load(), w.stats.Load())
	}
}

func TestANoteTheIndexDoesNotHoldYetHasNoTabKindAndIsNotRead(t *testing.T) {
	w := newTabKindWorld(t, getTabKindNotes())
	if err := os.WriteFile(filepath.Join(w.vault.Path, "fresh.md"), []byte("---\ntype: deck\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	readers := countingReaders{VaultReaders: vaultreader.Readers{}, reads: w.reads, stats: w.stats}
	folder := file.NewListFolder(file.NewGetFolderEntries(readers, w.db.NoteQueries()), indexnote.NewTabKinds(w.heads, w.choose))

	listing, err := folder.Execute(t.Context(), w.vault, "")
	if err != nil {
		t.Fatal(err)
	}

	if _, isListed := listing.TabKinds["fresh.md"]; isListed {
		t.Errorf("the note the index lacks opens in %q, want no tab kind", listing.TabKinds["fresh.md"])
	}
	if listing.TabKinds["deck.md"] != "deck" {
		t.Errorf("the indexed deck opens in %q", listing.TabKinds["deck.md"])
	}
	if got := w.listTabKinds(t, []string{"fresh.md", "gone.md"}); len(got) != 0 {
		t.Errorf("tab kinds %v, want none", got)
	}
	if w.reads.Load() != 0 || w.stats.Load() != 0 {
		t.Errorf("%d files read and %d stat, want none", w.reads.Load(), w.stats.Load())
	}
}

func TestFiftyPathsAreOneQuestionToTheIndex(t *testing.T) {
	notes := map[string]string{}
	paths := make([]string, 0, 50)
	for i := range 50 {
		path := filepath.ToSlash(filepath.Join("batch", string(rune('a'+i%26))+string(rune('a'+i/26))+".md"))
		notes[path] = "---\ntype: deck\n---\n"
		paths = append(paths, path)
	}
	w := newTabKindWorld(t, notes)

	got := w.listTabKinds(t, paths)

	if len(got) != 50 {
		t.Errorf("%d tab kinds, want 50", len(got))
	}
	if w.heads.calls.Load() != 1 {
		t.Errorf("%d questions to the index, want 1", w.heads.calls.Load())
	}
}

func TestPropertyConditionsUseTheIndexedProperties(t *testing.T) {
	w := newTabKindWorld(t, getTabKindNotes())

	got := w.listTabKinds(t, []string{"rated.md", "low.md", "folder/inner.md"})

	if got["rated.md"] != "rated" {
		t.Errorf("rank 3 opens in %q, want rated", got["rated.md"])
	}
	if got["low.md"] != "note" {
		t.Errorf("rank 1 opens in %q, want note", got["low.md"])
	}
	if got["folder/inner.md"] != "deck" {
		t.Errorf("the deck opens in %q, want deck", got["folder/inner.md"])
	}
}
