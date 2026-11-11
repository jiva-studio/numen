package vault_test

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/vaultfixture"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
)

// A scan cuts the notes at the chunk size the embedding plugin reports. These tests are about when it
// reads a note whose file did not change: when the recorded cut is not the cut it makes now.

func newOrchardVault(t *testing.T) domain.Vault {
	t.Helper()
	return vaultfixture.NewVault(t, map[string]string{
		"Hedge.md":   "# Hedge\n\nThe hedge along the lane is thick and tangled.\n",
		"Gate.md":    "# Gate\n\nThe gate of the orchard sticks in wet weather.\n",
		"Compost.md": "# Compost\n\nThe compost heap sits behind the shed.\n",
	})
}

func scanAt(db *container.Index, tokens int) vaults.Scan {
	return scanAtBacklog(db, tokens, db.Sources())
}

func scanAtBacklog(db *container.Index, tokens int, backlog sourceport.SourceBacklog) vaults.Scan {
	scan := vaults.NewScan(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), backlog)
	scan.Walks = &db.Walks
	scan.Cut = chunking.NewCutNotes(tokens)
	scan.ShouldRecut = tokens > 0
	return scan
}

// forbiddenBacklog fails the test when it is asked which sources were cut by another recipe.
type forbiddenBacklog struct {
	sourceport.SourceBacklog
	t *testing.T
}

func (b forbiddenBacklog) GetSourcesWithOtherRecipe(context.Context, domain.VaultID, domain.SourceKind, []string, int) ([]string, error) {
	b.t.Error("a scan that does not know the chunk size asked for the paths to cut again")
	return nil, nil
}

func getNotesWithOtherChunking(t *testing.T, db *container.Index, v domain.Vault, tokens int) []string {
	t.Helper()
	paths, err := db.Sources().GetSourcesWithOtherRecipe(t.Context(), v.ID, domain.KindNote, []string{chunking.NewCutNotes(tokens).Recipe()}, 100)
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func runScan(t *testing.T, scan vaults.Scan, v domain.Vault) vaultreading.ScanResult {
	t.Helper()
	res, err := scan.Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestAScanChunksEveryNoteOfAnotherChunkSizeAndNoNoteOnTheNextScan(t *testing.T) {
	t.Parallel()
	v, db := newOrchardVault(t), openIndex(t)

	if res := runScan(t, scanAt(db, 256), v); res.Indexed != 3 {
		t.Fatalf("first scan indexed %d notes, want 3", res.Indexed)
	}
	res := runScan(t, scanAt(db, 512), v)
	if res.Indexed != 3 || res.Unchanged != 0 {
		t.Errorf("scan at another size: indexed %d, unchanged %d; want 3 and 0", res.Indexed, res.Unchanged)
	}
	if got := getNotesWithOtherChunking(t, db, v, 512); len(got) != 0 {
		t.Errorf("%v were not cut at 512 tokens after the scan", got)
	}
	res = runScan(t, scanAt(db, 512), v)
	if res.Indexed != 0 || res.Unchanged != 3 {
		t.Errorf("second scan at the same size: indexed %d, unchanged %d; want 0 and 3", res.Indexed, res.Unchanged)
	}
}

func TestAScanAtTheRecordedChunkSizeReadsNoNote(t *testing.T) {
	t.Parallel()
	v, db := newOrchardVault(t), openIndex(t)
	runScan(t, scanAt(db, 512), v)

	if res := runScan(t, scanAt(db, 512), v); res.Indexed != 0 || res.Unchanged != 3 {
		t.Errorf("indexed %d, unchanged %d; want 0 and 3", res.Indexed, res.Unchanged)
	}
}

func TestAScanWithNoCheckGivenReadsNoUnchangedNoteWhateverItsSize(t *testing.T) {
	t.Parallel()
	v, db := newOrchardVault(t), openIndex(t)
	runScan(t, scanAt(db, 256), v)

	scan := scanAtBacklog(db, 512, nil)
	if res := runScan(t, scan, v); res.Indexed != 0 || res.Unchanged != 3 {
		t.Errorf("indexed %d, unchanged %d; want 0 and 3", res.Indexed, res.Unchanged)
	}
}

func TestAScanWithoutAChunkSizeNeitherAsksForRecutPathsNorReadsAnyNote(t *testing.T) {
	t.Parallel()
	v, db := newOrchardVault(t), openIndex(t)
	runScan(t, scanAt(db, 512), v)

	scan := scanAtBacklog(db, 0, forbiddenBacklog{SourceBacklog: db.Sources(), t: t})
	if res := runScan(t, scan, v); res.Indexed != 0 || res.Unchanged != 3 {
		t.Errorf("indexed %d, unchanged %d; want 0 and 3", res.Indexed, res.Unchanged)
	}
}

func TestANoteChangedWithoutAKnownSizeUsesTheDefaultAndALaterSizeRechunksOnlyTheOthers(t *testing.T) {
	t.Parallel()
	v, db := newOrchardVault(t), openIndex(t)
	runScan(t, scanAt(db, 512), v)

	changed := filepath.Join(v.Path, "Gate.md")
	if err := os.WriteFile(changed, []byte("# Gate\n\nThe gate of the orchard was mended in spring.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Date(2040, time.January, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(changed, later, later); err != nil {
		t.Fatal(err)
	}

	if res := runScan(t, scanAt(db, 0), v); res.Indexed != 1 || res.Unchanged != 2 {
		t.Errorf("scan with an unknown size: indexed %d, unchanged %d; want 1 and 2", res.Indexed, res.Unchanged)
	}
	if got := getNotesWithOtherChunking(t, db, v, 0); !slices.Equal(got, []string{"Compost.md", "Hedge.md"}) {
		t.Errorf("notes not cut at the default = %v, want the two left alone", got)
	}

	if res := runScan(t, scanAt(db, 512), v); res.Indexed != 1 || res.Unchanged != 2 {
		t.Errorf("scan at 512: indexed %d, unchanged %d; want only the note cut at the default", res.Indexed, res.Unchanged)
	}
	if res := runScan(t, scanAt(db, 256), v); res.Indexed != 3 {
		t.Errorf("scan at 256: indexed %d, want all 3", res.Indexed)
	}
}

// recipeless saves notes without the recipe of their cut, as an index made before notes kept one.
type recipeless struct{ noteport.NoteRepository }

func (r recipeless) Save(ctx context.Context, v domain.VaultID, notes []domain.NoteChunks) error {
	for i := range notes {
		notes[i].Recipe = ""
	}
	return r.NoteRepository.Save(ctx, v, notes)
}

func TestNotesWithoutARecordedChunkingAreChunkedOnceWhenTheSizeIsKnown(t *testing.T) {
	t.Parallel()
	v, db := newOrchardVault(t), openIndex(t)
	old := vaults.NewScan(vaultreader.Readers{}, db.Vaults(), recipeless{db.Notes()}, db.NoteQueries(), db.Maintenance(), nil)
	old.Walks = &db.Walks
	old.Cut = chunking.NewCutNotes(512)
	runScan(t, old, v)
	if got := getNotesWithOtherChunking(t, db, v, 512); len(got) != 3 {
		t.Fatalf("%d notes have a recorded cut, want none", 3-len(got))
	}

	if res := runScan(t, scanAt(db, 512), v); res.Indexed != 3 {
		t.Errorf("first scan with a known size indexed %d notes, want 3", res.Indexed)
	}
	if res := runScan(t, scanAt(db, 512), v); res.Indexed != 0 {
		t.Errorf("next scan indexed %d notes, want none", res.Indexed)
	}
	if got := getNotesWithOtherChunking(t, db, v, 512); len(got) != 0 {
		t.Errorf("%v still have no recorded cut", got)
	}
}

func TestRechunkedNotesAreCountedAndSavedInGroupsLikeChangedFiles(t *testing.T) {
	t.Parallel()
	const notes = 600
	v, db := vaultfixture.GenerateVault(t, notes), openIndex(t)
	runScan(t, scanAt(db, 256), v)

	content := &groupedWrites{}
	scan := vaults.NewScan(vaultreader.Readers{}, db.Vaults(), content, db.NoteQueries(), db.Maintenance(), db.Sources())
	scan.Walks = &db.Walks
	scan.Cut = chunking.NewCutNotes(512)
	scan.ShouldRecut = true
	res := runScan(t, scan, v)

	if res.Indexed != notes || len(content.groups) != 2 {
		t.Errorf("indexed %d notes in %d groups, want %d in 2", res.Indexed, len(content.groups), notes)
	}
	for _, group := range content.groups {
		if len(group) > 500 {
			t.Errorf("a group of %d notes", len(group))
		}
	}
}
