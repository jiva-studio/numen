package pull

import (
	"context"
	"slices"
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
	r.forgetAbsent(present)
	if err := r.save(ctx); err != nil {
		return err
	}
	return r.settleRemovals(ctx)
}

func (r *run) forgetAbsent(present map[string]bool) {
	for _, record := range slices.Clone(r.records) {
		if !present[string(record.FileID)] {
			r.removeRecord(record.FileID)
		}
	}
}
