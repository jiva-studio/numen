// Package engines holds the recogniser of the plugin: it opens it when a page is first read and
// opens it again once a setting that decides what it reads with changes.
package engines

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/settings"
)

// Recogniser is the recogniser of a page and its contents.
type Recogniser interface {
	recognition.Recogniser
	Close() error
}

// Opener opens the recogniser the configuration names.
type Opener func(ctx context.Context, cfg settings.Config) (Recogniser, error)

// Reporter receives the failures the holder cannot return.
type Reporter interface {
	ReportCloseError(err error)
}

// Holder keeps one recogniser for the current configuration.
type Holder struct {
	mu       sync.Mutex
	cfg      settings.Config
	open     Opener
	reporter Reporter
	engine   Recogniser
}

// NewHolder returns a holder that opens its recogniser with open for cfg.
func NewHolder(cfg settings.Config, open Opener, reporter Reporter) *Holder {
	return &Holder{mu: sync.Mutex{}, cfg: cfg, open: open, reporter: reporter, engine: nil}
}

// UpdateSettings applies the plugin settings and closes the recogniser when they change what it reads with.
func (h *Holder) UpdateSettings(values map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	next := settings.ParseRecogniserSettings(h.cfg.Recognize, values)
	if next == h.cfg.Recognize {
		return
	}
	h.cfg.Recognize = next
	h.dropLocked()
}

// GetEngine returns the open recogniser and opens it first when there is none.
func (h *Holder) GetEngine(ctx context.Context) (recognition.Recogniser, error) {
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

// Close closes the open recogniser.
func (h *Holder) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeLocked()
}

func (h *Holder) dropLocked() {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="Stderr logging of close failure does not affect engine state."
	if err := h.closeLocked(); err != nil {
		h.reporter.ReportCloseError(err)
	}
}

func (h *Holder) closeLocked() error {
	if h.engine == nil {
		return nil
	}
	err := h.engine.Close()
	h.engine = nil
	return err
}
