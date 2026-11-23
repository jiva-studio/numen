package localchanges_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/localchanges"
)

func newRecord(number byte, path, content string) note.FileRecord {
	return note.FileRecord{
		FileID: bytes.Repeat([]byte{number}, 16), Path: path, BaseHash: note.Hash([]byte(content)), HeadRev: []byte{number}, Size: int64(len(content)),
		ModifiedAt: 0, Kind: note.Kind(path), BlobID: nil, BlobHash: nil, BlobSize: 0, SealedKey: nil,
	}
}

func getBases(records []note.FileRecord) map[string]string {
	bases := map[string]string{}
	for _, record := range records {
		bases[record.Path] = record.BaseHash
	}
	return bases
}

// detect looks at the vault as a pass does: the scan of changes first, then the search for moved and removed files.
func detect(t *testing.T, c *synctest.Computer, records []note.FileRecord) transfer.Scan {
	t.Helper()
	in := transfer.ScanInput{VaultID: vault, Bases: getBases(records), Records: records}
	if _, err := localchanges.NewScanLocalChanges(c.LocalChangesDeps()).Execute(t.Context(), in); err != nil {
		t.Fatal(err)
	}
	found, err := localchanges.NewDetectMoves(c.DetectDeps()).Execute(t.Context(), in)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func getRemovedPaths(scan transfer.Scan) []string {
	paths := make([]string, len(scan.Removals))
	for i, record := range scan.Removals {
		paths[i] = record.Path
	}
	return paths
}

func TestDetectMoves_FindsAFileThatIsGoneAndAFileWithItsHashAndSizeAsAMove(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/Report.pdf", "pdf bytes")}
	c.WriteNote(vault, "Papers/Old report.pdf", "pdf bytes")

	found := detect(t, c, records)

	if len(found.Moves) != 1 || found.Moves[0].To != "Papers/Old report.pdf" || found.Moves[0].Record.Path != "Papers/Report.pdf" {
		t.Errorf("moves = %+v, want the report moved to its new path", found.Moves)
	}
	if len(found.Removals) != 0 {
		t.Errorf("removals = %v, want none for a file that moved", getRemovedPaths(found))
	}
}

func TestDetectMoves_FindsNoMoveWhenTwoNewFilesHaveTheHashOfTheGoneFile(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/Report.pdf", "pdf bytes")}
	c.WriteNote(vault, "Papers/Copy 1.pdf", "pdf bytes")
	c.WriteNote(vault, "Papers/Copy 2.pdf", "pdf bytes")

	found := detect(t, c, records)

	if len(found.Moves) != 0 {
		t.Errorf("moves = %+v, want none: the match is not unique", found.Moves)
	}
}

func TestDetectMoves_FindsNoMoveWhenTwoGoneFilesHaveTheHashOfTheNewFile(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/A.pdf", "pdf bytes"), newRecord(2, "Papers/B.pdf", "pdf bytes")}
	c.WriteNote(vault, "Papers/C.pdf", "pdf bytes")

	found := detect(t, c, records)

	if len(found.Moves) != 0 {
		t.Errorf("moves = %+v, want none: the match is not unique", found.Moves)
	}
}

func TestDetectMoves_FindsNoMoveForAFileWithTheSizeButNotTheHash(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/Report.pdf", "aaaa")}
	c.WriteNote(vault, "Papers/Other.pdf", "bbbb")

	found := detect(t, c, records)

	if len(found.Moves) != 0 {
		t.Errorf("moves = %+v, want none for another file of the same size", found.Moves)
	}
}

func TestDetectMoves_FindsNoMoveToAFileThatIsNotThePersons(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/Report.pdf", "pdf bytes")}
	c.WriteNote(vault, "Papers/Report.pdf.part", "pdf bytes")

	found := detect(t, c, records)

	if len(found.Moves) != 0 {
		t.Errorf("moves = %+v, want none to a temporary file", found.Moves)
	}
}

func TestDetectMoves_WaitsForAFileYoungerThanTenSecondsBeforeItCallsItAMove(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/Report.pdf", "pdf bytes")}
	c.WriteNote(vault, "Papers/Old report.pdf", "pdf bytes")
	stat, err := c.Vaults.With(vault).StatFile(t.Context(), "Papers/Old report.pdf")
	if err != nil {
		t.Fatal(err)
	}
	c.Clock.Set(stat.ModTime.Add(3 * time.Second))

	young := detect(t, c, records)
	c.Clock.Advance(10 * time.Second)
	later := detect(t, c, records)

	if len(young.Moves) != 0 {
		t.Errorf("moves of a file 3 seconds old = %+v, want none yet", young.Moves)
	}
	if len(later.Moves) != 1 {
		t.Errorf("moves 13 seconds after the write = %+v, want the move", later.Moves)
	}
}

func TestDetectMoves_ListsAFileAsRemovedOnlyOnceItStayedGoneForTheDelay(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/Report.pdf", "pdf bytes")}

	first := detect(t, c, records)
	c.Clock.Advance(synctest.RemovalDelay / 2)
	half := detect(t, c, records)
	c.Clock.Advance(synctest.RemovalDelay / 2)
	whole := detect(t, c, records)

	if len(first.Removals) != 0 || len(half.Removals) != 0 {
		t.Errorf("removals at the first look and after half the delay = %v and %v, want none", getRemovedPaths(first), getRemovedPaths(half))
	}
	if paths := getRemovedPaths(whole); len(paths) != 1 || paths[0] != "Papers/Report.pdf" {
		t.Errorf("removals after the delay = %v, want the report", paths)
	}
}

func TestDetectMoves_CountsTheDelayAgainForAFileThatCameBackAndWentAgain(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/Report.pdf", "pdf bytes")}
	detect(t, c, records)
	c.Clock.Advance(600 * time.Millisecond)
	c.WriteNote(vault, "Papers/Report.pdf", "pdf bytes edited")
	back := detect(t, c, records)
	c.Vaults.With(vault).Remove("Papers/Report.pdf")
	c.Clock.Advance(600 * time.Millisecond)
	goneAgain := detect(t, c, records)
	c.Clock.Advance(synctest.RemovalDelay)
	waited := detect(t, c, records)

	if len(back.Removals) != 0 {
		t.Errorf("removals of a file that came back = %v, want none", getRemovedPaths(back))
	}
	if len(goneAgain.Removals) != 0 {
		t.Errorf("removals 0.6 seconds after it went again = %v, want none: the wait began again", getRemovedPaths(goneAgain))
	}
	if len(waited.Removals) != 1 {
		t.Errorf("removals one delay after it went again = %v, want the file", getRemovedPaths(waited))
	}
}

func TestDetectMoves_DoesNotListAFileThatIsInTheVaultAsRemoved(t *testing.T) {
	c := newComputer(t)
	records := []note.FileRecord{newRecord(1, "Papers/Report.pdf", "pdf bytes")}
	c.WriteNote(vault, "Papers/Report.pdf", "pdf bytes")
	in := transfer.ScanInput{VaultID: vault, Bases: getBases(records), Records: records}
	detectOnly := localchanges.NewDetectMoves(c.DetectDeps())
	if _, err := detectOnly.Execute(t.Context(), in); err != nil {
		t.Fatal(err)
	}
	c.Clock.Advance(synctest.RemovalDelay)

	found, err := detectOnly.Execute(t.Context(), in)

	if err != nil {
		t.Fatal(err)
	}
	if len(found.Removals) != 0 {
		t.Errorf("removals = %v, want none for a file the vault has", getRemovedPaths(found))
	}
}
