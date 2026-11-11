// Package models opens the transcription model once, under the configuration the settings last
// named, and keeps it until the plugin stops.
package models

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcribe"
)

// Opener opens the model the configuration names.
type Opener func(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error)

// NewOpener returns an opener of the Parakeet model on runtime.
func NewOpener(runtime engine.Runtime) Opener {
	return func(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error) {
		return engine.Open(ctx, runtime, cfg)
	}
}

// Holder keeps the open model.
type Holder struct {
	mu      sync.Mutex
	cfg     engine.Config
	isMock  bool
	open    Opener
	current engine.TranscriberPort
}

// NewHolder returns a holder that opens the model with open for cfg, or the mock model while isMock holds.
func NewHolder(cfg engine.Config, isMock bool, open Opener) *Holder {
	return &Holder{mu: sync.Mutex{}, cfg: cfg, isMock: isMock, open: open, current: nil}
}

// GetRecognizer returns the open model for the transcription use case and opens it first when there is none.
func (h *Holder) GetRecognizer(ctx context.Context) (transcribe.Recognizer, error) {
	port, err := h.get(ctx)
	if err != nil {
		return nil, err
	}
	return recognizer{port: port}, nil
}

// UpdateSettings applies the plugin settings to the configuration the model opens with.
func (h *Holder) UpdateSettings(values map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if model, ok := engine.FindModel(values["model"]); ok {
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

func (h *Holder) get(ctx context.Context) (engine.TranscriberPort, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.current != nil {
		return h.current, nil
	}
	if h.isMock {
		h.current = engine.NewMockTranscriber()
		return h.current, nil
	}
	loaded, err := h.open(context.WithoutCancel(ctx), h.cfg)
	if err != nil {
		return nil, err
	}
	h.current = loaded
	return h.current, nil
}
