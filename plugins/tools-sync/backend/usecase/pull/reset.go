package pull

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

func (r *run) reconcile(ctx context.Context) error {
	present := map[string]bool{}
	r.isListing = true
	for pages, since := 1, uint64(0); ; pages++ {
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
		if err := syncprotocol.CheckPage(pages, since, page.GetNextSeq(), page.GetHasMore()); err != nil {
			return err
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
			r.link.MergeRetired([]vaultlink.RetiredFile{{FileID: record.FileID, Version: record.Version, At: 0}})
			r.removeRecord(record.FileID)
		}
	}
}
