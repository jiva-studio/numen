// Package besthealth reports health to the host as a courtesy: a report the host refuses is logged and
// never fails the work that raised it.
package besthealth

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Health keeps the contract of ports.Health: Report and Clear return no error.
type Health struct {
	inner ports.Health
	warn  func(message string, err error)
}

// New returns a Health that hands items to inner and passes its errors to warn.
func New(inner ports.Health, warn func(message string, err error)) *Health {
	return &Health{inner: inner, warn: warn}
}

// Report raises the item and logs the error of the host.
func (h *Health) Report(ctx context.Context, item health.Item) error {
	if err := h.inner.Report(ctx, item); err != nil {
		h.warn("reporting the health item "+item.Key+" failed", err)
	}
	return nil
}

// Clear removes the item and logs the error of the host.
func (h *Health) Clear(ctx context.Context, key string) error {
	if err := h.inner.Clear(ctx, key); err != nil {
		h.warn("clearing the health item "+key+" failed", err)
	}
	return nil
}
