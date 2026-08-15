package port

import "context"

// IndexMaintenance is the upkeep an index needs that only a scan is in a
// position to ask for. A cache holds an opinion about itself, formed from what
// it contained when it last looked; filling it wholesale invalidates that.
//
// What the upkeep consists of is not the core's business, which is why this
// says what changed rather than what to do about it.
type IndexMaintenance interface {
	Changed(ctx context.Context) error
}
