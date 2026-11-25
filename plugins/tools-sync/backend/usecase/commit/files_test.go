package commit_test

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const (
	vaultID  = "Test"
	bigSize  = 65 << 20
	pdfBytes = "%PDF-1.7 the bytes of a report"
)

func (s *setup) move(t *testing.T, moves ...transfer.Move) []transfer.Outcome {
	t.Helper()
	outcomes, err := commit.NewCommitMoves(s.c.CommitDeps()).Execute(t.Context(), transfer.MoveInput{Vault: s.vault, Moves: moves})
	if err != nil {
		t.Fatal(err)
	}
	return outcomes
}

func (s *setup) remove(t *testing.T, records ...transfer.FileRecord) []transfer.Outcome {
	t.Helper()
	outcomes, err := commit.NewCommitRemovals(s.c.CommitDeps()).Execute(t.Context(), transfer.RemoveInput{Vault: s.vault, Records: records})
	if err != nil {
		t.Fatal(err)
	}
	return outcomes
}

func (s *setup) countCalls(procedure string) int { return s.server.Fake.CountCalls(procedure) }

func TestCommitMoves_CommitsAMetadataOnlyRevisionThatNamesTheBlobOfTheHead(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	first := s.run(t, transfer.File{Path: "Papers/Report.pdf", Content: []byte(pdfBytes), ModifiedAt: 1_700_000_005_000_000_000})[0]
	before := s.read(t)[0].GetRevision()
	reserved := s.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure)
	held := len(s.server.GetStoredBlobs(t))

	outcomes := s.move(t, transfer.Move{Record: first.Record, To: "Papers/Old report.pdf"})

	if len(outcomes) != 1 || outcomes[0].Status != transfer.Committed || outcomes[0].Path != "Papers/Old report.pdf" {
		t.Fatalf("outcomes = %+v, want one committed at the new path", outcomes)
	}
	stored := s.read(t)
	if len(stored) != 1 || !bytes.Equal(stored[0].GetRevId(), outcomes[0].RevID) {
		t.Fatalf("the server holds %d heads, want the one of the move", len(stored))
	}
	revision := stored[0].GetRevision()
	if !bytes.Equal(revision.GetBlobId(), before.GetBlobId()) || !bytes.Equal(revision.GetBlobHash(), before.GetBlobHash()) || revision.GetBlobSize() != before.GetBlobSize() {
		t.Error("the revision of the move does not name the blob of the head, with its hash and size")
	}
	if !bytes.Equal(revision.GetParentRev(), first.RevID) || revision.GetDeleted() {
		t.Errorf("revision parent %x, deleted %v; want the first revision as parent and no removal", revision.GetParentRev(), revision.GetDeleted())
	}
	if !bytes.Equal(revision.GetPathHash(), note.HashPath(s.vault.Key, "Papers/Old report.pdf")) {
		t.Error("the path hash is not the hash of the new path")
	}
	if !identity.VerifyForPlugin(revision.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID, syncprotocol.GetRevisionMessage(revision), [64]byte(revision.GetSignature())) {
		t.Error("the signature of the revision of the move does not verify")
	}
	oldMeta, newMeta := synctest.OpenMeta(t, s.vault.Key, before), synctest.OpenMeta(t, s.vault.Key, revision)
	if newMeta.Path != "Papers/Old report.pdf" || newMeta.Hash != oldMeta.Hash || newMeta.Size != oldMeta.Size || !bytes.Equal(newMeta.Key, oldMeta.Key) || newMeta.Kind != "binary" {
		t.Errorf("meta = %+v, want the new path with the hash, the size, the kind and the key of the head", newMeta)
	}
	if got := s.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != reserved {
		t.Errorf("reservations grew from %d to %d, want none for a move", reserved, got)
	}
	if got := len(s.server.GetStoredBlobs(t)); got != held {
		t.Errorf("the server holds %d bytes of blobs after the move, want the %d it held", got, held)
	}
	if plain := s.c.OpenBlob(t, s.vault.Key, revision); string(plain) != pdfBytes {
		t.Errorf("the blob of the moved file opens to %q, want the file", plain)
	}
	if outcomes[0].Record.Path != "Papers/Old report.pdf" || !bytes.Equal(outcomes[0].Record.HeadRev, outcomes[0].RevID) || outcomes[0].Record.BaseHash != first.Record.BaseHash {
		t.Errorf("record = %+v, want the new path and head with the base unchanged", outcomes[0].Record)
	}
}

func TestCommitMoves_CommitsAMovedFolderInOneCall(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	files := make([]transfer.File, 7)
	for i := range files {
		files[i] = transfer.File{Path: fmt.Sprintf("Food/File %d.pdf", i+1), Content: []byte(fmt.Sprintf("file %d", i+1))}
	}
	pushed := s.run(t, files...)
	moves := make([]transfer.Move, len(pushed))
	for i, one := range pushed {
		moves[i] = transfer.Move{Record: one.Record, To: fmt.Sprintf("Meals/File %d.pdf", i+1)}
	}
	calls := s.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure)
	reserved := s.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure)

	outcomes := s.move(t, moves...)

	for i, outcome := range outcomes {
		if outcome.Status != transfer.Committed || outcome.Path != moves[i].To {
			t.Fatalf("outcome %d = %+v, want committed at %s, in the order of the moves", i, outcome, moves[i].To)
		}
	}
	if got := s.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got != calls+1 {
		t.Errorf("commit calls grew from %d to %d, want one call for the whole folder", calls, got)
	}
	if got := s.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != reserved {
		t.Errorf("reservations grew from %d to %d, want none for a folder that moved", reserved, got)
	}
	for _, one := range s.read(t) {
		if meta := synctest.OpenMeta(t, s.vault.Key, one.GetRevision()); len(meta.Path) < 6 || meta.Path[:6] != "Meals/" {
			t.Errorf("a file is at %q on the server, want every file of the folder under Meals", meta.Path)
		}
	}
}

func TestCommitMoves_ReportsAPathThatAnotherFileUses(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	pushed := s.run(t, transfer.File{Path: "a.pdf", Content: []byte("a")}, transfer.File{Path: "b.pdf", Content: []byte("b")})

	outcomes := s.move(t, transfer.Move{Record: pushed[0].Record, To: "b.pdf"})

	if outcomes[0].Status != transfer.PathTaken || !bytes.Equal(outcomes[0].ExistingFileID, pushed[1].FileID) {
		t.Errorf("outcome = %+v, want PathTaken by the file b.pdf", outcomes[0])
	}
	for _, one := range s.read(t) {
		if bytes.Equal(one.GetRevision().GetFileId(), pushed[0].FileID) && !bytes.Equal(one.GetRevId(), pushed[0].RevID) {
			t.Error("the head of a.pdf changed after a move the server refused")
		}
	}
}

func TestCommitMoves_ReportsAFileWhoseHeadIsAnotherRevision(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	first := s.run(t, transfer.File{Path: "a.pdf", Content: []byte("a")})[0]
	second := s.run(t, transfer.File{FileID: first.FileID, ParentRev: first.RevID, Path: "a.pdf", Content: []byte("a changed")})[0]

	outcomes := s.move(t, transfer.Move{Record: first.Record, To: "Old/a.pdf"})

	if outcomes[0].Status != transfer.ParentMismatch || outcomes[0].Head == nil || !bytes.Equal(outcomes[0].Head.GetRevId(), second.RevID) {
		t.Errorf("outcome = %+v, want ParentMismatch with the changed revision as head", outcomes[0])
	}
}

func TestCommitRemovals_CommitsATombstoneWithNoBlobAndNoMetadata(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	first := s.run(t, transfer.File{Path: "Papers/Report.pdf", Content: []byte(pdfBytes)})[0]

	outcomes := s.remove(t, first.Record)

	if len(outcomes) != 1 || outcomes[0].Status != transfer.Committed || outcomes[0].Path != "Papers/Report.pdf" {
		t.Fatalf("outcomes = %+v, want one committed removal", outcomes)
	}
	stored := s.read(t)
	if len(stored) != 1 || !stored[0].GetRevision().GetDeleted() || !bytes.Equal(stored[0].GetRevId(), outcomes[0].RevID) {
		t.Fatalf("heads = %v, want one tombstone", stored)
	}
	revision := stored[0].GetRevision()
	if len(revision.GetBlobId()) != 0 || len(revision.GetBlobHash()) != 0 || revision.GetBlobSize() != 0 || len(revision.GetEncMeta()) != 0 {
		t.Errorf("tombstone has a blob %x or metadata of %d bytes, want neither", revision.GetBlobId(), len(revision.GetEncMeta()))
	}
	if !bytes.Equal(revision.GetParentRev(), first.RevID) || !bytes.Equal(revision.GetFileId(), first.FileID) {
		t.Error("the tombstone is not a child of the head of the file it removes")
	}
	if !bytes.Equal(revision.GetPathHash(), note.HashPath(s.vault.Key, "Papers/Report.pdf")) {
		t.Error("the path hash of the tombstone is not the hash of the removed path")
	}
	if !identity.VerifyForPlugin(revision.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID, syncprotocol.GetRevisionMessage(revision), [64]byte(revision.GetSignature())) {
		t.Error("the signature of the tombstone does not verify")
	}
	if outcomes[0].Record.Path != "Papers/Report.pdf" || !bytes.Equal(outcomes[0].Record.FileID, first.FileID) {
		t.Errorf("record = %+v, want the record of the removed file", outcomes[0].Record)
	}
}

func TestCommitRemovals_DoesNotRemoveAFileWhoseHeadChangedSinceTheRecord(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	first := s.run(t, transfer.File{Path: "Papers/Report.pdf", Content: []byte(pdfBytes)})[0]
	second := s.run(t, transfer.File{FileID: first.FileID, ParentRev: first.RevID, Path: "Papers/Report.pdf", Content: []byte(pdfBytes + " edited")})[0]

	outcomes := s.remove(t, first.Record)

	if outcomes[0].Status != transfer.ParentMismatch || outcomes[0].Head == nil || outcomes[0].Head.GetRevision().GetDeleted() || !bytes.Equal(outcomes[0].Head.GetRevId(), second.RevID) {
		t.Errorf("outcome = %+v, want ParentMismatch with the live changed revision as head", outcomes[0])
	}
	if stored := s.read(t); len(stored) != 1 || stored[0].GetRevision().GetDeleted() || !bytes.Equal(stored[0].GetRevId(), second.RevID) {
		t.Error("the server removed a file that was changed after the record")
	}
}

func TestCommitRemovals_CommitsManyRemovalsInOneCall(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	files := make([]transfer.File, 205)
	for i := range files {
		files[i] = transfer.File{Path: fmt.Sprintf("n%03d.pdf", i), Content: []byte(fmt.Sprintf("file %d", i))}
	}
	records := make([]transfer.FileRecord, 0, len(files))
	for _, one := range s.run(t, files...) {
		records = append(records, one.Record)
	}
	calls := s.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure)

	outcomes := s.remove(t, records...)

	if got := s.countCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got != calls+1 {
		t.Errorf("commit calls grew from %d to %d, want one call for 205 removals", calls, got)
	}
	for i, outcome := range outcomes {
		if outcome.Status != transfer.Committed || outcome.Path != files[i].Path {
			t.Fatalf("outcome %d = %+v, want a committed removal of %s in order", i, outcome, files[i].Path)
		}
	}
	for _, head := range s.read(t) {
		if !head.GetRevision().GetDeleted() {
			t.Error("a file is still live on the server after the removal of all")
			break
		}
	}
}

func (s *setup) runBig(t *testing.T, file transfer.File) []transfer.Outcome {
	t.Helper()
	outcomes, err := commit.NewCommitRevisions(s.c.CommitDeps()).Execute(t.Context(), ports.CommitInput{VaultID: vaultID, Vault: s.vault, Files: []transfer.File{file}})
	if err != nil {
		t.Fatal(err)
	}
	return outcomes
}

func TestCommitRevisions_SendsAFileAboveFourMiBThroughTheUploadInOneBodyWhenItFits(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	content := make([]byte, 5<<20)
	if _, err := rand.Read(content); err != nil {
		t.Fatal(err)
	}
	s.c.WriteNote(vaultID, "Videos/Clip.bin", string(content))
	stat, err := s.c.Vaults.With(vaultID).StatFile(t.Context(), "Videos/Clip.bin")
	if err != nil {
		t.Fatal(err)
	}

	outcomes := s.runBig(t, transfer.File{Path: "Videos/Clip.bin", ModifiedAt: stat.ModTime.UnixNano(), Stat: &stat})

	if len(outcomes) != 1 || outcomes[0].Status != transfer.Committed {
		t.Fatalf("outcomes = %+v, want one committed", outcomes)
	}
	if got := s.countCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != 1 {
		t.Errorf("single reservations = %d, want 1: a file of 5 MiB fits one body", got)
	}
	stored := s.read(t)
	if got := s.c.OpenBlob(t, s.vault.Key, stored[0].GetRevision()); !bytes.Equal(got, content) {
		t.Errorf("the blob opens to %d bytes, want the file of %d bytes", len(got), len(content))
	}
	meta := synctest.OpenMeta(t, s.vault.Key, stored[0].GetRevision())
	if meta.Hash != stat.Hash || meta.Size != stat.Size || meta.Kind != "binary" || meta.Path != "Videos/Clip.bin" {
		t.Errorf("meta = %+v, want the path, the hash %s and the size %d of the file", meta, stat.Hash, stat.Size)
	}
}

func TestCommitRevisions_SendsABigFileInPartsAndForgetsItsUploadOnceCommitted(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	source := synctest.NewSource("Videos/Trip.bin", bigSize, 1, synctest.GetStart().Add(-time.Hour))
	s.c.Spy().Serve(source)
	stat := source.GetStat()

	outcomes := s.runBig(t, transfer.File{Path: source.GetPath(), ModifiedAt: stat.ModTime.UnixNano(), Stat: &stat})

	if len(outcomes) != 1 || outcomes[0].Status != transfer.Committed {
		t.Fatalf("outcomes = %+v, want one committed", outcomes)
	}
	if got := s.countCalls(syncv1connect.BlobServiceReserveMultipartBlobProcedure); got != 1 {
		t.Errorf("multipart reservations = %d, want 1", got)
	}
	revision := s.read(t)[0].GetRevision()
	meta := synctest.OpenMeta(t, s.vault.Key, revision)
	if meta.Hash != stat.Hash || meta.Size != bigSize || revision.GetBlobSize() != outcomes[0].Record.BlobSize || revision.GetBlobSize() <= 64<<20 {
		t.Errorf("meta = %+v, blob size %d; want the hash and the size of the file and a stored size above 64 MiB", meta, revision.GetBlobSize())
	}
	if kept, err := s.c.Host.Store(vaultID).Read(t.Context(), "uploads.json"); err == nil {
		var state struct {
			Uploads []transfer.UploadEntry `json:"uploads"`
		}
		if err := json.Unmarshal(kept, &state); err != nil || len(state.Uploads) != 0 {
			t.Errorf("uploads.json holds %d entries (%v) after the commit, want none", len(state.Uploads), err)
		}
	} else if !errors.Is(err, host.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestCommitRevisions_KeepsTheUploadOfABigFileWhenTheServerRefusesItsParent(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	first := s.run(t, transfer.File{Path: "Videos/Trip.bin", Content: []byte("small")})[0]
	source := synctest.NewSource("Videos/Trip.bin", bigSize, 2, synctest.GetStart().Add(-time.Hour))
	s.c.Spy().Serve(source)
	stat := source.GetStat()
	wrong := make([]byte, 32)
	if _, err := rand.Read(wrong); err != nil {
		t.Fatal(err)
	}

	outcomes := s.runBig(t, transfer.File{FileID: first.FileID, ParentRev: wrong, Path: source.GetPath(), ModifiedAt: stat.ModTime.UnixNano(), Stat: &stat})

	if outcomes[0].Status != transfer.ParentMismatch || outcomes[0].Head == nil || !bytes.Equal(outcomes[0].Head.GetRevId(), first.RevID) {
		t.Fatalf("outcome = %+v, want ParentMismatch with the first revision as head", outcomes[0])
	}
	kept, err := s.c.Host.Store(vaultID).Read(t.Context(), "uploads.json")
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Uploads []transfer.UploadEntry `json:"uploads"`
	}
	if err := json.Unmarshal(kept, &state); err != nil || len(state.Uploads) != 1 || state.Uploads[0].PartsDone != state.Uploads[0].PartCount {
		t.Errorf("uploads.json = %s, want the finished upload kept until the pass decides", kept)
	}
}

func TestCommitRevisions_ReportsAFileThatIsTooLargeForTheServerAndContinues(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	huge := domain.FileStat{Path: "Videos/Huge.bin", Size: syncprotocol.MaxMultipartSize + 1, ModTime: synctest.GetStart(), Hash: "not read"}

	outcomes := s.runBig(t, transfer.File{Path: "Videos/Huge.bin", ModifiedAt: 1, Stat: &huge})

	if len(outcomes) != 1 || outcomes[0].Status != transfer.TooLarge || outcomes[0].Path != "Videos/Huge.bin" {
		t.Errorf("outcomes = %+v, want the file reported as too large", outcomes)
	}
	if stored := s.read(t); len(stored) != 0 {
		t.Errorf("the server holds %d heads, want none", len(stored))
	}
}
