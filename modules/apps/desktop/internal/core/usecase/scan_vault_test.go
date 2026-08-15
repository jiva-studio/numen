package usecase_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// The fixture vault is deliberately awkward: a note with no frontmatter, broken
// YAML, CRLF line endings, text that is not Latin, a PDF, and a folder that
// belongs to another tool.
func vaultAt(t *testing.T, root string) (domain.Vault, port.VaultReaders) {
	t.Helper()
	cfg, err := filesystem.ReadConfig(root, filesystem.DefaultServiceDir)
	if err != nil {
		t.Fatalf("fixture vault has no identity: %v", err)
	}
	return domain.Vault{ID: cfg.ID, Name: "fixture", Path: root},
		filesystem.Readers{ServiceDir: filesystem.DefaultServiceDir}
}

func openIndex(t *testing.T) *index.NoteRepository {
	t.Helper()
	repo, err := index.Open(t.Context(), filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestScanIndexesEveryNoteOnce(t *testing.T) {
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	notes := openIndex(t)

	res, err := usecase.ScanVault{Vaults: readers, Notes: notes}.Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 7 {
		t.Errorf("saw %d markdown files, want 7 — check what the walk skipped", res.Seen)
	}
	if res.Indexed != 7 || res.Unchanged != 0 || res.Removed != 0 {
		t.Errorf("first scan: %+v", res)
	}

	stats, err := notes.Stats(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Notes != 7 {
		t.Errorf("index holds %d notes, want 7", stats.Notes)
	}
}

func TestScanSkipsWhatIsNotVaultContent(t *testing.T) {
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	notes := openIndex(t)
	if _, err := (usecase.ScanVault{Vaults: readers, Notes: notes}).Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	// The service folder is ours and .obsidian belongs to another tool; a note
	// in either is not something the user wrote for us.
	hits, err := usecase.SearchNotes{Notes: notes}.Execute(ctx, v, "folder")
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if h.Path == ".obsidian/note-in-tool-folder.md" {
			t.Errorf("indexed a file from another tool's folder: %+v", h)
		}
	}
}

func TestSecondScanOpensNoFiles(t *testing.T) {
	ctx := t.Context()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	notes := openIndex(t)

	if _, err := (usecase.ScanVault{Vaults: readers, Notes: notes}).Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	counter := &countingReaders{VaultReaders: readers}
	second, err := usecase.ScanVault{Vaults: counter, Notes: notes}.Execute(ctx, v)
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
	notes := openIndex(t)
	scan := usecase.ScanVault{Vaults: readers, Notes: notes}

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

	search := usecase.SearchNotes{Notes: notes}
	hits, err := search.Execute(ctx, v, "crystallography")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Errorf("new text not searchable: %+v", hits)
	}
	// The vault is authoritative: what is not on disk is not in the index.
	stale, err := search.Execute(ctx, v, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Errorf("deleted note still searchable: %+v", stale)
	}
}

func TestSearchIsScopedToOneVault(t *testing.T) {
	ctx := t.Context()
	first, readers := vaultAt(t, testsupport.VaultDir(t))
	notes := openIndex(t)

	// One database holds every vault, so the leak this guards against is
	// invisible to any test that uses a single one.
	second := domain.Vault{ID: "01M02DTC80PABQQW3XS3XWDVHW", Name: "second", Path: first.Path}
	scan := usecase.ScanVault{Vaults: readers, Notes: notes}
	if _, err := scan.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := scan.Execute(ctx, second); err != nil {
		t.Fatal(err)
	}

	hits, err := usecase.SearchNotes{Notes: notes}.Execute(ctx, first, "entropy")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("no hits at all — the query is wrong, not the scoping")
	}
	stats, err := notes.Stats(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) > stats.Notes {
		t.Errorf("%d hits from a vault holding %d notes: results crossed vaults", len(hits), stats.Notes)
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
