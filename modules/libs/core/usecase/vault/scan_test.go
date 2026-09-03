package vault_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// The fixture vault is deliberately awkward: a note with no frontmatter, broken
// YAML, CRLF line endings, text that is not Latin, a PDF, and a hidden folder.
func vaultAt(t *testing.T, root string) (domain.Vault, port.VaultReaders) {
	t.Helper()
	cfg, err := filesystem.ReadConfig(root, filesystem.DefaultServiceDir)
	if err != nil {
		t.Fatalf("fixture vault has no identity: %v", err)
	}
	return domain.Vault{ID: domain.VaultID(cfg.ID), Name: "fixture", Path: root},
		filesystem.VaultReaders{}
}

func openIndex(t *testing.T) *container.Index {
	t.Helper()
	db, err := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func scanner(readers port.VaultReaders, db *container.Index) usecase.Scan {
	return usecase.Scan{
		Readers:     readers,
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: db.Maintenance(),
	}
}

func TestScanIndexesEveryNoteOnce(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	res, err := scanner(readers, db).Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 14 {
		t.Errorf("saw %d markdown files, want 14 — check what the walk skipped", res.Notes)
	}
	if res.Indexed != 14 || res.Unchanged != 0 || res.Removed != 0 {
		t.Errorf("first scan: %+v", res)
	}

	summary, err := db.Queries().Summary(ctx, string(v.ID))
	if err != nil {
		t.Fatal(err)
	}
	if summary.Notes != 14 {
		t.Errorf("index holds %d notes, want 14", summary.Notes)
	}
}

// TestAScanSeesBooksBesideNotes. A book found in a vault is reported without
// anybody asking, and it is not counted as a note: what `Seen` means did not
// change when a second kind of source became visible.
func TestAScanSeesBooksBesideNotes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	root := testsupport.CopyVault(t)
	testsupport.WriteBook(t, root, "library/A Book.epub")
	v, readers := vaultAt(t, root)
	db := openIndex(t)

	res, err := scanner(readers, db).Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 14 {
		t.Errorf("saw %d notes, want the 14 the fixture holds", res.Notes)
	}
	// The fixture carries a PDF of its own beside the EPUB written here.
	if res.Assets != 2 {
		t.Errorf("saw %d sources of another kind, want the book and the document", res.Assets)
	}
	if res.Indexed != 14 || res.Removed != 0 {
		t.Errorf("scan of a vault with a book in it: %+v", res)
	}

	summary, err := db.Queries().Summary(ctx, string(v.ID))
	if err != nil {
		t.Fatal(err)
	}
	if summary.Notes != 14 {
		t.Errorf("index holds %d notes, want 14 — the book was parsed as one", summary.Notes)
	}
}

// TestAFormatNothingExtractsIsNotSeenAtAll. Processing is triggered by type, so
// a file of a format no reader handles is neither a note nor a source of
// another kind, however much of a vault it is.
func TestAFormatNothingExtractsIsNotSeenAtAll(t *testing.T) {
	t.Parallel()
	root := testsupport.CopyVault(t)
	if err := os.WriteFile(filepath.Join(root, "assets", "scan.png"), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, readers := vaultAt(t, root)
	db := openIndex(t)

	res, err := scanner(readers, db).Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture carries a document of its own, and it is a source. The image
	// written beside it is not.
	if res.Assets != 1 {
		t.Errorf("counted %d sources of another kind, want the document alone", res.Assets)
	}
	if res.Notes != 14 {
		t.Errorf("saw %d notes, want 14", res.Notes)
	}
}

// TestTwoVaultsCountAndAnswerForTheirOwnSourcesOnly. Two vaults sharing no
// words, and both directions asked: one index holds every vault, so a query
// that forgets which one returns a plausible number.
func TestTwoVaultsCountAndAnswerForTheirOwnSourcesOnly(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	first := testsupport.NewVault(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n\nthermodynamics\n",
	})
	testsupport.WriteBook(t, first.Path, "library/Thermodynamics.epub")

	second := testsupport.NewVault(t, map[string]string{
		"Quasar.md": "---\ntitle: Quasar\n---\n\n# Quasar\n\nredshift\n",
		"Pulsar.md": "---\ntitle: Pulsar\n---\n\n# Pulsar\n\nredshift\n",
	})
	testsupport.WriteBook(t, second.Path, "shelf/Astronomy.epub")
	testsupport.WriteBook(t, second.Path, "shelf/Cosmology.epub")

	db := openIndex(t)
	scan := scanner(filesystem.VaultReaders{}, db)

	one, err := scan.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if one.Notes != 1 || one.Assets != 1 {
		t.Errorf("the first vault scanned as %+v, want one note and one book", one)
	}
	two, err := scan.Execute(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if two.Notes != 2 || two.Assets != 2 {
		t.Errorf("the second vault scanned as %+v, want two notes and two books", two)
	}

	if got := titles(t, db, first, "thermodynamics"); !slices.Equal(got, []string{"Entropy"}) {
		t.Errorf("the first vault holds %v", got)
	}
	if got := titles(t, db, second, "thermodynamics"); len(got) != 0 {
		t.Errorf("the second vault answered with the first's notes: %v", got)
	}
	if got := titles(t, db, second, "redshift"); !slices.Equal(got, []string{"Pulsar", "Quasar"}) {
		t.Errorf("the second vault holds %v", got)
	}
	if got := titles(t, db, first, "redshift"); len(got) != 0 {
		t.Errorf("the first vault answered with the second's notes: %v", got)
	}
}

func TestScanSkipsWhatIsNotVaultContent(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	if _, err := scanner(readers, db).Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	matches, err := db.Queries().Search(ctx, string(v.ID), "hidden", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range matches {
		if m.Path == ".obsidian/note-in-a-hidden-folder.md" {
			t.Errorf("indexed a file from a hidden folder: %+v", m)
		}
	}
}

func TestSecondScanOpensNoFiles(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	if _, err := scanner(readers, db).Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	counter := &countingReaders{VaultReaders: readers}
	second, err := scanner(counter, db).Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	// This is what makes a scan cheap enough to run at startup: an unchanged
	// vault is decided on size and modification time, without opening a file.
	if counter.reads != 0 {
		t.Errorf("second scan read %d files, want 0", counter.reads)
	}
	if second.Unchanged != second.Notes || second.Indexed != 0 {
		t.Errorf("second scan: %+v", second)
	}
}

func TestEditedNoteIsReindexedAndDeletedNoteDisappears(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	// A copy, because this test writes: the fixture is shared and must stay
	// exactly as committed.
	root := testsupport.CopyVault(t)
	v, readers := vaultAt(t, root)
	db := openIndex(t)
	scan := scanner(readers, db)

	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	edited := filepath.Join(root, "daily", "2026-08-15.md")
	if err := os.WriteFile(edited, []byte("# Journal\n\na completely different text about crystallography\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Modification time is the invalidation key, and a test can write twice
	// within one timestamp tick.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(edited, future, future); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "edge", "crlf.md")); err != nil {
		t.Fatal(err)
	}

	res, err := scan.Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Indexed != 1 {
		t.Errorf("reindexed %d notes, want 1", res.Indexed)
	}
	if res.Removed != 1 {
		t.Errorf("removed %d notes, want 1", res.Removed)
	}

	queries := db.Queries()
	matches, err := queries.Search(ctx, string(v.ID), "crystallography", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Errorf("new text not searchable: %+v", matches)
	}
	// The vault is authoritative: what is not on disk is not in the index.
	stale, err := queries.Search(ctx, string(v.ID), "windows", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Errorf("deleted note still searchable: %+v", stale)
	}
}

// cancellingReaders stops the scan the moment the walk is over.
//
// Not during it: the reader refuses a cancelled context itself, so a scan that
// never looked at one would still stop there. What is left unguarded by anyone
// else is the pass over what the walk found, which is the whole of a warm scan.
type cancellingReaders struct {
	port.VaultReaders
	cancel context.CancelFunc
}

func (c *cancellingReaders) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := c.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return &cancellingReader{VaultReader: reader, parent: c}, nil
}

type cancellingReader struct {
	port.VaultReader
	parent *cancellingReaders
}

func (c *cancellingReader) Walk(ctx context.Context, fn func(domain.Fingerprint) error) error {
	err := c.VaultReader.Walk(ctx, fn)
	c.parent.cancel()
	return err
}

// countingReaders records how many files a scan actually opened.
type countingReaders struct {
	port.VaultReaders
	reads int
}

func (c *countingReaders) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := c.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return &countingReader{VaultReader: reader, parent: c}, nil
}

type countingReader struct {
	port.VaultReader
	parent *countingReaders
}

func (c *countingReader) Read(ctx context.Context, path string) ([]byte, error) {
	c.parent.reads++
	return c.VaultReader.Read(ctx, path)
}

func TestSearchNeverCrossesVaults(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := openIndex(t)

	first, readers := vaultAt(t, testsupport.VaultDir(t))
	second := testsupport.NewVault(t, map[string]string{
		"quasar.md": "# Quasar\n\nA word that exists in no other vault.\n",
	})

	if _, err := scanner(readers, db).Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := scanner(readers, db).Execute(ctx, second); err != nil {
		t.Fatal(err)
	}

	queries := db.Queries()

	// The two vaults hold disjoint words, so a query that forgets its vault
	// shows up as a match that cannot belong to the vault being searched. One
	// database for every vault makes that failure invisible by construction,
	// and a test that shares content between the vaults cannot see it either.
	leaked, err := queries.Search(ctx, string(first.ID), "quasar", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaked) != 0 {
		t.Errorf("searching the first vault returned the second vault's notes: %+v", leaked)
	}

	other, err := queries.Search(ctx, string(second.ID), "entropy", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Errorf("searching the second vault returned the first vault's notes: %+v", other)
	}

	own, err := queries.Search(ctx, string(second.ID), "quasar", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(own) != 1 || own[0].Path != "quasar.md" {
		t.Errorf("the second vault cannot find its own note: %+v", own)
	}
}

func TestFingerprintsAndSummaryNeverCrossVaults(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := openIndex(t)

	// Two vaults holding a note at the same path is not a corner case: copying
	// a vault folder produces exactly that. If Fingerprints leaks, the second
	// vault's note looks unchanged and is never indexed — silently, with the
	// scan reporting success.
	shared := "notes/Entropy.md"
	first := testsupport.NewVault(t, map[string]string{
		shared: "# Entropy\n\nthe first vault\n",
	})
	second := testsupport.NewVault(t, map[string]string{
		shared:     "# Entropy\n\nthe second vault\n",
		"extra.md": "# Extra\n\nonly the second vault has this\n",
	})
	readers := filesystem.VaultReaders{}

	if _, err := scanner(readers, db).Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	result, err := scanner(readers, db).Execute(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if result.Indexed != 2 {
		t.Errorf("the second vault indexed %d notes, want 2 — a shared path was taken for already known", result.Indexed)
	}

	queries := db.Queries()

	known, err := queries.Fingerprints(ctx, string(first.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 1 {
		t.Errorf("the first vault knows %d files, want 1: %v", len(known), known)
	}
	if _, leaked := known["extra.md"]; leaked {
		t.Error("the first vault knows a file that belongs to the second")
	}

	summary, err := queries.Summary(ctx, string(first.ID))
	if err != nil {
		t.Fatal(err)
	}
	if summary.Notes != 1 {
		t.Errorf("the first vault summarises %d notes, want 1", summary.Notes)
	}
}

// vanishingReader reports a file that is gone by the time it is read, which is
// what happens when the user saves, moves or deletes a note during a scan.
type vanishingReader struct {
	port.VaultReader
	gone string
}

func (v vanishingReader) Read(ctx context.Context, path string) ([]byte, error) {
	if path == v.gone {
		return nil, fs.ErrNotExist
	}
	return v.VaultReader.Read(ctx, path)
}

type vanishingReaders struct {
	port.VaultReaders
	gone string
}

func (v vanishingReaders) Open(vault domain.Vault) (port.VaultReader, error) {
	reader, err := v.VaultReaders.Open(vault)
	if err != nil {
		return nil, err
	}
	return vanishingReader{VaultReader: reader, gone: v.gone}, nil
}

func TestAFileThatDisappearsDuringAScanDoesNotStopIt(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	scan := scanner(vanishingReaders{VaultReaders: readers, gone: "notes/Entropy.md"}, db)
	res, err := scan.Execute(ctx, v)
	if err != nil {
		t.Fatalf("one vanished file ended the scan: %v", err)
	}
	if res.Vanished != 1 {
		t.Errorf("vanished = %d, want 1", res.Vanished)
	}
	if res.Indexed != res.Notes-1 {
		t.Errorf("indexed %d of %d seen", res.Indexed, res.Notes)
	}

	if res.Removed != 0 {
		t.Errorf("removed %d notes: a file that could not be read is not a deletion", res.Removed)
	}
}

type refusingReader struct {
	port.VaultReader
	refused string
}

func (r refusingReader) Read(ctx context.Context, path string) ([]byte, error) {
	if path == r.refused {
		return nil, fs.ErrPermission
	}
	return r.VaultReader.Read(ctx, path)
}

type refusingReaders struct {
	port.VaultReaders
	refused string
}

func (r refusingReaders) Open(vault domain.Vault) (port.VaultReader, error) {
	reader, err := r.VaultReaders.Open(vault)
	if err != nil {
		return nil, err
	}
	return refusingReader{VaultReader: reader, refused: r.refused}, nil
}

// A permission bit, a broken ACL, a device that went away. A refresh counts
// such a file and carries on, and the two ways the index changes agree.
func TestAFileNobodyCanReadDoesNotStopAScan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	scan := scanner(refusingReaders{VaultReaders: readers, refused: "notes/Entropy.md"}, db)
	res, err := scan.Execute(ctx, v)
	if err != nil {
		t.Fatalf("one unreadable file ended the scan: %v", err)
	}
	if res.Unreadable != 1 {
		t.Errorf("unreadable = %d, want 1", res.Unreadable)
	}
	if res.Indexed != res.Notes-1 {
		t.Errorf("indexed %d of %d seen", res.Indexed, res.Notes)
	}
	if res.Removed != 0 {
		t.Errorf("removed %d notes: a file nobody can read is not a deletion", res.Removed)
	}
}

func TestAVanishedFileKeepsWhatTheIndexAlreadyHad(t *testing.T) {
	t.Parallel()
	// Saving through a temporary file and a rename makes a note briefly absent.
	// A scan that catches that moment must not take the note out of search
	// until the next one.
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	if _, err := scanner(readers, db).Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	gone := "notes/Entropy.md"
	res, err := scanner(vanishingReaders{VaultReaders: readers, gone: gone}, db).Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed != 0 {
		t.Errorf("removed %d notes, want 0", res.Removed)
	}

	known, err := db.Queries().Fingerprints(ctx, string(v.ID))
	if err != nil {
		t.Fatal(err)
	}
	if _, kept := known[gone]; !kept {
		t.Error("a note that was briefly absent was dropped from the index")
	}
	matches, err := db.Queries().Search(ctx, string(v.ID), "uncertainty", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Errorf("the note is no longer searchable: %+v", matches)
	}
}

func TestFrontmatterThatCannotBeStoredDoesNotFailTheScan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := openIndex(t)
	// `.nan` is valid YAML and not representable in JSON. The note is still a
	// note, so it must still be indexed.
	v := testsupport.NewVault(t, map[string]string{
		"odd.md": "---\nvalue: .nan\n---\n\n# Odd\n\nsearchable all the same\n",
	})

	if _, err := scanner(filesystem.VaultReaders{}, db).Execute(ctx, v); err != nil {
		t.Fatalf("a note with unstorable frontmatter ended the scan: %v", err)
	}
	matches, err := db.Queries().Search(ctx, string(v.ID), "searchable", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Errorf("the note was not indexed: %+v", matches)
	}
}

func TestNewestNotesAreIndexedFirst(t *testing.T) {
	t.Parallel()
	// A vault has a working set and an archive. While a scan runs, what the
	// person is looking for is what they touched recently, so that is what the
	// index gets first.
	ctx := t.Context()
	root := testsupport.CopyVault(t)
	v, readers := vaultAt(t, root)
	db := openIndex(t)

	newest := filepath.Join(root, "edge", "unicode.md")
	at := time.Now().Add(time.Hour)
	if err := os.Chtimes(newest, at, at); err != nil {
		t.Fatal(err)
	}

	// Asked of the order the notes reach the index in: what the index holds
	// part-way through is whole groups, which says nothing about order.
	written := &groupedWrites{}
	scan := scanner(readers, db)
	scan.Notes = written
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	if len(written.groups) == 0 || written.groups[0][0] != "edge/unicode.md" {
		t.Errorf("first note indexed was %v, want edge/unicode.md", written.groups)
	}
}

// TestScanStopsWhenCancelled. A cancelled scan keeps what it committed and
// loses the group it was filling, whose files are read again next time.
//
// Asserted on how far the walk got, not on the error: the next read refuses a
// cancelled context by itself, so the error arrives either way.
func TestScanStopsWhenCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	v := testsupport.GenerateVault(t, 600)
	db := openIndex(t)

	scan := scanner(filesystem.VaultReaders{}, db)
	scan.OnProgress = func(usecase.ScanResult) { cancel() }

	res, err := scan.Execute(ctx, v)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled scan returned %v", err)
	}
	if res.Notes != 500 {
		t.Errorf("a cancelled scan walked %d files, want the 500 it had reached", res.Notes)
	}

	known, err := db.Queries().Fingerprints(t.Context(), string(v.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 500 {
		t.Errorf("a cancelled scan left %d notes indexed, want the 500 it committed", len(known))
	}
}

// TestAWarmScanStopsWhenCancelled is the case the check in the loop exists for.
// A warm scan opens no file, so nothing refuses the cancelled context on the
// scan's behalf and it has to notice by itself.
//
// Cancelled part-way rather than before it starts: cancelled up front, the
// first query refuses and the loop is never reached, so the check it is written
// for is never the thing that stopped it.
func TestAWarmScanStopsWhenCancelled(t *testing.T) {
	t.Parallel()
	v := testsupport.GenerateVault(t, 600)
	db := openIndex(t)
	scan := scanner(filesystem.VaultReaders{}, db)

	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	warm := scanner(&cancellingReaders{VaultReaders: filesystem.VaultReaders{}, cancel: cancel}, db)

	res, err := warm.Execute(ctx, v)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled warm scan returned %v", err)
	}
	if res.Notes != 0 {
		t.Errorf("a warm scan cancelled before its first note looked at %d of 600", res.Notes)
	}
}
