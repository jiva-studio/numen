// Package fetchers holds the downloader and the fetch settings of the running plugin, and makes
// the downloader again once a setting about it changes.
package fetchers

import (
	"sync"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/ports"
)

// Open makes the downloader that fetches with programs.
type Open func(programs fetch.Config) ports.Fetcher

// Holder serves the current downloader and settings to the links.
type Holder struct {
	mu      sync.RWMutex
	open    Open
	values  values
	current ports.Fetcher
}

// New returns a holder with the default settings, whose downloader open makes.
func New(open Open) *Holder {
	h := &Holder{mu: sync.RWMutex{}, open: open, values: getDefaultValues(), current: nil}
	h.current = h.open(h.values.Programs)
	return h
}

// GetFetcher returns the current downloader.
func (h *Holder) GetFetcher() ports.Fetcher {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.current
}

func (h *Holder) GetSettings() fetch.Settings {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.values.Fetch
}

// GetPrograms returns the programs the current downloader uses.
func (h *Holder) GetPrograms() fetch.Config {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.values.Programs
}

// UpdateConfig applies the setting values and makes the downloader again.
func (h *Holder) UpdateConfig(settings map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = parseValues(settings, h.values)
	h.current = h.open(h.values.Programs)
}
