package connection

import (
	"context"
	"log/slog"
	"sync/atomic"
)

// Releaser tells the host that the process of the plugin has finished its work.
type Releaser interface {
	Release(ctx context.Context) error
}

type lookupKey struct{}

// lookup records whether the operation of one call answered a lookup: connections.detected or an option
// list.
type lookup struct {
	isAnswered bool
}

func withLookup(ctx context.Context) (context.Context, *lookup) {
	answered := &lookup{}
	return context.WithValue(ctx, lookupKey{}, answered), answered
}

func markLookup(ctx context.Context) {
	if answered, ok := ctx.Value(lookupKey{}).(*lookup); ok {
		answered.isAnswered = true
	}
}

// openCalls counts the calls of a provider that have not answered, and releases the process after a
// lookup that no other call overlaps.
type openCalls struct {
	releaser Releaser
	count    atomic.Int64
}

func (o *openCalls) begin() {
	o.count.Add(1)
}

func (o *openCalls) end(ctx context.Context, answered *lookup) {
	isLast := o.count.Add(-1) == 0
	if !isLast || !answered.isAnswered || o.releaser == nil {
		return
	}
	if err := o.releaser.Release(context.WithoutCancel(ctx)); err != nil {
		slog.Debug("connection: release the process", "error", err)
	}
}
