package hostendpoint

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var errEndpointShutdown = errors.New("the host endpoint shuts down")

// background is the work the endpoint finishes after it has answered the call that asked for it, such
// as a change of the vault the window shows. It runs under a context the end of that call does not
// cancel, and the endpoint cancels and waits for it when it closes.
type background struct {
	mu       sync.Mutex
	cancels  map[int]context.CancelFunc
	next     int
	isClosed bool
	workers  sync.WaitGroup
	handle   atomic.Pointer[func(error)]
}

// start runs the work, and returns false when the endpoint is closing and runs nothing.
func (b *background) start(ctx context.Context, work func(ctx context.Context)) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.isClosed {
		return false
	}
	workCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	if b.cancels == nil {
		b.cancels = map[int]context.CancelFunc{}
	}
	id := b.next
	b.next++
	b.cancels[id] = cancel
	b.workers.Add(1)
	go func() {
		defer b.workers.Done()
		defer b.finish(id)
		work(workCtx)
	}()
	return true
}

func (b *background) finish(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if cancel, isRunning := b.cancels[id]; isRunning {
		cancel()
		delete(b.cancels, id)
	}
}

// stop cancels the work that runs and waits for it to end. Nothing starts afterwards.
func (b *background) stop() {
	b.mu.Lock()
	b.isClosed = true
	for _, cancel := range b.cancels {
		cancel()
	}
	b.mu.Unlock()
	b.workers.Wait()
}

// report hands a failure of the work to the error handler, where one is set.
func (b *background) report(err error) {
	if handle := b.handle.Load(); handle != nil {
		(*handle)(err)
	}
}

// SetErrorHandler is where a failure of work the endpoint finishes after its answer is reported,
// such as a change of the vault the window shows.
func (e *Endpoint) SetErrorHandler(handle func(error)) {
	e.background.handle.Store(&handle)
}
