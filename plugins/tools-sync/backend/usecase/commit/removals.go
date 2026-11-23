package commit

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

// CommitRemovals commits the removal of notes as tombstones: no blob, no metadata.
type CommitRemovals struct{ deps Deps }

// NewCommitRemovals returns the use case over the dependencies.
func NewCommitRemovals(deps Deps) *CommitRemovals { return &CommitRemovals{deps: deps} }

// Execute commits the removals in batches of one hundred and returns one outcome for each, in the
// order of the records. The parent of a tombstone is the HeadRev of its record.
func (u *CommitRemovals) Execute(ctx context.Context, in transfer.RemoveInput) ([]transfer.Outcome, error) {
	author, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	outcomes := make([]transfer.Outcome, 0, len(in.Records))
	for start := 0; start < len(in.Records); start += batchSize {
		batch := in.Records[start:min(start+batchSize, len(in.Records))]
		revisions := make([]*syncv1.Revision, len(batch))
		for i, record := range batch {
			if revisions[i], err = u.newTombstone(ctx, in.Vault, author.DeviceSignKey, record); err != nil {
				return nil, err
			}
		}
		response, err := u.deps.Revisions.CommitRevisions(ctx, in.Vault.ID, revisions)
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, getRemovalOutcomes(batch, response)...)
	}
	return outcomes, nil
}

func (u *CommitRemovals) newTombstone(ctx context.Context, vault transfer.Vault, author []byte, record note.FileRecord) (*syncv1.Revision, error) {
	revision := &syncv1.Revision{
		VaultId: vault.ID, FileId: record.FileID, ParentRev: record.HeadRev, Deleted: true,
		PathHash: note.PathHash(vault.Key, record.Path), AuthorDeviceSignKey: author,
	}
	signature, err := u.deps.Identity.SignAsDevice(ctx, syncprotocol.GetRevisionMessage(revision))
	if err != nil {
		return nil, fmt.Errorf("sign the removal of %s: %w", record.Path, err)
	}
	revision.Signature = signature
	return revision, nil
}

func getRemovalOutcomes(batch []note.FileRecord, response *syncv1.CommitRevisionsResponse) []transfer.Outcome {
	outcomes := make([]transfer.Outcome, len(batch))
	for i, record := range batch {
		outcomes[i] = transfer.Outcome{Path: record.Path, Status: transfer.Committed, FileID: record.FileID}
		if response.GetCommitted() {
			outcomes[i].RevID = response.GetRevisions()[i].GetRevId()
			continue
		}
		outcomes[i].Status = transfer.ParentMismatch
		if conflict := findConflict(response.GetConflicts(), record.FileID); conflict != nil {
			outcomes[i].Head = conflict.GetHead()
		}
	}
	return outcomes
}
