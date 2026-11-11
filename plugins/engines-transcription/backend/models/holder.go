// Package models opens the transcription model once, under the configuration the settings last
// named, and keeps it until the plugin stops.
package models

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcribe"
)

// Model is an open transcription model.
type Model interface {
	transcribe.Recognizer
	Close() error
}

// Opener opens the model the configuration names.
type Opener func(ctx context.Context, cfg settings.Config) (Model, error)

// Holder keeps the open model.
type Holder struct {
	mu      sync.Mutex
	cfg     settings.Config
	open    Opener
	current Model
}

// NewHolder returns a holder that opens the model with open for cfg.
func NewHolder(cfg settings.Config, open Opener) *Holder {
	return &Holder{mu: sync.Mutex{}, cfg: cfg, open: open, current: nil}
}

// GetRecognizer returns the open model for the transcription use case and opens it first when there is none.
func (h *Holder) GetRecognizer(ctx context.Context) (transcribe.Recognizer, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.current != nil {
		return h.current, nil
	}
	loaded, err := h.open(context.WithoutCancel(ctx), h.cfg)
	if err != nil {
		return nil, err
	}
	h.current = loaded
	return h.current, nil
}

// UpdateSettings applies the plugin settings to the configuration the model opens with.
func (h *Holder) UpdateSettings(values map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if model, ok := settings.FindModel(values["model"]); ok {
		h.cfg.Model = model
	}
}

// Close closes the open model.
func (h *Holder) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.current == nil {
		return nil
	}
	err := h.current.Close()
	h.current = nil
	return err
}
