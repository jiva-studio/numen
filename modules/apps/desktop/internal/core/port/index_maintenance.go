package port

import "context"

// IndexMaintenance is the upkeep an index needs that only a scan is in a
// position to ask for. A cache holds an opinion about itself, formed from what
// it contained when it last looked; filling it wholesale invalidates that.
//
// This says what changed. What the upkeep consists of is the index's own
// business.
type IndexMaintenance interface {
	Changed(ctx context.Context) error
}
