package syncvault_test

import (
	"bytes"
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
)

const (
	pdf     = "Papers/Report.pdf"
	pdfText = "%PDF-1.7 \xff\x00 the bytes of a report"
)

func (p *pair) heads(t *testing.T) []*syncv1.StoredRevision {
	t.Helper()
	return p.a.ReadChanges(t, p.a.GetRemoteID(t, source))
}

func (p *pair) getLive(t *testing.T) []*syncv1.StoredRevision {
	t.Helper()
	return slices.DeleteFunc(p.heads(t), func(one *syncv1.StoredRevision) bool { return one.GetRevision().GetDeleted() })
}

func (p *pair) countCalls(procedure string) int { return p.server.Fake.CountCalls(procedure) }

// afterDelay lets a removed file wait the time the passes of the tests watch it for.
func (p *pair) afterDelay() { p.a.Clock.Advance(synctest.RemovalDelay) }

func (p *pair) writeMany(t *testing.T, count int) {
	t.Helper()
	for i := range count {
		p.a.WriteNote(source, fmt.Sprintf("Papers/File %d.pdf", i+1), fmt.Sprintf("file %d", i+1))
	}
}

func (p *pair) removeMany(count int) {
	for i := range count {
		p.a.Vaults.With(source).Remove(fmt.Sprintf("Papers/File %d.pdf", i+1))
	}
}

func wantNoFile(t *testing.T, c *synctest.Computer, id domain.VaultID, path string) {
	t.Helper()
	if got, ok := c.ReadNote(id, path); ok {
		t.Errorf("file %s of %s = %q, want none", path, id, got)
	}
}

func TestSyncVault_AFileOfAnyKindArrivesWithItsKindAndTheServerKnowsNoName(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	for path, text := range map[string]string{pdf: pdfText, "Data/list.txt": "milk\nbread\n", "Notes/Plan.md": "tea"} {
		p.a.WriteNote(source, path, text)
	}

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, pdf, pdfText)
	wantNote(t, p.b, target, "Data/list.txt", "milk\nbread\n")
	kinds := map[string]string{}
	for _, one := range p.heads(t) {
		meta := synctest.OpenMeta(t, p.a.GetVaultKey(t, source), one.GetRevision())
		kinds[meta.Path] = meta.Kind
	}
	if kinds[pdf] != "binary" || kinds["Data/list.txt"] != "text" || kinds["Notes/Plan.md"] != "markdown" {
		t.Errorf("kinds = %v, want binary, text and markdown by extension", kinds)
	}
	stored := p.server.GetStoredBlobs(t)
	for _, plain := range []string{"Report.pdf", "Papers", "the bytes of a report"} {
		if bytes.Contains(stored, []byte(plain)) {
			t.Errorf("the server holds %q as plain text", plain)
		}
	}
}

func TestSyncVault_AChangedFileChangesOnTheSecondComputerAndAFileOfTheSecondAppearsOnTheFirst(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.WriteNote(source, pdf, pdfText+" changed")
	p.b.WriteNote(target, "Papers/Plan.pdf", "plan bytes")

	sync(t, p.a, source)
	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.b, target, pdf, pdfText+" changed")
	wantNote(t, p.a, source, "Papers/Plan.pdf", "plan bytes")
}

func TestSyncVault_DoesNotSendAFileBackThatItPlacedItself(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	reserved := p.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure)
	committed := p.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure)

	sync(t, p.b, target)

	if got := p.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != reserved {
		t.Errorf("reservations grew from %d to %d, want none for a file the pull placed", reserved, got)
	}
	if got := p.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got != committed {
		t.Errorf("commits grew from %d to %d, want none for a file the pull placed", committed, got)
	}
}

func TestSyncVault_ARenamedFileIsRenamedOnTheSecondComputerWithoutSendingItsBytesAgain(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	blob := p.heads(t)[0].GetRevision().GetBlobId()
	reserved := p.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure)
	if _, err := p.a.Vaults.With(source).MoveFile(t.Context(), pdf, "Papers/Old report.pdf", nil); err != nil {
		t.Fatal(err)
	}

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, "Papers/Old report.pdf", pdfText)
	wantNoFile(t, p.b, target, pdf)
	heads := p.heads(t)
	if len(heads) != 1 || !bytes.Equal(heads[0].GetRevision().GetBlobId(), blob) {
		t.Errorf("heads = %d, want one file that still names its first blob", len(heads))
	}
	if got := p.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != reserved {
		t.Errorf("reservations grew from %d to %d, want none for a rename", reserved, got)
	}
	if meta := synctest.OpenMeta(t, p.a.GetVaultKey(t, source), heads[0].GetRevision()); meta.Path != "Papers/Old report.pdf" {
		t.Errorf("the server's metadata names %q, want the new path", meta.Path)
	}
}

func TestSyncVault_ARenamedFolderIsRenamedOnTheSecondComputerInOneStep(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	spy := p.b.Spy()
	p.a.WriteNote(source, "Food/Menu.pdf", "menu bytes")
	p.a.WriteNote(source, "Food/Shopping list.md", "milk")
	p.a.WriteNote(source, "Food/Map.png", "map bytes")
	sync(t, p.a, source)
	sync(t, p.b, target)
	commits := p.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure)
	reserved := p.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure)
	if _, err := p.a.Vaults.With(source).MoveFile(t.Context(), "Food", "Meals", nil); err != nil {
		t.Fatal(err)
	}

	sync(t, p.a, source)

	if got := p.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got != commits+1 {
		t.Errorf("commit calls grew from %d to %d, want one for the whole folder", commits, got)
	}
	sync(t, p.b, target)

	if moves := spy.GetMoves(); len(moves) != 1 || moves[0].From != "Food" || moves[0].To != "Meals" {
		t.Errorf("moves = %+v, want one move of the folder", moves)
	}
	wantNote(t, p.b, target, "Meals/Menu.pdf", "menu bytes")
	wantNote(t, p.b, target, "Meals/Shopping list.md", "milk")
	wantNote(t, p.b, target, "Meals/Map.png", "map bytes")
	wantNoFile(t, p.b, target, "Food/Menu.pdf")
	if got := p.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != reserved {
		t.Errorf("reservations grew from %d to %d, want none for a folder that moved", reserved, got)
	}
	if got := p.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got != commits+1 {
		t.Errorf("commit calls = %d, want the second computer to send nothing after it moved the folder", got)
	}
}

func TestSyncVault_ARemovedFileIsRemovedOnTheSecondComputerOnceItStayedMissingForTheDelay(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove(pdf)

	sync(t, p.a, source)

	if live := p.getLive(t); len(live) != 1 {
		t.Fatalf("live files on the server after the first look = %d, want the removal to wait", len(live))
	}
	p.afterDelay()
	sync(t, p.a, source)
	sync(t, p.b, target)

	if live := p.getLive(t); len(live) != 0 {
		t.Errorf("live files on the server = %d, want the file removed", len(live))
	}
	wantNoFile(t, p.b, target, pdf)
	if trashed := p.b.Vaults.With(target).Trashed(); !slices.Equal(trashed, []string{pdf}) {
		t.Errorf("trashed on the second computer = %v, want the file", trashed)
	}
	if _, ok := p.b.GetFileRow(t, target, pdf); ok {
		t.Error("the second computer still has a record of the removed file")
	}
}

func TestSyncVault_ARemovedNoteAndARemovedFolderAreRemovedOnTheSecondComputer(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	p.a.WriteNote(source, "Papers/Report.pdf", pdfText)
	p.a.WriteNote(source, "Papers/Plan.pdf", "plan")
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove("Shopping list.md")
	p.a.Vaults.With(source).Remove("Papers/Report.pdf")
	p.a.Vaults.With(source).Remove("Papers/Plan.pdf")

	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	sync(t, p.b, target)

	for _, path := range []string{"Shopping list.md", "Papers/Report.pdf", "Papers/Plan.pdf"} {
		wantNoFile(t, p.b, target, path)
	}
}

func TestSyncVault_AFileRemovedAndWrittenAgainAtItsPathIsANewFile(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	first := p.heads(t)[0]
	p.a.Vaults.With(source).Remove(pdf)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.WriteNote(source, pdf, pdfText+" again")

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, pdf, pdfText+" again")
	live := p.getLive(t)
	if len(live) != 1 || bytes.Equal(live[0].GetRevision().GetFileId(), first.GetRevision().GetFileId()) || len(live[0].GetRevision().GetParentRev()) != 0 {
		t.Errorf("live files = %d, want one new file with no parent", len(live))
	}
}

func TestSyncVault_AFileRemovedAndWrittenAgainWithinTheDelayIsAChangeAndNotARemoval(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove(pdf)
	sync(t, p.a, source)
	p.a.WriteNote(source, pdf, pdfText+" again")
	p.afterDelay()

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, pdf, pdfText+" again")
	heads := p.heads(t)
	if len(heads) != 1 || heads[0].GetRevision().GetDeleted() {
		t.Errorf("heads = %d, want the one file changed and never removed", len(heads))
	}
}

func TestSyncVault_AChangeOnTheSecondComputerBeatsTheRemovalOnTheFirstWhenThePullSeesTheRemoval(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove(pdf)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	tombstone := p.heads(t)[0].GetRevId()
	p.b.WriteNote(target, pdf, pdfText+" edited there")

	sync(t, p.b, target)
	sync(t, p.a, source)

	live := p.getLive(t)
	if len(live) != 1 || !bytes.Equal(live[0].GetRevision().GetParentRev(), tombstone) {
		t.Fatalf("live files = %d, want the edit committed as a child of the removal", len(live))
	}
	wantNote(t, p.a, source, pdf, pdfText+" edited there")
	wantNote(t, p.b, target, pdf, pdfText+" edited there")
}

func TestSyncVault_AChangeBeatsARemovalThatArrivesBetweenThePullAndThePush(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.b.WriteNote(target, pdf, pdfText+" edited there")
	var tombstone []byte
	p.server.AfterNext("/GetChanges", func() {
		records, err := p.a.State.ReadRecords(t.Context(), source)
		if err != nil {
			t.Error(err)
			return
		}
		vault := transfer.Vault{ID: p.a.GetRemoteID(t, source), Key: p.a.GetVaultKey(t, source)}
		outcomes, err := commit.NewCommitRemovals(p.a.CommitDeps()).Execute(t.Context(), transfer.RemoveInput{Vault: vault, Records: records})
		if err != nil || len(outcomes) != 1 || outcomes[0].Status != transfer.Committed {
			t.Errorf("removal = %+v (%v), want committed", outcomes, err)
			return
		}
		tombstone = outcomes[0].RevID
	})

	sync(t, p.b, target)

	live := p.getLive(t)
	if len(live) != 1 || !bytes.Equal(live[0].GetRevision().GetParentRev(), tombstone) || len(tombstone) == 0 {
		t.Fatalf("live files = %d, want the edit committed again as a child of the removal", len(live))
	}
	if row, ok := p.b.GetFileRow(t, target, pdf); !ok || row.BaseHash != note.Hash([]byte(pdfText+" edited there")) {
		t.Errorf("row = %+v (there: %v), want the edit as the base", row, ok)
	}
}

func TestSyncVault_AChangeBeatsARemovalTheServerPrunedAsANewRevisionOfTheSameFile(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	fileID := p.heads(t)[0].GetRevision().GetFileId()
	p.b.WriteNote(target, pdf, pdfText+" edited there")
	p.server.AfterNext("/GetChanges", func() {
		records, err := p.a.State.ReadRecords(t.Context(), source)
		if err != nil {
			t.Error(err)
			return
		}
		vault := transfer.Vault{ID: p.a.GetRemoteID(t, source), Key: p.a.GetVaultKey(t, source)}
		if _, err := commit.NewCommitRemovals(p.a.CommitDeps()).Execute(t.Context(), transfer.RemoveInput{Vault: vault, Records: records}); err != nil {
			t.Error(err)
		}
		p.server.Fake.PruneTombstones()
	})

	sync(t, p.b, target)

	live := p.getLive(t)
	if len(live) != 1 || !bytes.Equal(live[0].GetRevision().GetFileId(), fileID) || len(live[0].GetRevision().GetParentRev()) != 0 {
		t.Fatalf("live files = %d, want one revision of the same file with no parent", len(live))
	}
}

func TestSyncVault_ARemovalIsNotAppliedWhenTheOtherComputerChangedTheFileMeanwhile(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove(pdf)
	sync(t, p.a, source)
	p.b.WriteNote(target, pdf, pdfText+" edited there")
	sync(t, p.b, target)
	p.afterDelay()

	sync(t, p.a, source)

	heads := p.heads(t)
	if len(heads) != 1 || heads[0].GetRevision().GetDeleted() {
		t.Fatalf("heads = %d, want the file left on the server", len(heads))
	}
	wantNote(t, p.a, source, pdf, pdfText+" edited there")
}

func TestSyncVault_ManyRemovedFilesPauseTheSyncAndSendNothingUntilTheAnswer(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.writeMany(t, 10)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.removeMany(10)
	sync(t, p.a, source)
	p.afterDelay()

	sync(t, p.a, source)
	p.a.WriteNote(source, "Papers/New.pdf", "new")
	sync(t, p.a, source)
	sync(t, p.b, target)

	paused, err := p.a.State.ReadPending(t.Context(), source)
	if err != nil || paused == nil || paused.Direction != transfer.DirectionSend || len(paused.Files) != 10 || paused.Tracked != 10 {
		t.Fatalf("pending = %+v (%v), want a pause of 10 removals to send of 10 tracked", paused, err)
	}
	if !p.a.HasItem("Sync is paused: 10 of 10 files are removed") {
		t.Errorf("health items = %+v, want the pause", p.a.Health.List())
	}
	if live := p.getLive(t); len(live) != 10 {
		t.Errorf("live files on the server = %d, want the 10 untouched and the new file not sent", len(live))
	}
	for i := range 10 {
		wantNote(t, p.b, target, fmt.Sprintf("Papers/File %d.pdf", i+1), fmt.Sprintf("file %d", i+1))
	}
}

func TestSyncVault_TheRemovalOfManyFilesIsSentOnceThePersonConfirmsItOnEachComputer(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.writeMany(t, 10)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.removeMany(10)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	if _, err := deletions.NewResolveDeletions(p.a.DeleteDeps()).Execute(t.Context(), source, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	sync(t, p.a, source)
	sync(t, p.b, target)

	if live := p.getLive(t); len(live) != 0 {
		t.Errorf("live files on the server = %d, want none", len(live))
	}
	wantNote(t, p.b, target, "Papers/File 1.pdf", "file 1")
	if _, err := deletions.NewResolveDeletions(p.b.DeleteDeps()).Execute(t.Context(), target, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}

	sync(t, p.b, target)

	for i := range 10 {
		wantNoFile(t, p.b, target, fmt.Sprintf("Papers/File %d.pdf", i+1))
	}
	if p.a.HasItemKey(deletions.HealthKey) || p.b.HasItemKey(deletions.HealthKey) {
		t.Error("the pause is still shown after the answers")
	}
}

func TestSyncVault_TheFilesAreRestoredWhenThePersonKeepsThemAfterTheyWereRemovedHere(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.writeMany(t, 10)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.removeMany(10)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	if _, err := deletions.NewResolveDeletions(p.a.DeleteDeps()).Execute(t.Context(), source, transfer.ChoiceKeep); err != nil {
		t.Fatal(err)
	}

	sync(t, p.a, source)

	for i := range 10 {
		wantNote(t, p.a, source, fmt.Sprintf("Papers/File %d.pdf", i+1), fmt.Sprintf("file %d", i+1))
	}
	if live := p.getLive(t); len(live) != 10 {
		t.Errorf("live files on the server = %d, want all 10", len(live))
	}
	if pending, err := p.a.State.ReadPending(t.Context(), source); err != nil || pending != nil {
		t.Errorf("pending = %+v (%v), want none after the answer", pending, err)
	}
}

func TestSyncVault_AFewRemovedFilesDoNotPauseTheSync(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.writeMany(t, 10)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove("Papers/File 1.pdf")
	sync(t, p.a, source)
	p.afterDelay()

	sync(t, p.a, source)
	sync(t, p.b, target)

	if pending, err := p.a.State.ReadPending(t.Context(), source); err != nil || pending != nil {
		t.Errorf("pending = %+v (%v), want none for one file of ten", pending, err)
	}
	wantNoFile(t, p.b, target, "Papers/File 1.pdf")
	wantNote(t, p.b, target, "Papers/File 2.pdf", "file 2")
}

func TestSyncVault_ManyFilesRemovedOnTheOtherComputerPauseThisOneAndAreRestoredWhenThePersonKeepsThem(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.writeMany(t, 10)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.removeMany(10)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	if _, err := deletions.NewResolveDeletions(p.a.DeleteDeps()).Execute(t.Context(), source, transfer.ChoiceRemove); err != nil {
		t.Fatal(err)
	}
	sync(t, p.a, source)

	sync(t, p.b, target)

	paused, err := p.b.State.ReadPending(t.Context(), target)
	if err != nil || paused == nil || paused.Direction != transfer.DirectionApply || len(paused.Files) != 10 {
		t.Fatalf("pending = %+v (%v), want a pause of 10 removals to apply", paused, err)
	}
	if !p.b.HasItem("Sync is paused: 10 of 10 files were removed on another computer") {
		t.Errorf("health items = %+v, want the pause", p.b.Health.List())
	}
	wantNote(t, p.b, target, "Papers/File 1.pdf", "file 1")
	if _, err := deletions.NewResolveDeletions(p.b.DeleteDeps()).Execute(t.Context(), target, transfer.ChoiceKeep); err != nil {
		t.Fatal(err)
	}

	sync(t, p.b, target)
	sync(t, p.a, source)

	if live := p.getLive(t); len(live) != 10 {
		t.Errorf("live files on the server = %d, want the 10 kept files sent again as edits", len(live))
	}
	for i := range 10 {
		wantNote(t, p.a, source, fmt.Sprintf("Papers/File %d.pdf", i+1), fmt.Sprintf("file %d", i+1))
	}
}

func TestSyncVault_DeletesAStagedFileNoDownloadNamesAtTheStartOfAPassEvenWhenTheRemovalsWait(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.writeMany(t, 10)
	sync(t, p.a, source)
	p.removeMany(10)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	p.a.Vaults.Store().With(source).File("0badc0de.part", []byte("left behind"))

	sync(t, p.a, source)

	files, err := p.a.Host.Store(source).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Name == "0badc0de.part" {
			t.Error("a staged file that no download names was left by a pass whose removals wait")
		}
	}
}

func TestSyncVault_StopsAtAServerThatAsksForAFullCheckAndSendsNothing(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, pdf, pdfText)
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.Vaults.With(source).Remove(pdf)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	p.server.Fake.PruneTombstones()
	p.b.WriteNote(target, "Papers/New.pdf", "new")
	heads := len(p.heads(t))

	sync(t, p.b, target)

	if !p.b.HasItem("Sync needs to compare the vault again") {
		t.Errorf("health items = %+v, want the full check asked for", p.b.Health.List())
	}
	if got := len(p.heads(t)); got != heads {
		t.Errorf("heads grew from %d to %d, want nothing sent in a pass that stops at a reset", heads, got)
	}
}
