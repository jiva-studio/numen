package embedding

import (
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrArriving is a model that is not on this machine yet.
var ErrArriving = errors.New("the model is still arriving")

// Embedder is an embedder that is still being fetched and loaded.
//
// What it is is known before it is here: the settings say the model, so the
// vector index is fitted and a vector is claimed under the right recipe while
// the weights are still coming down.
//
// Two ways of waiting. A pass that fills the index waits; a person who has
// typed a question is answered by the words.
type Embedder struct {
	model port.EmbeddingModel

	// ready is closed once there is something to answer with, or a reason there
	// never will be.
	ready chan struct{}
	once  sync.Once

	mu     sync.RWMutex
	held   port.Embedder
	reason error
	// isSettled is the wait being over: something answered for it, it was let go
	// of, or it was closed.
	isSettled bool
}

// WaitingEmbedder is an embedder whose model may still be on its way. Fetching
// a model and preparing it are their own step in the list of what is being
// done, and a pass that fills an index waits here until that step is over.
type WaitingEmbedder interface {
	port.Embedder
	// Wait blocks until the model turns up, and says what stopped it.
	Wait(ctx context.Context) error
}

// NewArriving is an embedder being loaded somewhere else, under the identity
// the settings give it.
func NewArriving(is port.EmbeddingModel) *Embedder {
	return &Embedder{model: is, ready: make(chan struct{})}
}

// ReportArrival is the model turning up, or the reason it never will. It is the
// first of these that counts, and a model arriving after the wait is over is
// let go of where it stands.
func (e *Embedder) ReportArrival(held port.Embedder, why error) {
	e.mu.Lock()
	late := e.isSettled
	if !late {
		e.isSettled = true
		e.held, e.reason = held, why
	}
	e.mu.Unlock()

	e.once.Do(func() { close(e.ready) })
	if late {
		_ = letGo(held)
	}
}

// Disown is the model turning out not to be the one whose vectors are stored.
// It is let go of, and nothing is asked of it again.
//
// The reason is in place before anybody can see the wait is over.
func (e *Embedder) Disown(why error) error {
	e.mu.Lock()
	held := e.held
	e.held, e.reason, e.isSettled = nil, why, true
	e.mu.Unlock()

	e.once.Do(func() { close(e.ready) })
	return letGo(held)
}

// letGo closes a model that is here, and says what closing it said. There is
// nothing to close where nothing landed.
func letGo(held port.Embedder) error {
	if held == nil {
		return nil
	}
	return held.Close()
}

// Model is what the settings say this is, known before the weights are here.
func (e *Embedder) Model() port.EmbeddingModel { return e.model }

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
	case e.held == nil:
		return ErrArriving
	}
	return nil
}

// Close lets go of the model. The wait is over from here, so a model still on
// its way down is let go of the moment it arrives.
func (e *Embedder) Close() error {
	e.mu.Lock()
	held := e.held
	e.held, e.isSettled = nil, true
	e.mu.Unlock()
	return letGo(held)
}

// GetWaitingEmbedder waits for the model. Nothing is owed to anybody watching a
// pass over a vault.
func (e *Embedder) GetWaitingEmbedder() port.Embedder { return waitingEmbedder{e} }

// GetImpatientEmbedder does not wait. A search short of the half that asks by
// meaning is a search the words answer.
func (e *Embedder) GetImpatientEmbedder() port.Embedder { return impatient{e} }

// embed is what the model answers with, once it is here.
func (e *Embedder) embed(ctx context.Context, texts []string) ([][]float32, error) {
	e.mu.RLock()
	held := e.held
	e.mu.RUnlock()
	if held == nil {
		return nil, ErrArriving
	}
	return held.Embed(ctx, texts)
}

type waitingEmbedder struct{ e *Embedder }

func (w waitingEmbedder) Model() port.EmbeddingModel { return w.e.model }

// Wait blocks until the model turns up, and says what stopped it.
func (w waitingEmbedder) Wait(ctx context.Context) error { return w.e.Wait(ctx) }

func (w waitingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := w.e.Wait(ctx); err != nil {
		return nil, err
	}
	return w.e.embed(ctx, texts)
}

// Close lets go of the one model both ways of waiting ask.
func (w waitingEmbedder) Close() error { return w.e.Close() }

type impatient struct{ e *Embedder }

func (i impatient) Model() port.EmbeddingModel { return i.e.model }

func (i impatient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	select {
	case <-i.e.ready:
	default:
		return nil, ErrArriving
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
func (i impatient) Close() error { return i.e.Close() }
