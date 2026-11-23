package pull

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// removal is a file another computer removed, waiting for the guard. revID and version are of the signed removal.
type removal struct {
	record  transfer.FileRecord
	stat    domain.FileStat
	revID   []byte
	version uint64
}

func (r *run) applyDeletion(ctx context.Context, stored *syncv1.StoredRevision, record transfer.FileRecord, isKnown bool) error {
	version := stored.GetRevision().GetFileVersion()
	if !isKnown {
		if version > r.link.GetFloor(stored.GetRevision().GetFileId()) {
			r.link.AddRetired(vaultlink.RetiredFile{FileID: stored.GetRevision().GetFileId(), Version: version})
		}
		return nil
	}
	hash, stat, err := r.readLocal(ctx, record.Path)
	if err != nil {
		return err
	}
	switch {
	case hash == "":
		r.removeRecord(record.FileID)
		r.link.AddRetired(vaultlink.RetiredFile{FileID: record.FileID, Version: version})
	case hash != record.BaseHash:
		record.HeadRev, record.Version = stored.GetRevId(), version
		r.setRecord(record)
	default:
		r.queueRemoval(removal{record: record, stat: stat, revID: stored.GetRevId(), version: version}, stored.GetSeq())
	}
	return nil
}

func (r *run) settleRemovals(ctx context.Context) error {
	if len(r.removals) == 0 {
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
	r.removals = nil
	return r.save(ctx)
}

func (r *run) settleRemoval(ctx context.Context, one removal, isKept bool) error {
	if !isKept {
		_, err := r.moves.RemoveFile(ctx, one.record.Path, &one.stat)
		if err == nil || errors.Is(err, host.ErrNotExist) {
			r.removeRecord(one.record.FileID)
			r.link.AddRetired(vaultlink.RetiredFile{FileID: one.record.FileID, Version: one.version})
			return nil
		}
		if !errors.Is(err, host.ErrStale) {
			return err
		}
	}
	one.record.HeadRev, one.record.Version = one.revID, one.version
	if isKept {
		one.record.BaseHash = ""
	}
	r.setRecord(one.record)
	return nil
}

func (r *run) queueRemoval(one removal, seq uint64) {
	if len(r.removals) == 0 || seq < r.removalSeq {
		r.removalSeq = seq
	}
	r.removals = append(r.removals, one)
}
