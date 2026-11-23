package pull_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

const (
	pdf     = "Papers/Report.pdf"
	pdfText = "%PDF-1.7 the bytes of a report"
	bigSize = 65 << 20
)

func (p *pair) remove(t *testing.T, records ...transfer.FileRecord) []transfer.Outcome {
	t.Helper()
	outcomes, err := commit.NewCommitRemovals(p.a.CommitDeps()).Execute(t.Context(), transfer.RemoveInput{Vault: p.vault, Records: records})
	if err != nil {
		t.Fatal(err)
	}
	return outcomes
}

func (p *pair) move(t *testing.T, moves ...transfer.Move) []transfer.Outcome {
	t.Helper()
	outcomes, err := commit.NewCommitMoves(p.a.CommitDeps()).Execute(t.Context(), transfer.MoveInput{Vault: p.vault, Moves: moves})
	if err != nil {
		t.Fatal(err)
	}
	return outcomes
}

func (p *pair) pushMany(t *testing.T, folder string, count int) []transfer.Outcome {
	t.Helper()
	files := make([]transfer.File, count)
	for i := range files {
		files[i] = transfer.File{Path: fmt.Sprintf("%s/File %d.pdf", folder, i+1), Content: []byte(fmt.Sprintf("file %d", i+1))}
	}
	outcomes, err := commit.NewCommitRevisions(p.a.CommitDeps()).Execute(t.Context(), transfer.CommitInput{Vault: p.vault, Files: files})
	if err != nil {
		t.Fatal(err)
	}
	return outcomes
}

func (p *pair) getStagedNames(t *testing.T) []string {
	t.Helper()
	files, err := p.b.Host.Store(target).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, file := range files {
		if strings.HasSuffix(file.Name, ".part") {
			names = append(names, file.Name)
		}
	}
	return names
}

func (p *pair) readNote(t *testing.T, path string) string {
	t.Helper()
	text, _ := p.b.ReadNote(target, path)
	return text
}

func TestPullChanges_PlacesAFileOfAnyKindWholeThroughTheHost(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	p.push(t, pdf, pdfText, nil)

	result := p.pull(t, trustAll)

	if got := p.readNote(t, pdf); got != pdfText {
		t.Errorf("file = %q, want the bytes of the report", got)
	}
	places := spy.GetPlaces()
	if len(places) != 1 || places[0].Path != pdf || places[0].Hash != note.Hash([]byte(pdfText)) || places[0].Baseline != nil {
		t.Errorf("placements = %+v, want one of the whole file onto a free path", places)
	}
	if names := p.getStagedNames(t); len(names) != 0 {
		t.Errorf("staged files left = %v, want none after the placement", names)
	}
	if !slices.Equal(result.Applied, []string{pdf}) {
		t.Errorf("applied = %v, want the file", result.Applied)
	}
	if row, ok := p.b.GetFileRow(t, target, pdf); !ok || row.BaseHash != note.Hash([]byte(pdfText)) {
		t.Errorf("row = %+v (there: %v), want the hash of the file as base", row, ok)
	}
}

func TestPullChanges_ReplacesAFileThatEqualsItsBaseWithABaselineOfWhatItSaw(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.push(t, pdf, pdfText+" edited", &first)

	p.pull(t, trustAll)

	if got := p.readNote(t, pdf); got != pdfText+" edited" {
		t.Errorf("file = %q, want the new bytes", got)
	}
	places := spy.GetPlaces()
	if len(places) != 2 || places[1].Baseline == nil || places[1].Baseline.Hash != note.Hash([]byte(pdfText)) {
		t.Errorf("placements = %+v, want the second to replace the file it read", places)
	}
}

func TestPullChanges_KeepsAFileChangedOnBothComputersAndPlacesNothing(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, pdf, "edited here")
	p.push(t, pdf, pdfText+" edited there", &first)

	result := p.pull(t, trustAll)

	if got := p.readNote(t, pdf); got != "edited here" {
		t.Errorf("file = %q, want the local bytes", got)
	}
	if !slices.Equal(result.Conflicts, []string{pdf}) || len(spy.GetPlaces()) != 1 {
		t.Errorf("conflicts = %v with %d placements, want the file in conflict and only the first placement", result.Conflicts, len(spy.GetPlaces()))
	}
}

func TestPullChanges_TreatsAFileThatAppearedBeforeItsPlacementAsAConflict(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	spy.RunBeforePlace(func(call synctest.PlaceCall) { p.b.WriteNote(target, call.Path, "written meanwhile") })
	p.push(t, pdf, pdfText, nil)

	result := p.pull(t, trustAll)

	if got := p.readNote(t, pdf); got != "written meanwhile" {
		t.Errorf("file = %q, want the file that appeared, untouched", got)
	}
	if !slices.Equal(result.Conflicts, []string{pdf}) {
		t.Errorf("conflicts = %v, want the file", result.Conflicts)
	}
	if _, ok := p.b.GetFileRow(t, target, pdf); ok {
		t.Error("a record was kept for a file that was not placed")
	}
	if names := p.getStagedNames(t); len(names) != 0 {
		t.Errorf("staged files left = %v, want none", names)
	}
}

func TestPullChanges_RaisesTheIntegrityItemAndStopsThePositionBeforeTheFileWhenTheHostFindsAnotherHash(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	spy.RunBeforePlace(func(call synctest.PlaceCall) {
		p.b.Vaults.Store().With(target).WriteFile(call.Name, []byte("tampered"))
	})
	p.push(t, pdf, pdfText, nil)
	head := p.a.ReadChanges(t, p.vault.ID)[0]

	result := p.pull(t, trustAll)

	if !p.b.HasItem("A file from the sync server is damaged") {
		t.Errorf("health items = %+v, want A file from the sync server is damaged", p.b.Health.List())
	}
	if _, ok := p.b.ReadNote(target, pdf); ok {
		t.Error("a file that did not match its hash was placed")
	}
	if link := p.b.Link(t, target); link.ServerSeq != head.GetSeq()-1 {
		t.Errorf("position = %d, want %d: the pull stops before the damaged file", link.ServerSeq, head.GetSeq()-1)
	}
	if len(result.Applied) != 0 || len(p.getStagedNames(t)) != 0 {
		t.Errorf("applied %v with staged files %v, want nothing placed and nothing left staged", result.Applied, p.getStagedNames(t))
	}
}

func TestPullChanges_RaisesTheIntegrityItemForABlobThatIsNotTheOneOfItsRevision(t *testing.T) {
	p := newPair(t)
	p.push(t, pdf, pdfText, nil)
	p.server.SetBlobCorruption(true)

	result := p.pull(t, trustAll)

	if !p.b.HasItemKey(pull.KeyIntegrity) {
		t.Errorf("health items = %+v, want the integrity item", p.b.Health.List())
	}
	if _, ok := p.b.ReadNote(target, pdf); ok || len(result.Applied) != 0 {
		t.Error("a damaged file was placed")
	}
	if names := p.getStagedNames(t); len(names) != 0 {
		t.Errorf("staged files left = %v, want none", names)
	}
}

func TestPullChanges_DeletesAStagedFileNoDownloadNamesAtItsStart(t *testing.T) {
	p := newPair(t)
	p.b.Vaults.Store().With(target).WriteFile("0badc0de.part", []byte("left behind"))

	p.pull(t, trustAll)

	if names := p.getStagedNames(t); len(names) != 0 {
		t.Errorf("staged files = %v, want the leftover deleted", names)
	}
}

func TestPullChanges_RemovesAFileTheOtherComputerRemovedThroughTheTrashWithTheStatItRead(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.remove(t, first.Record)

	p.pull(t, trustAll)

	if _, ok := p.b.ReadNote(target, pdf); ok {
		t.Error("the file is still in the vault")
	}
	removes := spy.GetRemoves()
	if len(removes) != 1 || removes[0].Path != pdf || removes[0].Baseline == nil || removes[0].Baseline.Hash != note.Hash([]byte(pdfText)) {
		t.Errorf("removals = %+v, want one of the file with the stat it had", removes)
	}
	if trashed := p.b.Vaults.With(target).Trashed(); !slices.Equal(trashed, []string{pdf}) {
		t.Errorf("trashed = %v, want the file", trashed)
	}
	if _, ok := p.b.GetFileRow(t, target, pdf); ok {
		t.Error("the record of a removed file is still kept")
	}
}

func TestPullChanges_KeepsAFileEditedHereAndPointsItsRecordAtTheRemoval(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, pdf, "edited here")
	removal := p.remove(t, first.Record)[0]

	p.pull(t, trustAll)

	if got := p.readNote(t, pdf); got != "edited here" || len(spy.GetRemoves()) != 0 {
		t.Errorf("file = %q with %d removals, want the edited file kept", got, len(spy.GetRemoves()))
	}
	row, ok := p.b.GetFileRow(t, target, pdf)
	wantHead, err := json.Marshal(removal.RevID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || string(row.HeadRev) != string(wantHead) || row.BaseHash != note.Hash([]byte(pdfText)) {
		t.Errorf("row = %+v (there: %v), want the removal as head and the old base: the next push is an edit over the removal", row, ok)
	}
}

func TestPullChanges_DropsTheRecordOfARemovedFileThatIsAlreadyMissing(t *testing.T) {
	p := newPair(t)
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.Vaults.With(target).Remove(pdf)
	p.remove(t, first.Record)

	p.pull(t, trustAll)

	if _, ok := p.b.GetFileRow(t, target, pdf); ok {
		t.Error("the record of a file that is gone is still kept")
	}
}

func TestPullChanges_IgnoresTheRemovalOfAFileItNeverHad(t *testing.T) {
	p := newPair(t)
	first := p.push(t, pdf, pdfText, nil)
	p.remove(t, first.Record)

	result := p.pull(t, trustAll)

	if len(result.Applied) != 0 || len(result.Conflicts) != 0 || p.b.Link(t, target).ServerSeq == 0 {
		t.Errorf("result = %+v, want nothing applied and the position moved on", result)
	}
}

func TestPullChanges_DefersManyRemovalsToTheGuardAndStopsThePosition(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	pushed := p.pushMany(t, "Papers", 10)
	p.pull(t, trustAll)
	records := make([]transfer.FileRecord, len(pushed))
	for i, one := range pushed {
		records[i] = one.Record
	}
	p.remove(t, records...)
	before := p.b.Link(t, target).ServerSeq

	p.pull(t, trustAll)

	if len(spy.GetRemoves()) != 0 {
		t.Errorf("%d files were removed, want none before the answer", len(spy.GetRemoves()))
	}
	paused, err := p.b.State.ReadPending(t.Context(), target)
	if err != nil || paused == nil || paused.Direction != transfer.DirectionApply || len(paused.Files) != 10 || paused.Tracked != 10 {
		t.Fatalf("pending = %+v (%v), want a pause of 10 removals to apply of 10 tracked", paused, err)
	}
	if !p.b.HasItem("Sync is paused: 10 of 10 files were removed on another computer") {
		t.Errorf("health items = %+v, want the pause", p.b.Health.List())
	}
	if got := p.b.Link(t, target).ServerSeq; got != before || got >= p.a.ReadChanges(t, p.vault.ID)[0].GetSeq() {
		t.Errorf("position = %d, want %d: it stays before the removals", got, before)
	}
	if got := p.readNote(t, "Papers/File 1.pdf"); got != "file 1" {
		t.Errorf("file 1 = %q, want it kept while the answer is open", got)
	}
}

func TestPullChanges_RemovesTheFilesOnceThePersonConfirmsTheRemoval(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	pushed := p.pushMany(t, "Papers", 10)
	p.pull(t, trustAll)
	records := make([]transfer.FileRecord, len(pushed))
	for i, one := range pushed {
		records[i] = one.Record
	}
	p.remove(t, records...)
	p.pull(t, trustAll)
	if _, err := deletions.NewResolveDeletions(p.b.DeleteDeps()).Execute(t.Context(), target, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	p.pull(t, trustAll)

	if len(spy.GetRemoves()) != 10 || len(p.b.Vaults.With(target).Trashed()) != 10 {
		t.Errorf("%d removals and %d files in the trash, want 10 of each", len(spy.GetRemoves()), len(p.b.Vaults.With(target).Trashed()))
	}
	if rows := p.b.ReadFileRows(t, target); len(rows) != 0 {
		t.Errorf("records = %+v, want none", rows)
	}
	if p.b.HasItemKey(deletions.HealthKey) {
		t.Error("the pause is still shown after the answer")
	}
	if got := p.b.Link(t, target).ServerSeq; got != p.a.ReadChanges(t, p.vault.ID)[0].GetSeq() {
		t.Errorf("position = %d, want the removals read", got)
	}
}

func TestPullChanges_KeepsTheFilesWhenThePersonKeepsThemAndSendsThemAsEdits(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	pushed := p.pushMany(t, "Papers", 10)
	p.pull(t, trustAll)
	records := make([]transfer.FileRecord, len(pushed))
	for i, one := range pushed {
		records[i] = one.Record
	}
	removals := p.remove(t, records...)
	p.pull(t, trustAll)
	if _, err := deletions.NewResolveDeletions(p.b.DeleteDeps()).Execute(t.Context(), target, transfer.ChoiceKeep); err != nil {
		t.Fatal(err)
	}

	p.pull(t, trustAll)

	if len(spy.GetRemoves()) != 0 || p.readNote(t, "Papers/File 1.pdf") != "file 1" {
		t.Errorf("%d removals, file 1 = %q; want every file kept", len(spy.GetRemoves()), p.readNote(t, "Papers/File 1.pdf"))
	}
	row, ok := p.b.GetFileRow(t, target, "Papers/File 1.pdf")
	wantHead, err := json.Marshal(removals[0].RevID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || row.BaseHash != "" || string(row.HeadRev) != string(wantHead) {
		t.Errorf("row = %+v (there: %v), want an empty base and the removal as head, so the file goes up again as an edit", row, ok)
	}
}

func TestPullChanges_MovesAFileToItsNewPathWithTheStatItReadAndDownloadsNothing(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.move(t, transfer.Move{Record: first.Record, To: "Papers/Old report.pdf"})
	urls := p.server.Fake.CountCalls(syncv1connect.BlobServiceGetBlobUrlsProcedure)

	result := p.pull(t, trustAll)

	if got := p.readNote(t, "Papers/Old report.pdf"); got != pdfText {
		t.Errorf("file at the new path = %q, want the report", got)
	}
	if _, ok := p.b.ReadNote(target, pdf); ok {
		t.Error("the file is still at its old path")
	}
	moves := spy.GetMoves()
	if len(moves) != 1 || moves[0].From != pdf || moves[0].To != "Papers/Old report.pdf" || moves[0].Baseline == nil || moves[0].Baseline.Hash != note.Hash([]byte(pdfText)) {
		t.Errorf("moves = %+v, want one with the stat of the file", moves)
	}
	if got := p.server.Fake.CountCalls(syncv1connect.BlobServiceGetBlobUrlsProcedure); got != urls {
		t.Errorf("blob URL requests grew from %d to %d, want no download for a move", urls, got)
	}
	row, ok := p.b.GetFileRow(t, target, "Papers/Old report.pdf")
	if !ok || row.BaseHash != note.Hash([]byte(pdfText)) || !slices.Equal(result.Applied, []string{"Papers/Old report.pdf"}) {
		t.Errorf("row = %+v (there: %v), applied %v; want the new path with the base unchanged", row, ok, result.Applied)
	}
	if _, ok := p.b.GetFileRow(t, target, pdf); ok {
		t.Error("the record of the old path is still kept")
	}
}

func TestPullChanges_MovesAFolderWholeWhenTheBatchMovesEveryFileInsideIt(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	pushed := p.pushMany(t, "Food", 3)
	p.pull(t, trustAll)
	moves := make([]transfer.Move, len(pushed))
	for i, one := range pushed {
		moves[i] = transfer.Move{Record: one.Record, To: fmt.Sprintf("Meals/File %d.pdf", i+1)}
	}
	p.move(t, moves...)

	p.pull(t, trustAll)

	calls := spy.GetMoves()
	if len(calls) != 1 || calls[0].From != "Food" || calls[0].To != "Meals" || calls[0].Baseline != nil {
		t.Errorf("moves = %+v, want one move of the folder Food to Meals", calls)
	}
	for i := range 3 {
		if got := p.readNote(t, fmt.Sprintf("Meals/File %d.pdf", i+1)); got != fmt.Sprintf("file %d", i+1) {
			t.Errorf("file %d at the new path = %q", i+1, got)
		}
		if _, ok := p.b.GetFileRow(t, target, fmt.Sprintf("Meals/File %d.pdf", i+1)); !ok {
			t.Errorf("file %d has no record at its new path", i+1)
		}
	}
}

func TestPullChanges_MovesTheFilesOneByOneWhenTheBatchOmitsAFileOfTheFolder(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	pushed := p.pushMany(t, "Food", 3)
	p.pull(t, trustAll)
	p.move(t,
		transfer.Move{Record: pushed[0].Record, To: "Meals/File 1.pdf"},
		transfer.Move{Record: pushed[1].Record, To: "Meals/File 2.pdf"},
	)

	p.pull(t, trustAll)

	calls := spy.GetMoves()
	if len(calls) != 2 || calls[0].Baseline == nil || calls[1].Baseline == nil || calls[0].From == "Food" || calls[1].From == "Food" {
		t.Errorf("moves = %+v, want two moves of one file each, each with its stat", calls)
	}
	if got := p.readNote(t, "Food/File 3.pdf"); got != "file 3" {
		t.Errorf("the file that stayed = %q, want it kept in its folder", got)
	}
}

func TestPullChanges_KeepsALocalFileAtTheNewPathAndMovesNothing(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, "Papers/Old report.pdf", "another file")
	p.move(t, transfer.Move{Record: first.Record, To: "Papers/Old report.pdf"})

	result := p.pull(t, trustAll)

	if !slices.Equal(result.Conflicts, []string{"Papers/Old report.pdf"}) || len(spy.GetMoves()) != 0 {
		t.Errorf("conflicts = %v with %d moves, want the new path in conflict and no move", result.Conflicts, len(spy.GetMoves()))
	}
	if got := p.readNote(t, pdf); got != pdfText {
		t.Errorf("file at the old path = %q, want it kept", got)
	}
}

func TestPullChanges_KeepsAFileEditedHereWhereItIsWhenTheOtherComputerMovedIt(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.b.WriteNote(target, pdf, "edited here")
	p.move(t, transfer.Move{Record: first.Record, To: "Papers/Old report.pdf"})

	result := p.pull(t, trustAll)

	if len(spy.GetMoves()) != 0 || len(result.Conflicts) != 1 {
		t.Errorf("%d moves and conflicts %v, want the edited file left where it is", len(spy.GetMoves()), result.Conflicts)
	}
	if got := p.readNote(t, pdf); got != "edited here" {
		t.Errorf("file = %q, want the local bytes", got)
	}
}

func TestPullChanges_CountsAnUnverifiedMoveOnce(t *testing.T) {
	p := newPair(t)
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.move(t, transfer.Move{Record: first.Record, To: "Papers/Old report.pdf"})

	result := p.pull(t, func([]byte) bool { return false })

	if result.Unverified != 1 {
		t.Errorf("unverified = %d, want 1 for one revision", result.Unverified)
	}
}

func TestPullChanges_StopsAtAPageThatAsksForAFullCheckAndStoresNothing(t *testing.T) {
	p := newPair(t)
	first := p.push(t, pdf, pdfText, nil)
	p.pull(t, trustAll)
	p.remove(t, first.Record)
	if pruned := p.server.Fake.PruneTombstones(); pruned != 1 {
		t.Fatalf("pruned = %d, want the one removal", pruned)
	}
	before := p.b.Link(t, target)

	result := p.pull(t, trustAll)

	if !result.IsReset || !p.b.HasItem("Sync needs to compare the vault again") {
		t.Errorf("result = %+v with items %+v, want the reset noted and the item raised", result, p.b.Health.List())
	}
	if after := p.b.Link(t, target); after.ServerSeq != before.ServerSeq {
		t.Errorf("position = %d, want %d unchanged", after.ServerSeq, before.ServerSeq)
	}
	if got := p.readNote(t, pdf); got != pdfText {
		t.Errorf("file = %q, want it untouched", got)
	}
}

// watchedBlobs reports a call of check at every read of a blob body.
type watchedBlobs struct {
	ports.BlobReader
	check func()
}

type watchedBody struct {
	io.ReadCloser
	check func()
}

func (b watchedBlobs) OpenBlob(ctx context.Context, vault, blob []byte, from int64) (io.ReadCloser, error) {
	body, err := b.BlobReader.OpenBlob(ctx, vault, blob, from)
	if err != nil {
		return nil, err
	}
	return watchedBody{ReadCloser: body, check: b.check}, nil
}

func (b watchedBody) Read(p []byte) (int, error) {
	b.check()
	return b.ReadCloser.Read(p)
}

func TestPullChanges_PlacesABigFileOnceAndShowsNoFileAtItsPathUntilItIsWhole(t *testing.T) {
	p := newPair(t)
	spy := p.b.Spy()
	trip := synctest.NewSource("Videos/Trip.bin", bigSize, 1, synctest.GetStart().Add(-time.Hour))
	p.a.Spy().Serve(trip)
	stat := trip.GetStat()
	outcomes, err := commit.NewCommitRevisions(p.a.CommitDeps()).Execute(t.Context(), transfer.CommitInput{
		VaultID: source, Vault: p.vault, Files: []transfer.File{{Path: trip.GetPath(), ModifiedAt: stat.ModTime.UnixNano(), Stat: &stat}},
	})
	if err != nil || outcomes[0].Status != transfer.Committed {
		t.Fatalf("outcomes = %+v (%v), want the big file committed", outcomes, err)
	}
	deps := p.b.PullDeps()
	wired := p.b.WireDeps()
	reads := 0
	deps.Downloads = download.NewDownloadFile(download.Deps{
		Blobs: watchedBlobs{BlobReader: wired.DownloadBlobs, check: func() {
			reads++
			if _, isThere := p.b.ReadNote(target, trip.GetPath()); isThere {
				t.Error("a file was in the vault while it was still being received")
			}
		}},
		Entries: p.b.State, Host: p.b.Host, Digests: wired.Digests,
	})

	result, err := pull.NewPullChanges(deps).Execute(t.Context(), transfer.PullInput{VaultID: target, IsAuthor: trustAll})

	if err != nil {
		t.Fatal(err)
	}
	if reads == 0 {
		t.Error("the body of the blob was never read")
	}
	placed, err := p.b.Vaults.With(target).StatFile(t.Context(), trip.GetPath())
	if err != nil || placed.Hash != stat.Hash || placed.Size != stat.Size {
		t.Errorf("placed file = %+v (%v), want the file of hash %s", placed, err, stat.Hash)
	}
	places := spy.GetPlaces()
	if len(places) != 1 || places[0].Hash != stat.Hash || !slices.Equal(result.Applied, []string{trip.GetPath()}) {
		t.Errorf("placements = %+v, applied %v; want one placement of the whole file", places, result.Applied)
	}
	if names := p.getStagedNames(t); len(names) != 0 {
		t.Errorf("staged files left = %v, want none", names)
	}
}
