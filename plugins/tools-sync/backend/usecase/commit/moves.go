package commit

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

// MoveInput is the moves to commit to a vault, with the shares of the vault, whose folders seal the metadata of the files moved into or inside them.
type MoveInput struct {
	Vault  transfer.Vault
	Moves  []transfer.Move
	Shares []share.Root
}

// CommitMoves commits files that have another path as revisions that change only their metadata: they name the
// blob of the head, so nothing is read or uploaded.
type CommitMoves struct{ deps Deps }

// NewCommitMoves returns the use case over the dependencies.
func NewCommitMoves(deps Deps) *CommitMoves { return &CommitMoves{deps: deps} }

// Execute commits the moves in batches of note.MovedBatchFiles, each batch in one call, and returns one outcome for
// each move, in order.
func (u *CommitMoves) Execute(ctx context.Context, in MoveInput) ([]transfer.Outcome, error) {
	author, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	outcomes := make([]transfer.Outcome, 0, len(in.Moves))
	for start := 0; start < len(in.Moves); start += note.MovedBatchFiles {
		batch := in.Moves[start:min(start+note.MovedBatchFiles, len(in.Moves))]
		revisions := make([]*syncv1.Revision, len(batch))
		for i, move := range batch {
			if revisions[i], err = u.build(ctx, in, author.DeviceSignKey, move); err != nil {
				return nil, err
			}
		}
		done, err := commitBatch(ctx, u.deps, in.Vault, revisions)
		if err != nil {
			return nil, err
		}
		for i, outcome := range done {
			outcome.Path = batch[i].To
			if outcome.Status == transfer.Committed {
				outcome.Record = batch[i].Record
				outcome.Record.Path, outcome.Record.HeadRev, outcome.Record.Version = batch[i].To, outcome.RevID, batch[i].Record.Version+1
			}
			outcomes = append(outcomes, outcome)
		}
	}
	return outcomes, nil
}

func (u *CommitMoves) build(ctx context.Context, in MoveInput, author []byte, move transfer.Move) (*syncv1.Revision, error) {
	vault := in.Vault
	record := move.Record
	key, err := u.deps.Crypto.OpenData(ctx, note.GetRecordKeyContext(vault.ID, record.FileID), record.SealedKey)
	if err != nil {
		return nil, fmt.Errorf("open the key of %s: %w", record.Path, err)
	}
	meta := note.Meta{Path: move.To, Size: record.Size, MTime: record.ModifiedAt, Hash: record.BaseHash, Key: key, Kind: record.Kind}
	run := &batchRun{deps: u.deps, vault: vault, author: author, shares: in.Shares}
	sealed, err := run.sealMeta(move.To, record.FileID, meta)
	if err != nil {
		return nil, err
	}
	revision := &syncv1.Revision{
		VaultId: vault.ID, FileId: record.FileID, ParentRev: record.HeadRev, BlobId: record.BlobID, BlobHash: record.BlobHash, BlobSize: record.BlobSize,
		EncMeta: sealed.encoded, PathHash: note.HashPath(vault.Key, move.To), ShareRootId: sealed.shareID, AuthorDeviceSignKey: author, Epoch: sealed.epoch, FileVersion: record.Version + 1,
	}
	return revision, u.sign(ctx, revision)
}

func (u *CommitMoves) sign(ctx context.Context, revision *syncv1.Revision) error {
	signature, err := u.deps.Identity.SignAsDevice(ctx, syncprotocol.GetRevisionMessage(revision))
	revision.Signature = signature
	return err
}

// CommitRemovals commits the removal of files as tombstones: no blob, no metadata.
type CommitRemovals struct{ deps Deps }

// NewCommitRemovals returns the use case over the dependencies.
func NewCommitRemovals(deps Deps) *CommitRemovals { return &CommitRemovals{deps: deps} }

// Execute commits the removals in batches of syncprotocol.MaxBatchRevisions and returns one outcome for each.
func (u *CommitRemovals) Execute(ctx context.Context, in transfer.RemoveInput) ([]transfer.Outcome, error) {
	author, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	outcomes := make([]transfer.Outcome, 0, len(in.Records))
	for start := 0; start < len(in.Records); start += syncprotocol.MaxBatchRevisions {
		batch := in.Records[start:min(start+syncprotocol.MaxBatchRevisions, len(in.Records))]
		revisions := make([]*syncv1.Revision, len(batch))
		for i, record := range batch {
			revisions[i] = &syncv1.Revision{
				VaultId: in.Vault.ID, FileId: record.FileID, ParentRev: record.HeadRev, Deleted: true, FileVersion: record.Version + 1,
				PathHash: note.HashPath(in.Vault.Key, record.Path), AuthorDeviceSignKey: author.DeviceSignKey,
			}
			signature, err := u.deps.Identity.SignAsDevice(ctx, syncprotocol.GetRevisionMessage(revisions[i]))
			if err != nil {
				return nil, err
			}
			revisions[i].Signature = signature
		}
		done, err := commitBatch(ctx, u.deps, in.Vault, revisions)
		if err != nil {
			return nil, err
		}
		for i, outcome := range done {
			outcome.Path, outcome.Record = batch[i].Path, batch[i]
			outcomes = append(outcomes, outcome)
		}
	}
	return outcomes, nil
}

func commitBatch(ctx context.Context, deps Deps, vault transfer.Vault, revisions []*syncv1.Revision) ([]transfer.Outcome, error) {
	response, err := deps.Revisions.CommitRevisions(ctx, vault.ID, revisions)
	if err != nil {
		return nil, err
	}
	if response.GetCommitted() && len(response.GetRevisions()) != len(revisions) {
		return nil, fmt.Errorf("%w: %d stored revisions for %d items", errUnexpectedAnswer, len(response.GetRevisions()), len(revisions))
	}
	outcomes := make([]transfer.Outcome, len(revisions))
	for i, revision := range revisions {
		outcomes[i] = transfer.Outcome{
			Path: "", Status: transfer.Committed, FileID: revision.GetFileId(), RevID: nil, Head: nil, ExistingFileID: nil, Record: transfer.NewEmptyRecord(),
		}
		if response.GetCommitted() {
			outcomes[i].RevID = response.GetRevisions()[i].GetRevId()
			continue
		}
		outcomes[i].Status = transfer.ParentMismatch
		if conflict := findConflict(response.GetConflicts(), revision.GetFileId()); conflict != nil {
			outcomes[i].Head, outcomes[i].ExistingFileID = conflict.GetHead(), conflict.GetExistingFileId()
			if conflict.GetKind() == syncv1.ConflictKind_CONFLICT_KIND_PATH_TAKEN {
				outcomes[i].Status = transfer.PathTaken
			}
		}
	}
	return outcomes, nil
}
