package engine

import (
	"context"
	"sync"
	"sync/atomic"
)

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
