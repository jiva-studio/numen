// Package sessions keeps the run open on each vault.
package sessions

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
)

// ErrNoRun is an answer named against a run this backend did not open. A run is one session at one
// window, and an answer belongs to the session it was given in.
var ErrNoRun = errors.New("no run of that name is open")

// Registry is the run open on each vault, by the vault's identity. One session to a vault: opening
// another releases the one before it, and the file that one wrote is never appended to again.
type Registry struct {
	mu  sync.Mutex
	run map[domain.VaultID]*flashcards.LogWriter
}

// NewRegistry returns a registry in which no run is open.
func NewRegistry() *Registry {
	return &Registry{mu: sync.Mutex{}, run: make(map[domain.VaultID]*flashcards.LogWriter)}
}

// Open opens a new run on v at now and makes it the active run for v.
func (r *Registry) Open(
	ctx context.Context, log flashcards.Log, v domain.Vault, now time.Time,
) (*flashcards.LogWriter, error) {
	run, err := log.Open(ctx, v, now)
	if err != nil {
		return nil, err
	}
	r.Store(v, run)
	return run, nil
}

// Store makes run the one open on the vault v.
func (r *Registry) Store(v domain.Vault, run *flashcards.LogWriter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.run[v.ID] = run
}

// GetRun returns the run named, if it is the one open on the vault.
func (r *Registry) GetRun(vault domain.VaultID, name string) (*flashcards.LogWriter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, is := r.run[vault]
	if !is || run.GetName() != name {
		return nil, fmt.Errorf("%w: %s", ErrNoRun, name)
	}
	return run, nil
}

// AnswerRequest names the run and the answer to append to it.
type AnswerRequest struct {
	Vault    domain.VaultID
	Run      string
	Card     review.CardID
	Rating   review.Rating
	Duration time.Duration
}

// RecordAnswer appends an answer to the active run named on vault.
func (r *Registry) RecordAnswer(
	ctx context.Context, record flashcards.Record, req AnswerRequest,
) (review.Answer, error) {
	run, err := r.GetRun(req.Vault, req.Run)
	if err != nil {
		return review.Answer{}, err
	}
	return record.Answer(ctx, run, req.Card, req.Rating, req.Duration)
}

// UndoAnswer withdraws an answer from the active run named on vault.
func (r *Registry) UndoAnswer(
	ctx context.Context,
	record flashcards.Record,
	vault domain.VaultID,
	runName string,
	answerID string,
) error {
	run, err := r.GetRun(vault, runName)
	if err != nil {
		return err
	}
	_, err = record.TakeBack(ctx, run, answerID)
	return err
}
