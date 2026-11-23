package pull

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

// removal is a file another computer removed, waiting for the guard.
type removal struct {
	record transfer.FileRecord
	stat   domain.FileStat
	revID  []byte
}

// applyDeletion queues the removal of an unedited file, keeps an edited one with the tombstone as the head so the
// next push is an edit that beats the removal, and drops the record of a file that is gone.
func (r *run) applyDeletion(ctx context.Context, stored *syncv1.StoredRevision, record transfer.FileRecord, isKnown bool) error {
	if !isKnown {
		return nil
	}
	hash, stat, err := r.readLocal(ctx, record.Path)
	if err != nil {
		return err
	}
	switch {
	case hash == "":
		r.removeRecord(record.FileID)
	case hash != record.BaseHash:
		record.HeadRev = stored.GetRevId()
		r.setRecord(record)
	default:
		r.removals = append(r.removals, removal{record: record, stat: stat, revID: stored.GetRevId()})
		if !r.hasRemovals {
			r.removalSeq, r.hasRemovals = stored.GetSeq(), true
		}
	}
	return nil
}

// settleRemovals asks the guard about the queued removals and carries out its answer.
func (r *run) settleRemovals(ctx context.Context) error {
	if !r.hasRemovals {
		return nil
	}
	candidates := make([]transfer.Removal, len(r.removals))
	for i, one := range r.removals {
		candidates[i] = transfer.Removal{FileID: one.record.FileID, Path: one.record.Path, BaseHash: one.record.BaseHash, RevID: one.revID}
	}
	verdict, err := r.deps.Guard.Execute(ctx, transfer.GuardInput{
		Vault: r.in.VaultID, Direction: transfer.DirectionApply, Candidates: candidates, Tracked: len(r.records),
	})
	if err != nil || verdict == transfer.Pause {
		return err
	}
	for _, one := range r.removals {
		if err := r.settleRemoval(ctx, one, verdict == transfer.Keep); err != nil {
			return err
		}
	}
	r.removals, r.hasRemovals = nil, false
	return r.save(ctx)
}

func (r *run) settleRemoval(ctx context.Context, one removal, isKept bool) error {
	if !isKept {
		_, err := r.moves.RemoveFile(ctx, one.record.Path, &one.stat)
		if err == nil || errors.Is(err, host.ErrNotExist) {
			r.removeRecord(one.record.FileID)
			return nil
		}
		if !errors.Is(err, host.ErrStale) {
			return err
		}
	}
	one.record.HeadRev = one.revID
	if isKept {
		one.record.BaseHash = ""
	}
	r.setRecord(one.record)
	return nil
}
