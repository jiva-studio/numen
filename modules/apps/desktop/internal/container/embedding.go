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
// Two ways of waiting, because two callers want different things. A pass that
// fills the index has nowhere to be and waits; a person who has typed a
// question is answered by the words rather than kept waiting for the weights.
type Embedding struct {
	is port.EmbeddingModel

	// here is closed once there is something to answer with, or a reason there
	// never will be.
	here chan struct{}
	once sync.Once

	mu   sync.RWMutex
	held port.Embedder
	why  error
}

// Arriving is an embedder being loaded somewhere else, under the identity the
// settings give it.
func Arriving(is port.EmbeddingModel) *Embedding {
	return &Embedding{is: is, here: make(chan struct{})}
}

// Landed is the model turning up, or the reason it never will. It is the first
// of these that counts.
func (e *Embedding) Landed(held port.Embedder, why error) {
	e.once.Do(func() {
		e.mu.Lock()
		e.held, e.why = held, why
		e.mu.Unlock()
		close(e.here)
	})
}

// Disown is the model turning out not to be the one whose vectors are stored.
// It is here and it is not used.
func (e *Embedding) Disown(why error) {
	e.Landed(nil, why)
	e.mu.Lock()
	e.held, e.why = nil, why
	e.mu.Unlock()
}

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

// Close lets go of the model. A model that has not turned up holds nothing, and
// the download behind it ends with the process.
func (e *Embedding) Close() error {
	select {
	case <-e.here:
	default:
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if closer, ok := e.held.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// Filling waits for the model. Nothing is owed to anybody watching a pass over
// a vault, and a pass that gave up would leave the vault unsearchable by
// meaning until something woke it again.
func (e *Embedding) Filling() port.Embedder { return waiting{e} }

// Asking does not wait. A search short of the half that asks by meaning is a
// search the words answer, and a person gets it now.
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
