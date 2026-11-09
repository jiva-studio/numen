package indexport

import "context"

// IndexMaintenance is the upkeep an index needs that only a scan can request. A cache
// holds a view of itself formed from its contents when it last looked; filling it
// wholesale invalidates that view.
//
// This states what changed. What the upkeep consists of is the index's own concern.
type IndexMaintenance interface {
	ReportChanges(ctx context.Context) error
}
