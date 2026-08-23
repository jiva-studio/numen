package container

import (
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// ErrArriving is a model that is not on this machine yet.
var ErrArriving = errors.New("the model is still arriving")

// Embedding is an embedder that is still being fetched and loaded.
//
// What it is is known before it is here: the settings say the model, so the
// vector index is fitted and a vector is claimed under the right recipe while
// the weights are still coming down.
//
// Two ways of waiting. A pass that fills the index waits; a person who has
// typed a question is answered by the words.
type Embedding struct {
	is port.EmbeddingModel

	// here is closed once there is something to answer with, or a reason there
	// never will be.
	here chan struct{}
	once sync.Once

	mu   sync.RWMutex
	held port.Embedder
	why  error
	// settled is the wait being over: something answered for it, it was let go
	// of, or it was closed.
	settled bool
}

// Arriving is an embedder being loaded somewhere else, under the identity the
// settings give it.
func Arriving(is port.EmbeddingModel) *Embedding {
	return &Embedding{is: is, here: make(chan struct{})}
}

// Landed is the model turning up, or the reason it never will. It is the first
// of these that counts, and a model arriving after the wait is over is let go
// of where it stands.
func (e *Embedding) Landed(held port.Embedder, why error) {
	e.mu.Lock()
	late := e.settled
	if !late {
		e.settled = true
		e.held, e.why = held, why
	}
	e.mu.Unlock()

	e.once.Do(func() { close(e.here) })
	if late {
		_ = letGo(held)
	}
}

// Disown is the model turning out not to be the one whose vectors are stored.
// It is let go of, and nothing is asked of it again.
//
// The reason is in place before anybody can see the wait is over.
func (e *Embedding) Disown(why error) error {
	e.mu.Lock()
	held := e.held
	e.held, e.why, e.settled = nil, why, true
	e.mu.Unlock()

	e.once.Do(func() { close(e.here) })
	return letGo(held)
}

// letGo closes a model that holds something, and says what closing it said.
func letGo(held port.Embedder) error {
	if closer, ok := held.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// Model is what the settings say this is, known before the weights are here.
func (e *Embedding) Model() port.EmbeddingModel { return e.is }

// Here says whether there is something to embed with.
func (e *Embedding) Here() bool {
	select {
	case <-e.here:
	default:
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.held != nil
}

// Wait blocks until the model turns up, and says what stopped it.
func (e *Embedding) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.here:
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	switch {
	case e.why != nil:
		return e.why
	case e.held == nil:
		return ErrArriving
	}
	return nil
}

// Close lets go of the model. The wait is over from here, so a model still on
// its way down is let go of the moment it arrives.
func (e *Embedding) Close() error {
	e.mu.Lock()
	held := e.held
	e.held, e.settled = nil, true
	e.mu.Unlock()
	return letGo(held)
}

// Filling waits for the model. Nothing is owed to anybody watching a pass over
// a vault.
func (e *Embedding) Filling() port.Embedder { return waiting{e} }

// Asking does not wait. A search short of the half that asks by meaning is a
// search the words answer.
func (e *Embedding) Asking() port.Embedder { return impatient{e} }

// embedding is what the model answers with, once it is here.
func (e *Embedding) embedding(ctx context.Context, texts []string) ([][]float32, error) {
	e.mu.RLock()
	held := e.held
	e.mu.RUnlock()
	if held == nil {
		return nil, ErrArriving
	}
	return held.Embed(ctx, texts)
}

type waiting struct{ e *Embedding }

func (w waiting) Model() port.EmbeddingModel { return w.e.is }

func (w waiting) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := w.e.Wait(ctx); err != nil {
		return nil, err
	}
	return w.e.embedding(ctx, texts)
}

type impatient struct{ e *Embedding }

func (i impatient) Model() port.EmbeddingModel { return i.e.is }

func (i impatient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	select {
	case <-i.e.here:
	default:
		return nil, ErrArriving
	}
	i.e.mu.RLock()
	why := i.e.why
	i.e.mu.RUnlock()
	if why != nil {
		return nil, why
	}
	return i.e.embedding(ctx, texts)
}
