package port

import "context"

// IndexStatistics is the index's own measurement of itself.
//
// The index offers more than one way to answer most questions, and it chooses
// between them from what it knows about how much is stored and how it is spread.
// Freshly filled, it knows nothing, and picks by rule of thumb — which on a
// large vault means reading far more than it needs to. The measurement is what
// turns the choice from a guess into a decision.
//
// It is a port because only a scan knows when the index has changed enough to
// be worth measuring again.
type IndexStatistics interface {
	Update(ctx context.Context) error
}
