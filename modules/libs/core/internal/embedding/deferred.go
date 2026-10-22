package embedding

import (
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/port/embeddingport"
)

// ErrModelNotReady is a model that is not on this machine yet.
var ErrModelNotReady = errors.New("the model is still arriving")

// Embedder is an embedder that is still being fetched and loaded.
//
// What it is is known before it is here: the settings say the model, so the
// vector index is fitted and a vector is claimed under the right recipe while
// the weights are still coming down.
//
// Two ways of waiting. A pass that fills the index waits; a person who has
// typed a question is answered by the words.
type Embedder struct {
	model embeddingport.EmbeddingModel

	// ready is closed once there is something to answer with, or a reason there
	// never will be.
	ready chan struct{}
	once  sync.Once

	mu     sync.RWMutex
	item   embeddingport.Embedder
	reason error
	// isDone is the wait being over: something answered for it, it was let go
	// of, or it was closed.
	isDone bool
}

// WaitingEmbedder is an embedder whose model may still be on its way. Fetching
// a model and preparing it are their own step in the list of what is being
// done, and a pass that fills an index waits here until that step is over.
type WaitingEmbedder interface {
	embeddingport.Embedder
	// Wait blocks until the model turns up, and says what stopped it.
	Wait(ctx context.Context) error
}

// NewDeferredEmbedder is an embedder being loaded somewhere else, under the identity
// the settings give it.
func NewDeferredEmbedder(is embeddingport.EmbeddingModel) *Embedder {
	return &Embedder{model: is, ready: make(chan struct{})}
}

// SetModel is the model turning up, or the reason it never will. It is the
// first of these that counts, and a model arriving after the wait is over is
// release at its current position.
func (e *Embedder) SetModel(embedder embeddingport.Embedder, why error) {
	e.mu.Lock()
	late := e.isDone
	if !late {
		e.isDone = true
		e.item, e.reason = embedder, why
	}
	e.mu.Unlock()

	e.once.Do(func() { close(e.ready) })
	if late {
		_ = closeIfPresent(embedder)
	}
}

// DiscardModel is the model turning out not to be the one whose vectors are stored.
// It is release, and nothing is asked of it again.
//
// The reason is in place before anybody can see the wait is over.
func (e *Embedder) DiscardModel(why error) error {
	e.mu.Lock()
	target := e.item
	e.item, e.reason, e.isDone = nil, why, true
	e.mu.Unlock()

	e.once.Do(func() { close(e.ready) })
	return closeIfPresent(target)
}

// closeIfPresent closes a model that is here, and says what closing it said. There is
// nothing to close where nothing landed.
func closeIfPresent(target embeddingport.Embedder) error {
	if target == nil {
		return nil
	}
	return target.Close()
}

// GetModel is what the settings say this is, known before the weights are here.
func (e *Embedder) GetModel() embeddingport.EmbeddingModel { return e.model }

// Wait blocks until the model turns up, and says what stopped it.
func (e *Embedder) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.ready:
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	switch {
	case e.reason != nil:
		return e.reason
	case e.item == nil:
		return ErrModelNotReady
	}
	return nil
}

// Close lets go of the model. The wait is over from here, so a model still on
// its way down is release the moment it arrives.
func (e *Embedder) Close() error {
	e.mu.Lock()
	target := e.item
	e.item, e.isDone = nil, true
	e.mu.Unlock()
	return closeIfPresent(target)
}

// GetWaitingEmbedder waits for the model. Nothing is owed to anybody watching a
// pass over a vault.
func (e *Embedder) GetWaitingEmbedder() embeddingport.Embedder { return blockingEmbedder{e} }

// GetImpatientEmbedder does not wait. A search short of the half that asks by
// meaning is a search the words answer.
func (e *Embedder) GetImpatientEmbedder() embeddingport.Embedder { return nonBlockingEmbedder{e} }

// embed is what the model answers with, once it is here.
func (e *Embedder) embed(ctx context.Context, texts []string) ([][]float32, error) {
	e.mu.RLock()
	target := e.item
	e.mu.RUnlock()
	if target == nil {
		return nil, ErrModelNotReady
	}
	return target.Embed(ctx, texts)
}

type blockingEmbedder struct{ e *Embedder }

func (w blockingEmbedder) GetModel() embeddingport.EmbeddingModel { return w.e.model }

// Wait blocks until the model turns up, and says what stopped it.
func (w blockingEmbedder) Wait(ctx context.Context) error { return w.e.Wait(ctx) }

func (w blockingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := w.e.Wait(ctx); err != nil {
		return nil, err
	}
	return w.e.embed(ctx, texts)
}

// Close lets go of the one model both ways of waiting ask.
func (w blockingEmbedder) Close() error { return w.e.Close() }

// Concurrency reports how many parallel batches this embedder can execute.
func (w blockingEmbedder) Concurrency() int { return w.e.Concurrency() }

type nonBlockingEmbedder struct{ e *Embedder }

func (i nonBlockingEmbedder) GetModel() embeddingport.EmbeddingModel { return i.e.model }

func (i nonBlockingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	select {
	case <-i.e.ready:
	default:
		return nil, ErrModelNotReady
	}
	i.e.mu.RLock()
	why := i.e.reason
	i.e.mu.RUnlock()
	if why != nil {
		return nil, why
	}
	return i.e.embed(ctx, texts)
}

// Close lets go of the one model both ways of waiting ask.
func (i nonBlockingEmbedder) Close() error { return i.e.Close() }

// Concurrency reports how many parallel batches this embedder can execute.
func (i nonBlockingEmbedder) Concurrency() int { return i.e.Concurrency() }

// Concurrency reports how many parallel batches the model runs, and one while it
// has not arrived.
func (e *Embedder) Concurrency() int {
	e.mu.RLock()
	target := e.item
	e.mu.RUnlock()
	if target == nil {
		return 1
	}
	return target.Concurrency()
}
