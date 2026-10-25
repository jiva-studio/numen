package vault_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// newRefresh is a vault already scanned once, and the use case that brings named
// notes up to date afterwards.
func newRefresh(t *testing.T, notes map[string]string) (vaults.Refresh, *container.Index, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db := openIndex(t)
	if _, err := scanner(vaultreader.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return vaults.NewRefresh(
		vaultreader.Readers{},
		db.Vaults(),
		db.Notes(),
		db.SourcesKnown(),
		db.Sources(), time.Now,
	), db, v
}

// passages is what the index itself answers with, before anything opens the
// file a passage names.
func passages(t *testing.T, db *container.Index, v domain.Vault, query string) []domain.Passage {
	t.Helper()
	found, err := db.Passages().FindLexical(
		t.Context(), v.ID, query, []domain.SourceKind{domain.KindBook}, 10, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func titles(t *testing.T, db *container.Index, v domain.Vault, query string) []string {
	t.Helper()
	paths := searchNotes(t, db, v, query)
	named, err := db.Queries().GetNotes(t.Context(), v.ID, paths)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, named[path].Title)
	}
	slices.Sort(out)
	return out
}

// TestANoteIsLevelledInAVaultNothingHasWalked. A note is filed under the
// vault's row in the index, and a vault nothing has walked has none. A person
// who writes in the moment a vault opens is racing the first walk, and their
// note is findable either way.
func TestANoteIsLevelledInAVaultNothingHasWalked(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})
	db := openIndex(t)
	refresh := vaults.NewRefresh(
		vaultreader.Readers{}, db.Vaults(), db.Notes(), db.SourcesKnown(), db.Sources(), time.Now)

	if _, err := refresh.Execute(t.Context(), v, []string{"Note.md"}); err != nil {
		t.Fatal(err)
	}
	if found := titles(t, db, v, "entropy"); !slices.Equal(found, []string{"Note"}) {
		t.Errorf("the note is found as %v", found)
	}
}

// TestARefreshedNoteIsWhatIsOnDisk.
func TestARefreshedNoteIsWhatIsOnDisk(t *testing.T) {
	t.Parallel()
	refresh, db, v := newRefresh(t, map[string]string{
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
func TestADeletedNotePreservesTheIndex(t *testing.T) {
	t.Parallel()
	refresh, db, v := newRefresh(t, map[string]string{
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
	refresh, db, v := newRefresh(t, map[string]string{
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
	if !slices.Equal(res.GetChangedPaths(), []string{"Note.md"}) {
		t.Errorf("the caller received to look at %v", res.GetChangedPaths())
	}
	if got := titles(t, db, v, "entropy"); !slices.Equal(got, []string{"Note"}) {
		t.Errorf("the index holds %v", got)
	}
}

// TestABookThatWentLeavesTheIndex. A book is filed by kind and carries chunks
// that opens nothing.
func TestABookThatWentPreservesTheIndex(t *testing.T) {
	t.Parallel()
	refresh, db, v := newRefresh(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	})
	const book = "library/A Book.epub"
	testsupport.WriteBook(t, v.Path, book)
	if err := db.Sources().SaveExtraction(t.Context(), v.ID, domain.SourceChunks{
		Source: domain.Source{
			Fingerprint: domain.Fingerprint{Path: book, Kind: domain.KindBook, Size: 1, ModTime: time.Unix(0, 1)},
			Hash:        "a-hash",
			Recipe:      "epub",
		},
		Chunks: []domain.Chunk{{Start: 0, Length: 19, Text: "a reversible engine"}},
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
	target, err := db.SourcesKnown().GetSourcesUnder(t.Context(), v.ID, "library")
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 0 {
		t.Errorf("the index still holds %v", target)
	}
}

// TestARefreshedBookThatWentIsStillGone. Nothing about a second kind of source
// changes what a path that is not there means.
func TestARefreshedBookThatWentIsStillMissing(t *testing.T) {
	t.Parallel()
	refresh, _, v := newRefresh(t, map[string]string{
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
	vaultport.VaultReaders
	refuses string
}

func (u unreadableReaders) Open(vault domain.Vault) (vaultport.VaultReader, error) {
	reader, err := u.VaultReaders.Open(vault)
	if err != nil {
		return nil, err
	}
	return unreadableReader{VaultReader: reader, refuses: u.refuses}, nil
}

type unreadableReader struct {
	vaultport.VaultReader
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
	_, db, v := newRefresh(t, map[string]string{
		"Locked.md": "---\ntitle: Locked\n---\n\n# Locked\n\nentropy\n",
		"Note.md":   "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})

	for _, name := range []string{"Locked.md", "Note.md"} {
		body := "---\ntitle: " + name + " edited\n---\n\n# Edited\n\nentropy\n"
		if err := os.WriteFile(filepath.Join(v.Path, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	refresh := vaults.NewRefresh(
		unreadableReaders{VaultReaders: vaultreader.Readers{}, refuses: "Locked.md"},
		db.Vaults(),
		db.Notes(),
		db.SourcesKnown(),
		db.Sources(), time.Now,
	)
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
	db := openIndex(t)
	content := &countingNotes{}
	refresh := vaults.NewRefresh(
		vaultreader.Readers{}, &indexRows{}, content, db.SourcesKnown(), db.Sources(), time.Now,
	)

	if _, err := refresh.Execute(t.Context(), v, paths); err != nil {
		t.Fatal(err)
	}
	if content.groups < 3 {
		t.Errorf("wrote %d notes in %d groups, want groups of at most 500",
			content.notes, content.groups)
	}
	for _, size := range content.sizes {
		if size > 500 {
			t.Errorf("a group held %d notes", size)
		}
	}
	if content.notes != 1200 {
		t.Errorf("wrote %d notes of 1200", content.notes)
	}
}

// countingNotes records how a repository was written to, and stores nothing.
type countingNotes struct {
	groups int
	notes  int
	sizes  []int
}

func (c *countingNotes) Save(_ context.Context, _ domain.VaultID, notes []domain.NoteChunks) error {
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
	_, db, v := newRefresh(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})

	refresh := vaults.NewRefresh(
		vanishingReaders{VaultReaders: vaultreader.Readers{}, vanishedVal: "Note.md"},
		db.Vaults(),
		db.Notes(),
		db.SourcesKnown(),
		db.Sources(), time.Now,
	)
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

// savesCounted is the index's note repository, savesCounted the notes saved through it.
type savesCounted struct {
	noteport.NoteRepository
	saved atomic.Int32
}

func (c *savesCounted) Save(ctx context.Context, vaultID domain.VaultID, notes []domain.NoteChunks) error {
	c.saved.Add(int32(len(notes)))
	return c.NoteRepository.Save(ctx, vaultID, notes)
}

const settledNote = "---\ntitle: Note\n---\n\n# Note\n\nentropy\n"

// newCounted is a vault scanned once with every file long settled, and a refresh
// whose saves are counted.
func newCounted(t *testing.T, notes map[string]string) (vaults.Refresh, *savesCounted, *container.Index, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	for name := range notes {
		backdate(t, v, name)
	}
	db := openIndex(t)
	if _, err := scanner(vaultreader.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	saves := &savesCounted{NoteRepository: db.Notes()}
	refresh := vaults.NewRefresh(
		vaultreader.Readers{}, db.Vaults(), saves, db.SourcesKnown(), db.Sources(), time.Now,
	)
	return refresh, saves, db, v
}

func backdate(t *testing.T, v domain.Vault, name string) {
	t.Helper()
	pos := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(v.Path, name), pos, pos); err != nil {
		t.Fatal(err)
	}
}

func rewrite(t *testing.T, v domain.Vault, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(v.Path, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func refreshOne(t *testing.T, refresh vaults.Refresh, v domain.Vault, name string) vaults.RefreshResult {
	t.Helper()
	res, err := refresh.Execute(t.Context(), v, []string{name})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestAnUnchangedNoteIsReportedAsIndexedAndNotSavedAgain(t *testing.T) {
	t.Parallel()
	refresh, saves, _, v := newCounted(t, map[string]string{"Note.md": settledNote})

	res := refreshOne(t, refresh, v, "Note.md")

	if !slices.Equal(res.Indexed, []string{"Note.md"}) {
		t.Errorf("indexed %v", res.Indexed)
	}
	if n := saves.saved.Load(); n != 0 {
		t.Errorf("%d notes were saved", n)
	}
}

func TestANoteEditedOutsideTheAppWithAnotherSizeIsIndexed(t *testing.T) {
	t.Parallel()
	refresh, saves, db, v := newCounted(t, map[string]string{"Note.md": settledNote})

	rewrite(t, v, "Note.md", settledNote+"more entropy\n")
	backdate(t, v, "Note.md")
	refreshOne(t, refresh, v, "Note.md")

	if n := saves.saved.Load(); n != 1 {
		t.Errorf("%d notes were saved", n)
	}
	if got := searchNotes(t, db, v, "more"); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("the index finds %v", got)
	}
}

func TestANoteEditedOutsideTheAppWithAnotherModificationTimeIsIndexed(t *testing.T) {
	t.Parallel()
	refresh, saves, db, v := newCounted(t, map[string]string{"Note.md": settledNote})

	rewrite(t, v, "Note.md", "---\ntitle: Note\n---\n\n# Note\n\nenthalpy\n")
	pos := time.Now().Add(-30 * time.Minute)
	if err := os.Chtimes(filepath.Join(v.Path, "Note.md"), pos, pos); err != nil {
		t.Fatal(err)
	}
	refreshOne(t, refresh, v, "Note.md")

	if n := saves.saved.Load(); n != 1 {
		t.Errorf("%d notes were saved", n)
	}
	if got := searchNotes(t, db, v, "enthalpy"); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("the index finds %v", got)
	}
}

func TestForceSavesANoteTheIndexContainsAsItIs(t *testing.T) {
	t.Parallel()
	refresh, saves, _, v := newCounted(t, map[string]string{"Note.md": settledNote})
	refresh.Force = true

	res := refreshOne(t, refresh, v, "Note.md")

	if !slices.Equal(res.Indexed, []string{"Note.md"}) {
		t.Errorf("indexed %v", res.Indexed)
	}
	if n := saves.saved.Load(); n != 1 {
		t.Errorf("%d notes were saved", n)
	}
}

func TestARecentNoteTheLedgerDoesNotHoldIsSaved(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, map[string]string{"Note.md": settledNote})
	db := openIndex(t)
	if _, err := scanner(vaultreader.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	saves := &savesCounted{NoteRepository: db.Notes()}
	refresh := vaults.NewRefresh(vaultreader.Readers{}, db.Vaults(), saves, db.SourcesKnown(), db.Sources(), time.Now)

	refreshOne(t, refresh, v, "Note.md")

	if n := saves.saved.Load(); n != 1 {
		t.Errorf("%d notes were saved", n)
	}
}

func TestAWriteFollowedByTheWatchersRefreshIsIndexedOnce(t *testing.T) {
	t.Parallel()
	refresh, saves, db, v := newCounted(t, map[string]string{"Note.md": settledNote})

	rewrite(t, v, "Note.md", "---\ntitle: Note\n---\n\n# Note\n\nenthalpy\n")
	refreshOne(t, refresh, v, "Note.md")
	res := refreshOne(t, refresh, v, "Note.md")

	if !slices.Equal(res.Indexed, []string{"Note.md"}) {
		t.Errorf("the second refresh reported %v", res.Indexed)
	}
	if n := saves.saved.Load(); n != 1 {
		t.Errorf("%d notes were saved", n)
	}
	if got := searchNotes(t, db, v, "enthalpy"); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("the index finds %v", got)
	}
}

func TestASameSizeEditWithinTheModificationTimeGranularityIsIndexed(t *testing.T) {
	t.Parallel()
	refresh, saves, db, v := newCounted(t, map[string]string{"Note.md": settledNote})

	path := filepath.Join(v.Path, "Note.md")
	rewrite(t, v, "Note.md", "---\ntitle: Note\n---\n\n# Note\n\nenthalpy\n")
	refreshOne(t, refresh, v, "Note.md")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	rewrite(t, v, "Note.md", "---\ntitle: Note\n---\n\n# Note\n\nenthalpx\n")
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	refreshOne(t, refresh, v, "Note.md")

	if n := saves.saved.Load(); n != 2 {
		t.Errorf("%d notes were saved", n)
	}
	if got := searchNotes(t, db, v, "enthalpx"); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("the index finds %v", got)
	}
}

func TestADeletedNoteIsRemovedWithoutBeingSaved(t *testing.T) {
	t.Parallel()
	refresh, saves, db, v := newCounted(t, map[string]string{"Note.md": settledNote})

	if err := os.Remove(filepath.Join(v.Path, "Note.md")); err != nil {
		t.Fatal(err)
	}
	res := refreshOne(t, refresh, v, "Note.md")

	if !slices.Equal(res.Removed, []string{"Note.md"}) || len(res.Indexed) != 0 {
		t.Errorf("removed %v, indexed %v", res.Removed, res.Indexed)
	}
	if n := saves.saved.Load(); n != 0 {
		t.Errorf("%d notes were saved", n)
	}
	if got := searchNotes(t, db, v, "entropy"); len(got) != 0 {
		t.Errorf("the index finds %v", got)
	}
}

func TestALongRunOfSavesPreservesTheLedgerContainingOnlyTheRecentWindow(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, map[string]string{"Note.md": settledNote})
	db := openIndex(t)
	if _, err := scanner(vaultreader.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	var tick atomic.Int64
	start := time.Now()
	clock := func() time.Time { return start.Add(time.Duration(tick.Load())) }
	refresh := vaults.NewRefresh(
		vaultreader.Readers{}, db.Vaults(), db.Notes(), db.SourcesKnown(), db.Sources(), clock)

	const perWindow = 4
	step := vaults.TimestampGranularity / perWindow
	for i := range 10 * perWindow {
		tick.Store(int64(step) * int64(i))
		rewrite(t, v, "Note.md", fmt.Sprintf("---\ntitle: Note\n---\n\n# Note\n\nversion %03d\n", i))
		pos := clock()
		if err := os.Chtimes(filepath.Join(v.Path, "Note.md"), pos, pos); err != nil {
			t.Fatal(err)
		}
		refreshOne(t, refresh, v, "Note.md")
		if n := refresh.Ledger.Len(); n > 1 {
			t.Fatalf("the ledger holds %d entries after save %d", n, i)
		}
	}
	tick.Add(int64(2 * vaults.TimestampGranularity))
	other := vaults.NewRefresh(
		vaultreader.Readers{}, db.Vaults(), db.Notes(), db.SourcesKnown(), db.Sources(), clock)
	other.Ledger = refresh.Ledger
	refreshOne(t, other, v, "Note.md")
	if n := refresh.Ledger.Len(); n != 0 {
		t.Errorf("the ledger holds %d entries once the window has passed", n)
	}
}
