// Package model opens the transcription model at the first request, under the configuration the
// settings last named, and closes it when the settings name another model or the plugin stops.
package model

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/ports"
)

// Model is an open transcription model.
type Model interface {
	ports.Recognizer
	Close() error
}

// Opener opens the model the configuration names.
type Opener func(ctx context.Context, cfg settings.Config) (Model, error)

// Reporter receives the failures the holder cannot return.
type Reporter interface {
	ReportCloseError(err error)
}

// Holder keeps the open model.
type Holder struct {
	mu       sync.Mutex
	cfg      settings.Config
	open     Opener
	reporter Reporter
	current  *lease
}

// NewHolder returns a holder that opens its model with open for cfg. The failures to close a model
// go to reporter.
func NewHolder(cfg settings.Config, open Opener, reporter Reporter) *Holder {
	return &Holder{mu: sync.Mutex{}, cfg: cfg, open: open, reporter: reporter, current: nil}
}

// GetRecognizer returns the open model for the transcription use case and opens it first when there is none.
func (h *Holder) GetRecognizer(ctx context.Context) (ports.Recognizer, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.current != nil {
		return h.current, nil
	}
	loaded, err := h.open(context.WithoutCancel(ctx), h.cfg)
	if err != nil {
		return nil, err
	}
	h.current = &lease{holder: h, model: loaded, mu: sync.Mutex{}, uses: 0, isRetired: false, isClosed: false}
	return h.current, nil
}

// UpdateSettings applies the plugin settings to the configuration the model opens with and closes
// the open model when they name another one.
func (h *Holder) UpdateSettings(values map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	model, ok := settings.FindModel(values["model"])
	if !ok || model == h.cfg.Model {
		return
	}
	h.cfg.Model = model
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="Stderr logging of close failure does not affect model state."
	if err := h.retireLocked(); err != nil {
		h.reporter.ReportCloseError(err)
	}
}

// Close closes the open model.
func (h *Holder) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.retireLocked()
}

func (h *Holder) retireLocked() error {
	if h.current == nil {
		return nil
	}
	err := h.current.retire()
	h.current = nil
	return err
}
