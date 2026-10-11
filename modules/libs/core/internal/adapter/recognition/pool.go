package recognition

import (
	"context"
	"sync"
	"sync/atomic"
)

// runBounded calls do once for each index in [0, count), on at most limit
// goroutines at a time (at least one). No call starts after ctx is done, and
// runBounded returns when every started call has returned.
func runBounded(ctx context.Context, count, limit int, do func(index int)) {
	workers := min(max(limit, 1), count)
	var next atomic.Int64
	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			for ctx.Err() == nil {
				index := int(next.Add(1)) - 1
				if index >= count {
					return
				}
				do(index)
			}
		}()
	}
	group.Wait()
}
