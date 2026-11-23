package pull

import (
	"bytes"
	"context"
	"slices"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

func (r *run) reconcile(ctx context.Context) error {
	present := map[string]bool{}
	r.isListing = true
	for since := uint64(0); ; {
		page, err := r.deps.Revisions.GetChanges(ctx, r.remote, since, 0)
		if err != nil {
			return err
		}
		for _, stored := range page.GetChanges() {
			present[string(stored.GetRevision().GetFileId())] = true
		}
		if err := r.applyPage(ctx, page.GetChanges()); err != nil {
			return err
		}
		r.next = page.GetNextSeq()
		if err := r.save(ctx); err != nil {
			return err
		}
		if !page.GetHasMore() {
			break
		}
		since = page.GetNextSeq()
	}
	r.isListing = false
	if err := r.settleAbsent(ctx, present); err != nil {
		return err
	}
	if err := r.save(ctx); err != nil {
		return err
	}
	return r.settleRemovals(ctx)
}

func (r *run) settleAbsent(ctx context.Context, present map[string]bool) error {
	oldSeq := r.link.ServerSeq
	for _, record := range slices.Clone(r.records) {
		if present[string(record.FileID)] {
			continue
		}
		hash, stat, err := r.readLocal(ctx, record.Path)
		if err != nil {
			return err
		}
		if hash == "" || hash != record.BaseHash || len(present) == 0 || r.hasOtherRecordAt(record) {
			r.removeRecord(record.FileID)
			continue
		}
		r.queueRemoval(removal{record: record, stat: stat, revID: nil}, oldSeq+1)
	}
	return nil
}

func (r *run) hasOtherRecordAt(record transfer.FileRecord) bool {
	return slices.ContainsFunc(r.records, func(other transfer.FileRecord) bool {
		return other.Path == record.Path && !bytes.Equal(other.FileID, record.FileID)
	})
}
