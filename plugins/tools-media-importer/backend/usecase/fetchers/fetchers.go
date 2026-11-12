// Package fetchers holds the downloader and the fetch settings of the running plugin, and makes
// the downloader again once a setting about it changes.
package fetchers

import (
	"sync"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/settings"
)

// Holder serves the current downloader and settings to the links.
type Holder struct {
	mu      sync.RWMutex
	isMock  bool
	tools   download.Tools
	values  settings.Values
	current *download.Downloader
}

// New returns a holder with the default settings. It holds the mock downloader when isMock is set.
func New(isMock bool, tools download.Tools) *Holder {
	h := &Holder{
		mu: sync.RWMutex{}, isMock: isMock, tools: tools,
		values: settings.Defaults(), current: nil,
	}
	h.current = h.open()
	return h
}

func (h *Holder) open() *download.Downloader {
	if h.isMock {
		return download.NewMock(h.tools.Web)
	}
	return download.New(h.values.Programs, h.tools.Runner, h.tools.Files, h.tools.Paths, h.tools.Web)
}

// GetFetcher returns the current downloader.
func (h *Holder) GetFetcher() link.Fetcher {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.current
}

func (h *Holder) GetSettings() link.Settings {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.values.Fetch
}

// GetPrograms returns the programs the current downloader uses.
func (h *Holder) GetPrograms() download.Config {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.values.Programs
}

// UpdateConfig applies the setting values and makes the downloader again.
func (h *Holder) UpdateConfig(values map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = settings.Parse(values, h.values)
	h.current = h.open()
}
