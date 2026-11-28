package commit_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/wiretest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
)

type setup struct {
	c      *synctest.Computer
	server *synctest.Server
	vault  transfer.Vault
}

func newSetup(t *testing.T) *setup {
	t.Helper()
	return newSetupOnPlan(t, synctest.DefaultPlan)
}

func newSetupOnPlan(t *testing.T, plan string) *setup {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock)
	c := synctest.NewFirstComputerOnPlan(t, server, clock, "Test", synctest.NewWords(t), plan)
	c.Connect(t, "Test")
	return &setup{c: c, server: server, vault: transfer.Vault{ID: c.GetRemoteID(t, "Test"), Key: c.GetVaultKey(t, "Test")}}
}

func (s *setup) run(t *testing.T, files ...transfer.File) []transfer.Outcome {
	t.Helper()
	outcomes, err := commit.NewCommitRevisions(s.c.CommitDeps()).Execute(t.Context(), ports.CommitInput{Vault: s.vault, Files: files})
	if err != nil {
		t.Fatal(err)
	}
	return outcomes
}

func (s *setup) read(t *testing.T) []*syncv1.StoredRevision {
	t.Helper()
	return s.c.ReadChanges(t, s.vault.ID)
}

func TestCommitRevisions_CommitsANewNoteAsASignedRevision(t *testing.T) {
	s := newSetup(t)

	outcomes := s.run(t, transfer.File{Path: "Shopping list.md", Content: []byte("milk"), ModifiedAt: 1_700_000_005_000_000_000})

	if len(outcomes) != 1 || outcomes[0].Status != transfer.Committed || len(outcomes[0].FileID) != 16 {
		t.Fatalf("outcomes = %+v, want one committed with a new file id of 16 bytes", outcomes)
	}
	stored := s.read(t)
	if len(stored) != 1 {
		t.Fatalf("server holds %d revisions, want 1", len(stored))
	}
	revision := stored[0].GetRevision()
	if !bytes.Equal(revision.GetVaultId(), s.vault.ID) || !bytes.Equal(revision.GetFileId(), outcomes[0].FileID) || len(revision.GetParentRev()) != 0 {
		t.Errorf("revision = vault %x, file %x, parent %x; want this vault, the new file and no parent",
			revision.GetVaultId(), revision.GetFileId(), revision.GetParentRev())
	}
	if revision.GetDeleted() || revision.GetEpoch() != 0 || len(revision.GetShareRootId()) != 0 {
		t.Errorf("revision deleted %v, epoch %d, share root %x; want a live, epoch 0, unshared revision",
			revision.GetDeleted(), revision.GetEpoch(), revision.GetShareRootId())
	}
	id := syncprotocol.GetRevisionID(revision)
	if !bytes.Equal(outcomes[0].RevID, id[:]) || !bytes.Equal(stored[0].GetRevId(), id[:]) {
		t.Error("the id of the outcome is not the id of the revision")
	}
	if !identity.VerifyForPlugin(revision.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID, syncprotocol.GetRevisionMessage(revision), [64]byte(revision.GetSignature())) {
		t.Error("the signature of the revision does not verify under the plugin id")
	}
	if !bytes.Equal(revision.GetPathHash(), note.HashPath(s.vault.Key, "Shopping list.md")) {
		t.Error("the path hash is not the keyed hash of the path")
	}
}

func TestCommitRevisions_SealsThePathAndTheHashInTheMetadata(t *testing.T) {
	s := newSetup(t)

	s.run(t, transfer.File{Path: "Food/Shopping list.md", Content: []byte("milk"), ModifiedAt: 1_700_000_005_000_000_000})

	revision := s.read(t)[0].GetRevision()
	meta := synctest.OpenMeta(t, s.vault.Key, revision)
	if meta.Path != "Food/Shopping list.md" || meta.Size != 4 || meta.MTime != 1_700_000_005_000_000_000 || meta.Hash != note.Hash([]byte("milk")) ||
		meta.Kind != "markdown" || len(meta.Key) != 32 {
		t.Errorf("meta = %+v, want the path, the plain size 4, the time, the hash, the kind markdown and a key of 32 bytes", meta)
	}
	if got := s.c.OpenBlob(t, s.vault.Key, revision); string(got) != "milk" {
		t.Errorf("blob opens to %q, want milk", got)
	}
}

func TestCommitRevisions_NormalizesThePathToNFC(t *testing.T) {
	s := newSetup(t)

	s.run(t, transfer.File{Path: "Café.md", Content: []byte("tea")})

	revision := s.read(t)[0].GetRevision()
	if meta := synctest.OpenMeta(t, s.vault.Key, revision); meta.Path != "Café.md" {
		t.Errorf("meta path = %q, want the composed form", meta.Path)
	}
	if !bytes.Equal(revision.GetPathHash(), note.HashPath(s.vault.Key, "Café.md")) {
		t.Error("the path hash is not the hash of the composed path")
	}
}

func TestCommitRevisions_UploadsNoPlainText(t *testing.T) {
	s := newSetup(t)

	s.run(t, transfer.File{Path: "Shopping list.md", Content: []byte("milk and bread and eggs")})

	revision := s.read(t)[0].GetRevision()
	wire, err := wiretest.EncodeMessage(revision)
	if err != nil {
		t.Fatal(err)
	}
	stored := s.c.ReadStoredBlobs(t)
	for _, plain := range []string{"milk", "Shopping list", note.Hash([]byte("milk and bread and eggs"))} {
		if bytes.Contains(wire, []byte(plain)) || bytes.Contains(stored, []byte(plain)) {
			t.Errorf("the server holds %q as plain text", plain)
		}
	}
	if revision.GetBlobSize() != 56+4096+16 {
		t.Errorf("blob size = %d, want the padded size 4168 and not the size of the note", revision.GetBlobSize())
	}
}

func TestCommitRevisions_UsesANewKeyForEachRevision(t *testing.T) {
	s := newSetup(t)

	s.run(t, transfer.File{Path: "a.md", Content: []byte("same")}, transfer.File{Path: "b.md", Content: []byte("same")})

	keys := map[string]bool{}
	for _, one := range s.read(t) {
		keys[hex.EncodeToString(synctest.OpenMeta(t, s.vault.Key, one.GetRevision()).Key)] = true
	}
	if len(keys) != 2 {
		t.Errorf("%d distinct revision keys for two revisions, want 2", len(keys))
	}
}

func TestCommitRevisions_CommitsAChangedNoteAsAChildOfItsHead(t *testing.T) {
	s := newSetup(t)
	first := s.run(t, transfer.File{Path: "Shopping list.md", Content: []byte("milk")})[0]

	second := s.run(t, transfer.File{FileID: first.FileID, ParentRev: first.RevID, ParentVersion: 1, Path: "Shopping list.md", Content: []byte("bread")})[0]

	if second.Status != transfer.Committed || !bytes.Equal(second.FileID, first.FileID) {
		t.Fatalf("outcome = %+v, want committed under the same file", second)
	}
	stored := s.read(t)
	if len(stored) != 1 || !bytes.Equal(stored[0].GetRevision().GetParentRev(), first.RevID) || !bytes.Equal(stored[0].GetRevId(), second.RevID) {
		t.Errorf("head = %v, want the second revision with the first as its parent", stored)
	}
	if got := s.c.OpenBlob(t, s.vault.Key, stored[0].GetRevision()); string(got) != "bread" {
		t.Errorf("head blob opens to %q, want bread", got)
	}
}

func TestCommitRevisions_ReportsAParentThatIsNotTheHead(t *testing.T) {
	s := newSetup(t)
	first := s.run(t, transfer.File{Path: "Shopping list.md", Content: []byte("milk")})[0]
	wrong := make([]byte, 32)
	if _, err := rand.Read(wrong); err != nil {
		t.Fatal(err)
	}

	outcome := s.run(t, transfer.File{FileID: first.FileID, ParentRev: wrong, Path: "Shopping list.md", Content: []byte("bread")})[0]

	if outcome.Status != transfer.ParentMismatch || outcome.Head == nil || !bytes.Equal(outcome.Head.GetRevId(), first.RevID) {
		t.Errorf("outcome = %+v, want ParentMismatch with the first revision as head", outcome)
	}
	if stored := s.read(t); len(stored) != 1 || !bytes.Equal(stored[0].GetRevId(), first.RevID) {
		t.Error("the head on the server changed after a revision of another parent")
	}
}

func TestCommitRevisions_ReportsAPathThatAnotherFileUses(t *testing.T) {
	s := newSetup(t)
	first := s.run(t, transfer.File{Path: "Plan.md", Content: []byte("tea")})[0]

	outcome := s.run(t, transfer.File{Path: "Plan.md", Content: []byte("coffee")})[0]

	if outcome.Status != transfer.PathTaken || !bytes.Equal(outcome.ExistingFileID, first.FileID) || outcome.Head == nil || !bytes.Equal(outcome.Head.GetRevId(), first.RevID) {
		t.Errorf("outcome = %+v, want PathTaken by the first file, with its revision as head", outcome)
	}
}

func TestCommitRevisions_ReportsAPathOfAnotherFileThatDiffersInCaseOnlyAsACaseClash(t *testing.T) {
	s := newSetup(t)
	s.run(t, transfer.File{Path: "Plan.md", Content: []byte("tea")})

	outcome := s.run(t, transfer.File{Path: "plan.md", Content: []byte("coffee")})[0]

	if outcome.Status != transfer.CaseClash {
		t.Errorf("outcome = %+v, want CaseClash", outcome)
	}
}

func TestCommitRevisions_CommitsManyNotesInBatchesOfOneHundred(t *testing.T) {
	s := newSetup(t)
	files := make([]transfer.File, 205)
	for i := range files {
		files[i] = transfer.File{Path: fmt.Sprintf("n%03d.md", i), Content: []byte(fmt.Sprintf("note %d", i))}
	}

	outcomes := s.run(t, files...)

	for i, outcome := range outcomes {
		if outcome.Status != transfer.Committed || outcome.Path != files[i].Path {
			t.Fatalf("outcome %d = %+v, want committed for %s, in the order of the input", i, outcome, files[i].Path)
		}
	}
	if len(outcomes) != 205 || len(s.read(t)) != 205 {
		t.Errorf("%d outcomes and %d revisions on the server, want 205 of each", len(outcomes), len(s.read(t)))
	}
	if reserved := s.server.CountCalls(syncv1connect.BlobServiceReserveBlobsProcedure); reserved != 3 {
		t.Errorf("reservations = %d, want 3 for 205 files", reserved)
	}
}

func TestCommitRevisions_SkipsANoteTooLargeForTheServer(t *testing.T) {
	s := newSetup(t)
	large := make([]byte, syncprotocol.MaxBlobSize)
	if _, err := rand.Read(large); err != nil {
		t.Fatal(err)
	}

	outcomes := s.run(t, transfer.File{Path: "large.md", Content: large}, transfer.File{Path: "small.md", Content: []byte("milk")})

	if len(outcomes) != 2 || outcomes[0].Status != transfer.TooLarge || outcomes[1].Status != transfer.Committed {
		t.Fatalf("outcomes = %+v, want the large note skipped and the small note committed", outcomes)
	}
	if stored := s.read(t); len(stored) != 1 {
		t.Errorf("server holds %d revisions, want only the small note", len(stored))
	}
}

func TestCommitRevisions_ReturnsTheQuotaTheServerRefusesOn(t *testing.T) {
	s := newSetupOnPlan(t, "prod_tiny")

	_, err := commit.NewCommitRevisions(s.c.CommitDeps()).Execute(t.Context(),
		ports.CommitInput{Vault: s.vault, Files: []transfer.File{{Path: "Shopping list.md", Content: []byte(synctest.NewRandomText(t, 2<<20))}}})

	var quota syncprotocol.QuotaExceededError
	if !errors.As(err, &quota) || quota.Limit != 1<<20 {
		t.Errorf("error = %v, want QuotaExceededError with the limit of 1 MiB", err)
	}
	if stored := s.read(t); len(stored) != 0 {
		t.Errorf("server holds %d revisions, want none", len(stored))
	}
}

func TestCommitRevisions_ReturnsUnreachableWhenTheServerIsDown(t *testing.T) {
	s := newSetup(t)
	s.server.SetDown(true)

	_, err := commit.NewCommitRevisions(s.c.CommitDeps()).Execute(t.Context(),
		ports.CommitInput{Vault: s.vault, Files: []transfer.File{{Path: "Shopping list.md", Content: []byte("milk")}}})

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
}

func TestCommitRevisions_KeepsTheTextOfTheNoteOutOfTheError(t *testing.T) {
	s := newSetup(t)
	s.server.SetDown(true)

	_, err := commit.NewCommitRevisions(s.c.CommitDeps()).Execute(t.Context(),
		ports.CommitInput{Vault: s.vault, Files: []transfer.File{{Path: "Shopping list.md", Content: []byte("milk")}}})

	if err == nil || strings.Contains(err.Error(), "milk") || strings.Contains(err.Error(), hex.EncodeToString(s.vault.Key[:])) {
		t.Errorf("error = %v, want an error with neither the text nor the vault key", err)
	}
}

type nameless struct {
	ports.Revisions
	calls int
}

func (n *nameless) CommitRevisions(context.Context, []byte, []*syncv1.Revision) (*syncv1.CommitRevisionsResponse, error) {
	n.calls++
	return &syncv1.CommitRevisionsResponse{Conflicts: []*syncv1.Conflict{{FileId: []byte("not a file of the batch")}}}, nil
}

func TestCommitRevisions_StopsWhenTheConflictsNameNoFileOfTheBatch(t *testing.T) {
	s := newSetup(t)
	deps := s.c.CommitDeps()
	server := &nameless{Revisions: deps.Revisions}
	deps.Revisions = server

	_, err := commit.NewCommitRevisions(deps).Execute(t.Context(), ports.CommitInput{Vault: s.vault, Files: []transfer.File{{Path: "Shopping list.md", Content: []byte("milk")}}})

	if err == nil || server.calls != 1 {
		t.Errorf("error = %v after %d commits, want an error after one", err, server.calls)
	}
}

type shortAnswer struct{ ports.Revisions }

func (s shortAnswer) CommitRevisions(ctx context.Context, vaultID []byte, revisions []*syncv1.Revision) (*syncv1.CommitRevisionsResponse, error) {
	response, err := s.Revisions.CommitRevisions(ctx, vaultID, revisions)
	if err != nil {
		return nil, err
	}
	response.Revisions = nil
	return response, nil
}

func TestCommitRevisions_RefusesACommitThatAnswersFewerRevisionsThanItems(t *testing.T) {
	s := newSetup(t)
	deps := s.c.CommitDeps()
	deps.Revisions = shortAnswer{Revisions: deps.Revisions}

	_, err := commit.NewCommitRevisions(deps).Execute(t.Context(), ports.CommitInput{Vault: s.vault, Files: []transfer.File{{Path: "A.md", Content: []byte("a"), ModifiedAt: 1_700_000_005_000_000_000}}})

	if err == nil {
		t.Error("error = nil, want an error for a commit with no stored revision for its item")
	}
}

func TestCommitRemovals_RefusesACommitThatAnswersFewerRevisionsThanItems(t *testing.T) {
	s := newSetup(t)
	first := s.run(t, transfer.File{Path: "A.md", Content: []byte("a"), ModifiedAt: 1_700_000_005_000_000_000})[0]
	deps := s.c.CommitDeps()
	deps.Revisions = shortAnswer{Revisions: deps.Revisions}

	_, err := commit.NewCommitRemovals(deps).Execute(t.Context(), transfer.RemoveInput{Vault: s.vault, Records: []transfer.FileRecord{first.Record}})

	if err == nil {
		t.Error("error = nil, want an error for a commit with no stored revision for its item")
	}
}
