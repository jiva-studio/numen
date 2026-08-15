package vault_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// The fixture vault is deliberately awkward: a note with no frontmatter, broken
// YAML, CRLF line endings, text that is not Latin, a PDF, and a hidden folder.
func vaultAt(t *testing.T, root string) (domain.Vault, port.VaultReaders) {
	t.Helper()
	cfg, err := filesystem.ReadConfig(root, filesystem.DefaultServiceDir)
	if err != nil {
		t.Fatalf("fixture vault has no identity: %v", err)
	}
	return domain.Vault{ID: cfg.ID, Name: "fixture", Path: root},
		filesystem.Readers{}
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
		Readers: readers,
		Vaults:  db.Vaults(),
		Notes:   db.Notes(),
		Known:   db.Queries(),
	}
}

func TestScanIndexesEveryNoteOnce(t *testing.T) {
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	res, err := scanner(readers, db).Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 7 {
		t.Errorf("saw %d markdown files, want 7 — check what the walk skipped", res.Seen)
	}
	if res.Indexed != 7 || res.Unchanged != 0 || res.Removed != 0 {
		t.Errorf("first scan: %+v", res)
	}

	summary, err := db.Queries().Summary(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Notes != 7 {
		t.Errorf("index holds %d notes, want 7", summary.Notes)
	}
}

func TestScanSkipsWhatIsNotVaultContent(t *testing.T) {
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	if _, err := scanner(readers, db).Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	matches, err := db.Queries().Search(ctx, v.ID, "hidden", 10)
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
	if second.Unchanged != second.Seen || second.Indexed != 0 {
		t.Errorf("second scan: %+v", second)
	}
}

func TestEditedNoteIsReindexedAndDeletedNoteDisappears(t *testing.T) {
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
	matches, err := queries.Search(ctx, v.ID, "crystallography", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Errorf("new text not searchable: %+v", matches)
	}
	// The vault is authoritative: what is not on disk is not in the index.
	stale, err := queries.Search(ctx, v.ID, "windows", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Errorf("deleted note still searchable: %+v", stale)
	}
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
	// shows up as a match that cannot belong to the vault being searched. This
	// is the one failure ADR-0002 calls invisible by construction, and a test
	// that shares content between the vaults cannot see it either.
	leaked, err := queries.Search(ctx, first.ID, "quasar", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaked) != 0 {
		t.Errorf("searching the first vault returned the second vault's notes: %+v", leaked)
	}

	other, err := queries.Search(ctx, second.ID, "entropy", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Errorf("searching the second vault returned the first vault's notes: %+v", other)
	}

	own, err := queries.Search(ctx, second.ID, "quasar", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(own) != 1 || own[0].Path != "quasar.md" {
		t.Errorf("the second vault cannot find its own note: %+v", own)
	}
}

func TestFingerprintsAndSummaryNeverCrossVaults(t *testing.T) {
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
	readers := filesystem.Readers{}

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

	known, err := queries.Fingerprints(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 1 {
		t.Errorf("the first vault knows %d files, want 1: %v", len(known), known)
	}
	if _, leaked := known["extra.md"]; leaked {
		t.Error("the first vault knows a file that belongs to the second")
	}

	summary, err := queries.Summary(ctx, first.ID)
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
	if res.Indexed != res.Seen-1 {
		t.Errorf("indexed %d of %d seen", res.Indexed, res.Seen)
	}

	if res.Removed != 0 {
		t.Errorf("removed %d notes: a file that could not be read is not a deletion", res.Removed)
	}
}

func TestAVanishedFileKeepsWhatTheIndexAlreadyHad(t *testing.T) {
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

	known, err := db.Queries().Fingerprints(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, kept := known[gone]; !kept {
		t.Error("a note that was briefly absent was dropped from the index")
	}
	matches, err := db.Queries().Search(ctx, v.ID, "uncertainty", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Errorf("the note is no longer searchable: %+v", matches)
	}
}

func TestFrontmatterThatCannotBeStoredDoesNotFailTheScan(t *testing.T) {
	ctx := t.Context()
	db := openIndex(t)
	// `.nan` is valid YAML and not representable in JSON. The note is still a
	// note, so it must still be indexed.
	v := testsupport.NewVault(t, map[string]string{
		"odd.md": "---\nvalue: .nan\n---\n\n# Odd\n\nsearchable all the same\n",
	})

	if _, err := scanner(filesystem.Readers{}, db).Execute(ctx, v); err != nil {
		t.Fatalf("a note with unstorable frontmatter ended the scan: %v", err)
	}
	matches, err := db.Queries().Search(ctx, v.ID, "searchable", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Errorf("the note was not indexed: %+v", matches)
	}
}
