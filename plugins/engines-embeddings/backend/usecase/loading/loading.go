// Package loading loads a model once, in the background, and keeps it until a change of settings
// retires it.
package loading

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

// Start is what one load of the model needs: the description of the model the opener reads and the
// niceness the process takes once the model is loaded.
type Start[S any] struct {
	Spec S
	Nice int
}

// Ports is what the loader asks of the machine. Opened runs once for each loaded model, with the
// lock held.
type Ports[S any] struct {
	Open      func(ctx context.Context, spec S) (ports.Embedder, error)
	LowerNice func(nice int) error
	Report    func(what string, err error)
	Opened    func(emb ports.Embedder)
}

// Loader holds the loaded model and the load in progress. It shares the lock of its owner: the owner
// holds the lock around ResetLocked, IsLoaded and IsOpening, and the loader takes it in the other
// calls.
type Loader[S any] struct {
	lock     sync.Locker
	ports    Ports[S]
	embedder ports.Embedder
	opening  *opening
}

// opening is one load of the model, which runs on its own goroutine so that a download holds no lock
// a call of the host takes.
type opening struct {
	ready  chan struct{}
	cancel context.CancelFunc
	emb    ports.Embedder
	err    error
}

// ErrClosed is the error of a batch that meets a closed embedder.
var ErrClosed = errors.New("the embedder is closed")

// errSuperseded ends an opening whose model the settings changed while it ran.
var errSuperseded = errors.New("the model was changed while it opened")

func New[S any](lock sync.Locker, ports Ports[S]) *Loader[S] {
	return &Loader[S]{lock: lock, ports: ports, embedder: nil, opening: nil}
}

// Get returns the loaded model. A call that finds none starts the load with start, which runs with
// the lock held, and waits for it; a call that finds a load in progress waits for that one. A load
// outlives the call that started it.
func (l *Loader[S]) Get(ctx context.Context, start func() Start[S]) (ports.Embedder, error) {
	for {
		emb, err := l.wait(ctx, start)
		if !errors.Is(err, errSuperseded) {
			return emb, err
		}
	}
}

// StartLocked starts the load with start when no model is loaded and no load runs, and returns at once.
// A later Get waits for that load. The caller holds the lock.
func (l *Loader[S]) StartLocked(start func() Start[S]) {
	if l.embedder != nil || l.opening != nil {
		return
	}
	l.begin(context.Background(), start())
}

func (l *Loader[S]) wait(ctx context.Context, start func() Start[S]) (ports.Embedder, error) {
	l.lock.Lock()
	if l.embedder != nil {
		emb := l.embedder
		l.lock.Unlock()
		return emb, nil
	}
	op := l.opening
	if op == nil {
		op = l.begin(ctx, start())
	}
	l.lock.Unlock()

	select {
	case <-op.ready:
		return op.emb, op.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *Loader[S]) begin(caller context.Context, start Start[S]) *opening {
	// The load outlives the call that started it, and ends with the settings or the plugin.
	ctx, cancel := context.WithCancel(context.WithoutCancel(caller))
	op := &opening{ready: make(chan struct{}), cancel: cancel, emb: nil, err: nil}
	l.opening = op
	go func() {
		defer cancel()
		emb, err := l.ports.Open(ctx, start.Spec)
		l.finish(op, emb, err, start.Nice)
	}()
	return op
}

func (l *Loader[S]) finish(op *opening, emb ports.Embedder, err error, nice int) {
	l.lock.Lock()
	defer l.lock.Unlock()
	defer close(op.ready)

	if l.opening != op {
		op.err = errSuperseded
		if err == nil {
			l.closeInBackground(emb)
		}
		return
	}
	l.opening = nil
	if err != nil {
		op.err = fmt.Errorf("opening onnx embedding engine: %w", err)
		return
	}
	// The runtime starts its threads while it opens the model, and a thread starts at the niceness of
	// the thread that starts it.
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr log on priority error has no side effect"
	if err := l.ports.LowerNice(nice); err != nil {
		l.ports.Report("set the priority of the embeddings plugin", err)
	}
	l.embedder = emb
	l.ports.Opened(emb)
	op.emb = emb
}

// ResetLocked ends the load in progress and closes the loaded model in the background, so that the
// next Get loads a model again.
func (l *Loader[S]) ResetLocked() {
	l.retireOpening()
	if l.embedder != nil {
		l.closeInBackground(l.embedder)
		l.embedder = nil
	}
}

// IsLoaded returns whether a model is loaded.
func (l *Loader[S]) IsLoaded() bool {
	return l.embedder != nil
}

// IsOpening returns whether a load is in progress.
func (l *Loader[S]) IsOpening() bool {
	return l.opening != nil
}

// Close ends the load in progress and closes the loaded model, and returns the error of the close.
func (l *Loader[S]) Close() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.retireOpening()
	if l.embedder != nil {
		err := l.embedder.Close()
		l.embedder = nil
		return err
	}
	return nil
}

func (l *Loader[S]) retireOpening() {
	if l.opening != nil {
		l.opening.cancel()
		l.opening = nil
	}
}

func (l *Loader[S]) closeInBackground(emb ports.Embedder) {
	go func() {
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr log on background close error has no side effect"
		if err := emb.Close(); err != nil {
			l.ports.Report("close the embedder", err)
		}
	}()
}
