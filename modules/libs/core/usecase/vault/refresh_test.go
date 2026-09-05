package vault_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// refreshing is a vault already scanned once, and the use case that brings named
// notes up to date afterwards.
func refreshing(t *testing.T, notes map[string]string) (vaults.Refresh, *container.Index, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db := openIndex(t)
	if _, err := scanner(filesystem.VaultReaders{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return vaults.Refresh{
		Readers: filesystem.VaultReaders{},
		Notes:   db.Notes(),
		Known:   db.SourcesKnown(),
		Sources: db.Sources(),
	}, db, v
}

// passages is what the index itself answers with, before anything opens the
// file a passage names.
func passages(t *testing.T, db *container.Index, v domain.Vault, query string) []domain.Passage {
	t.Helper()
	found, err := db.Passages().Lexical(
		t.Context(), v.ID, query, []domain.SourceKind{domain.KindBook}, 10, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func titles(t *testing.T, db *container.Index, v domain.Vault, query string) []string {
	t.Helper()
	matches, err := db.NoteIndex().Search(t.Context(), v.ID, query, 10)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.Title)
	}
	slices.Sort(out)
	return out
}

// TestARefreshedNoteIsWhatIsOnDisk.
func TestARefreshedNoteIsWhatIsOnDisk(t *testing.T) {
	t.Parallel()
	refresh, db, v := refreshing(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})

	if err := os.WriteFile(filepath.Join(v.Path, "Note.md"),
		[]byte("---\ntitle: Renamed\n---\n\n# Renamed\n\nentropy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := refresh.Execute(t.Context(), v, []string{"Note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Indexed, []string{"Note.md"}) {
		t.Errorf("indexed %v", res.Indexed)
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Renamed"}) {
		t.Errorf("the index holds %v", got)
	}
}

// TestADeletedNoteLeavesTheIndex. The vault is authoritative: what is not on
// disk is not in the index, and a note that stays behind is still searchable
// and opens nothing.
func TestADeletedNoteLeavesTheIndex(t *testing.T) {
	t.Parallel()
	refresh, db, v := refreshing(t, map[string]string{
		"Note.md":  "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
		"Other.md": "---\ntitle: Other\n---\n\n# Other\n\nentropy\n",
	})

	if err := os.Remove(filepath.Join(v.Path, "Note.md")); err != nil {
		t.Fatal(err)
	}
	res, err := refresh.Execute(t.Context(), v, []string{"Note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Removed, []string{"Note.md"}) {
		t.Errorf("removed %v", res.Removed)
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Other"}) {
		t.Errorf("the index holds %v", got)
	}
}

// TestARefreshDoesNotTakeABookForARemovedNote. A refresh asks the reader whether
// each path is still there and reads fs.ErrNotExist as "it was removed". A book
// the vault holds is there, and taking its text out of it is its own step.
func TestARefreshDoesNotTakeABookForARemovedNote(t *testing.T) {
	t.Parallel()
	refresh, db, v := refreshing(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})
	testsupport.WriteBook(t, v.Path, "library/A Book.epub")

	res, err := refresh.Execute(t.Context(), v, []string{"library/A Book.epub", "Note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) != 0 {
		t.Errorf("removed %v — the vault holds the book", res.Removed)
	}
	if !slices.Equal(res.Assets, []string{"library/A Book.epub"}) {
		t.Errorf("reported %v as sources of another kind", res.Assets)
	}
	if !slices.Equal(res.Indexed, []string{"Note.md"}) {
		t.Errorf("indexed %v", res.Indexed)
	}
	if !slices.Equal(res.Changed(), []string{"Note.md"}) {
		t.Errorf("the caller was told to look at %v", res.Changed())
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Note"}) {
		t.Errorf("the index holds %v", got)
	}
}

// TestABookThatWentLeavesTheIndex. A book is filed by kind and carries chunks
// of its own, and a row left behind goes on answering searches with a passage
// that opens nothing.
func TestABookThatWentLeavesTheIndex(t *testing.T) {
	t.Parallel()
	refresh, db, v := refreshing(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	})
	const book = "library/A Book.epub"
	testsupport.WriteBook(t, v.Path, book)
	if err := db.Sources().SaveExtraction(t.Context(), v.ID, port.SourceChunks{
		Source: port.Source{
			Fingerprint: domain.Fingerprint{Path: book, Kind: domain.KindBook, Size: 1, ModTime: 1},
			Hash:        "a-hash",
			Recipe:      "epub",
		},
		Chunks: []port.Chunk{{Start: 0, Length: 19, Text: "a reversible engine"}},
	}); err != nil {
		t.Fatal(err)
	}
	if len(passages(t, db, v, "reversible")) == 0 {
		t.Fatal("the book was not in the index to begin with")
	}

	if err := os.Remove(filepath.Join(v.Path, filepath.FromSlash(book))); err != nil {
		t.Fatal(err)
	}
	if _, err := refresh.Execute(t.Context(), v, []string{book}); err != nil {
		t.Fatal(err)
	}

	if found := passages(t, db, v, "reversible"); len(found) != 0 {
		t.Errorf("the index answers with %d passages of a book the vault does not hold", len(found))
	}
	held, err := db.SourcesKnown().Under(t.Context(), v.ID, "library")
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 0 {
		t.Errorf("the index still holds %v", held)
	}
}

// TestARefreshedBookThatWentIsStillGone. Nothing about a second kind of source
// changes what a path that is not there means.
func TestARefreshedBookThatWentIsStillGone(t *testing.T) {
	t.Parallel()
	refresh, _, v := refreshing(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})

	res, err := refresh.Execute(t.Context(), v, []string{"library/Never Existed.epub"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Removed, []string{"library/Never Existed.epub"}) {
		t.Errorf("removed %v", res.Removed)
	}
	if len(res.Assets) != 0 {
		t.Errorf("reported %v as sources the vault holds", res.Assets)
	}
}

// unreadableReaders answers one path with a failure that is not "gone" — a
// permission, a broken link, a device that went away.
type unreadableReaders struct {
	port.VaultReaders
	refuses string
}

func (u unreadableReaders) Open(vault domain.Vault) (port.VaultReader, error) {
	reader, err := u.VaultReaders.Open(vault)
	if err != nil {
		return nil, err
	}
	return unreadableReader{VaultReader: reader, refuses: u.refuses}, nil
}

type unreadableReader struct {
	port.VaultReader
	refuses string
}

func (u unreadableReader) Read(ctx context.Context, path string) ([]byte, error) {
	if path == u.refuses {
		return nil, fs.ErrPermission
	}
	return u.VaultReader.Read(ctx, path)
}

// TestOneUnreadableFileDoesNotCostTheRest. A watcher's event arrives once, so
// anything dropped alongside a failure is dropped until the next full scan.
func TestOneUnreadableFileDoesNotCostTheRest(t *testing.T) {
	t.Parallel()
	refresh, db, v := refreshing(t, map[string]string{
		"Locked.md": "---\ntitle: Locked\n---\n\n# Locked\n\nentropy\n",
		"Note.md":   "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})

	for _, name := range []string{"Locked.md", "Note.md"} {
		body := "---\ntitle: " + name + " edited\n---\n\n# Edited\n\nentropy\n"
		if err := os.WriteFile(filepath.Join(v.Path, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	refresh.Readers = unreadableReaders{VaultReaders: filesystem.VaultReaders{}, refuses: "Locked.md"}
	res, err := refresh.Execute(t.Context(), v, []string{"Locked.md", "Note.md"})
	if err != nil {
		t.Fatalf("one unreadable file ended the refresh: %v", err)
	}
	if !slices.Equal(res.Unreadable, []string{"Locked.md"}) {
		t.Errorf("unreadable %v", res.Unreadable)
	}
	if !slices.Equal(res.Indexed, []string{"Note.md"}) {
		t.Errorf("indexed %v — the file that could be read was not", res.Indexed)
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Locked", "Note.md edited"}) {
		t.Errorf("the index holds %v", got)
	}
}

// TestARefreshWritesInGroups. One event can name a whole folder, so the bound
// on what is held in memory and on the length of one write is the same one a
// scan keeps.
func TestARefreshWritesInGroups(t *testing.T) {
	t.Parallel()
	notes := map[string]string{}
	paths := make([]string, 0, 1200)
	for i := range 1200 {
		path := fmt.Sprintf("note-%04d.md", i)
		notes[path] = fmt.Sprintf("---\ntitle: Note %d\n---\n\n# Note %d\n\nentropy\n", i, i)
		paths = append(paths, path)
	}

	v := testsupport.NewVault(t, notes)
	written := &countingNotes{}
	refresh := vaults.Refresh{Readers: filesystem.VaultReaders{}, Notes: written}

	if _, err := refresh.Execute(t.Context(), v, paths); err != nil {
		t.Fatal(err)
	}
	if written.groups < 3 {
		t.Errorf("wrote %d notes in %d groups, want groups of at most 500",
			written.notes, written.groups)
	}
	for _, size := range written.sizes {
		if size > 500 {
			t.Errorf("a group held %d notes", size)
		}
	}
	if written.notes != 1200 {
		t.Errorf("wrote %d notes of 1200", written.notes)
	}
}

// countingNotes records how a repository was written to, and stores nothing.
type countingNotes struct {
	groups int
	notes  int
	sizes  []int
}

func (c *countingNotes) Save(_ context.Context, _ domain.VaultID, notes []domain.Note) error {
	c.groups++
	c.notes += len(notes)
	c.sizes = append(c.sizes, len(notes))
	return nil
}

func (c *countingNotes) Remove(context.Context, domain.VaultID, []string) error { return nil }

// TestANoteThatVanishesMidReadKeepsItsRow. A file that is briefly absent is
// what an editor saving through a temporary file looks like, and the save that
// follows arrives as its own event.
func TestANoteThatVanishesMidReadKeepsItsRow(t *testing.T) {
	t.Parallel()
	refresh, db, v := refreshing(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})

	refresh.Readers = vanishingReaders{VaultReaders: filesystem.VaultReaders{}, gone: "Note.md"}
	res, err := refresh.Execute(t.Context(), v, []string{"Note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) != 0 {
		t.Errorf("removed %v — it was there when it was looked at", res.Removed)
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Note"}) {
		t.Errorf("the index holds %v", got)
	}
}
