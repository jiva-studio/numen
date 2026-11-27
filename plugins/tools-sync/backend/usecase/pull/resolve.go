package pull

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

func (r *run) resolve(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, local string, stat domain.FileStat) error {
	write, err := r.prepare(ctx, stored, meta)
	if err != nil {
		return r.settleFailure(ctx, stored, meta, err)
	}
	outcome, err := r.deps.Conflicts.Resolve(ctx, ports.ConflictInput{
		Vault: r.in.VaultID, Path: meta.Path, Local: ports.FileSide{Hash: local, Stat: stat},
		Head: ports.HeadSide{
			Hash: meta.Hash, Size: meta.Size,
			Replace: func(ctx context.Context, baseline domain.FileStat) error { return write(ctx, &baseline) },
		},
		Day: r.deps.Clock.Now(),
	})
	if err != nil {
		return r.holdUnresolved(ctx, stored, meta, err)
	}
	previous, isKnown := r.getRecord(stored.GetRevision().GetFileId())
	record, err := r.makeRecord(ctx, stored, meta, previous, isKnown)
	if err != nil {
		return err
	}
	if err := r.setContentRecord(ctx, record, previous, isKnown); err != nil {
		return errors.Join(err, r.deps.Downloads.Forget(ctx, r.in.VaultID, stored.GetRevision().GetFileId()))
	}
	if err := r.deps.State.WriteRecords(ctx, r.in.VaultID, r.records); err != nil {
		return err
	}
	r.result.Applied = append(r.result.Applied, outcome.Applied...)
	if outcome.Copy != "" {
		r.result.Resolved = append(r.result.Resolved, meta.Path)
	}
	return r.deps.Downloads.Forget(ctx, r.in.VaultID, stored.GetRevision().GetFileId())
}

func (r *run) holdUnresolved(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, err error) error {
	if errors.Is(err, host.ErrHashMismatch) || errors.Is(err, transfer.ErrIntegrity) {
		return r.settleFailure(ctx, stored, meta, err)
	}
	r.addConflict(meta.Path, stored.GetSeq())
	return r.deps.Downloads.Discard(ctx, r.in.VaultID, stored.GetRevision().GetFileId())
}

func (r *run) settleRace(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta) error {
	if err := r.deps.Downloads.Discard(ctx, r.in.VaultID, stored.GetRevision().GetFileId()); err != nil {
		return err
	}
	local, stat, _, err := r.readLocal(ctx, meta.Path)
	if err != nil {
		return err
	}
	switch {
	case local == "":
		r.addConflict(meta.Path, stored.GetSeq())
		return nil
	case local == meta.Hash:
		previous, isKnown := r.getRecord(stored.GetRevision().GetFileId())
		return r.adopt(ctx, stored, meta, previous, isKnown)
	}
	return r.resolve(ctx, stored, meta, local, stat)
}
