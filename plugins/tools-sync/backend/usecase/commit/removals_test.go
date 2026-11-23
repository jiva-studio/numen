package commit_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
)

func TestCommitRemovals_CommitsATombstoneOnTopOfTheHeadOfTheRecord(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	committed := s.run(t, transfer.File{Path: "Shopping list.md", Content: []byte("milk"), ModifiedAt: 1})[0]
	record := note.FileRecord{FileID: committed.FileID, Path: "Shopping list.md", BaseHash: note.Hash([]byte("milk")), HeadRev: committed.RevID}

	outcomes, err := commit.NewCommitRemovals(s.c.CommitDeps()).Execute(t.Context(), transfer.RemoveInput{Vault: s.vault, Records: []note.FileRecord{record}})

	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].Status != transfer.Committed || !bytes.Equal(outcomes[0].FileID, record.FileID) {
		t.Fatalf("outcomes = %+v, want one committed removal of the file", outcomes)
	}
	head := s.read(t)[0]
	if !head.GetRevision().GetDeleted() || !bytes.Equal(head.GetRevision().GetParentRev(), committed.RevID) || !bytes.Equal(head.GetRevId(), outcomes[0].RevID) {
		t.Errorf("head = %+v, want a tombstone on top of the committed revision", head)
	}
}

func TestCommitRemovals_AnswersParentMismatchWhenTheHeadMovedOn(t *testing.T) {
	s := newSetup(t, fakeserver.Config{})
	committed := s.run(t, transfer.File{Path: "Shopping list.md", Content: []byte("milk"), ModifiedAt: 1})[0]
	s.run(t, transfer.File{FileID: committed.FileID, ParentRev: committed.RevID, Path: "Shopping list.md", Content: []byte("bread"), ModifiedAt: 2})
	record := note.FileRecord{FileID: committed.FileID, Path: "Shopping list.md", BaseHash: note.Hash([]byte("milk")), HeadRev: committed.RevID}

	outcomes, err := commit.NewCommitRemovals(s.c.CommitDeps()).Execute(t.Context(), transfer.RemoveInput{Vault: s.vault, Records: []note.FileRecord{record}})

	if err != nil || len(outcomes) != 1 || outcomes[0].Status != transfer.ParentMismatch || outcomes[0].Head == nil {
		t.Errorf("outcomes = %+v, %v, want ParentMismatch naming the head", outcomes, err)
	}
	if s.read(t)[0].GetRevision().GetDeleted() {
		t.Error("the note was removed on a head it was not changed from")
	}
}
