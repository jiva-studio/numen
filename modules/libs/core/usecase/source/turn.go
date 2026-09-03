package source

import (
	"context"
	"slices"
	"sync"
)

// heavy is the one heavy run this machine does at a time. Recognising a scan
// and transcribing a recording each hold the models and the processor, and they
// take turns.
var heavy turn

// The queues a run waits in, the first of them served first. Work a person is
// sitting in front of goes before work the vault set itself.
const (
	queueAsked = iota
	queueUnasked
	queues
)

// A turn is handed out to one run at a time. A run waits in the queue its work
// belongs to and is served in the order it arrived there.
type turn struct {
	mu      sync.Mutex
	held    bool
	waiting [queues][]chan struct{}
}

// take waits for this machine's turn at the models and hands back what gives
// the turn up. waiting is called where the turn is not free, so that a person
// watching is told why nothing is moving. A context that ends while waiting
// takes no turn.
func (g *turn) take(ctx context.Context, asked bool, waiting func()) (func(), error) {
	at := queueUnasked
	if asked {
		at = queueAsked
	}

	g.mu.Lock()
	if !g.held {
		g.held = true
		g.mu.Unlock()
		return g.give, nil
	}
	stand := make(chan struct{})
	g.waiting[at] = append(g.waiting[at], stand)
	g.mu.Unlock()

	if waiting != nil {
		waiting()
	}
	select {
	case <-stand:
		return g.give, nil
	case <-ctx.Done():
		g.leave(at, stand)
		return nil, ctx.Err()
	}
}

// give hands the turn to whoever has waited longest in the first queue anybody
// stands in, and lets it go where nobody does.
func (g *turn) give() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for at := range g.waiting {
		if len(g.waiting[at]) > 0 {
			next := g.waiting[at][0]
			g.waiting[at] = g.waiting[at][1:]
			close(next)
			return
		}
	}
	g.held = false
}

// leave takes a run out of its queue. One already handed the turn holds it, and
// gives it on.
func (g *turn) leave(at int, stand chan struct{}) {
	g.mu.Lock()
	for i, one := range g.waiting[at] {
		if one == stand {
			g.waiting[at] = slices.Delete(g.waiting[at], i, i+1)
			g.mu.Unlock()
			return
		}
	}
	g.mu.Unlock()
	<-stand
	g.give()
}
