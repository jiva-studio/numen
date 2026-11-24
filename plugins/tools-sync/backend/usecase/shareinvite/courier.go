package shareinvite

import (
	"context"
	"sync"
)

// Courier runs for as long as the plugin does: it runs the waits of the links and requests the use cases start.
type Courier struct{ hub *Hub }

func NewCourier(hub *Hub) *Courier { return &Courier{hub: hub} }

// Execute runs each task the use cases queue, one goroutine each, until ctx ends and every task has ended.
func (c *Courier) Execute(ctx context.Context) error {
	var running sync.WaitGroup
	defer running.Wait()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case task := <-c.hub.tasks:
			running.Go(func() { task(ctx) })
		}
	}
}
