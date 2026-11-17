// Package engines holds the recognizer of the plugin: it opens it when a page is first read and
// opens it again once a setting that decides what it reads with changes.
package engines

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/ports"
)

// Recognizer is the recognizer of a page and its contents.
type Recognizer interface {
	ports.Recognizer
	Close() error
}

// Opener opens the recognizer the configuration names.
type Opener func(ctx context.Context, cfg settings.Config) (Recognizer, error)

// Reporter receives the failures the holder cannot return.
type Reporter interface {
	ReportCloseError(err error)
}

// Holder keeps one recognizer for the current configuration.
type Holder struct {
	mu       sync.Mutex
	cfg      settings.Config
	open     Opener
	reporter Reporter
	engine   Recognizer
}

// NewHolder returns a holder that opens its recognizer with open for cfg.
func NewHolder(cfg settings.Config, open Opener, reporter Reporter) *Holder {
	return &Holder{mu: sync.Mutex{}, cfg: cfg, open: open, reporter: reporter, engine: nil}
}

// UpdateSettings applies the plugin settings and closes the recognizer when they change what it reads with.
func (h *Holder) UpdateSettings(values map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	next := settings.ParseRecognizerSettings(h.cfg.Recognize, values)
	if next == h.cfg.Recognize {
		return
	}
	h.cfg.Recognize = next
	h.dropLocked()
}

// GetEngine returns the open recognizer and opens it first when there is none.
func (h *Holder) GetEngine(ctx context.Context) (ports.Recognizer, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.engine != nil {
		return h.engine, nil
	}
	opened, err := h.open(context.WithoutCancel(ctx), h.cfg)
	if err != nil {
		return nil, err
	}
	h.engine = opened
	return opened, nil
}

// Close closes the open recognizer.
func (h *Holder) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeEngineLocked()
}

func (h *Holder) dropLocked() {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="Stderr logging of close failure does not affect engine state."
	if err := h.closeEngineLocked(); err != nil {
		h.reporter.ReportCloseError(err)
	}
}

func (h *Holder) closeEngineLocked() error {
	if h.engine == nil {
		return nil
	}
	err := h.engine.Close()
	h.engine = nil
	return err
}
