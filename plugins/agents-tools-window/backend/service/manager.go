package service

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// timeSource gets the current time.
type timeSource interface {
	Now() time.Time
}

// WindowManager keeps the edit previews.
type WindowManager struct {
	mu         sync.RWMutex
	previews   map[string]EditPreview
	previewSeq atomic.Uint64
	clock      timeSource
}

// NewManager creates a window manager instance.
func NewManager(clock timeSource) *WindowManager {
	return &WindowManager{
		mu:         sync.RWMutex{},
		previews:   make(map[string]EditPreview),
		previewSeq: atomic.Uint64{},
		clock:      clock,
	}
}

// SavePreview stores an edit preview.
func (m *WindowManager) SavePreview(preview EditPreview) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.previews[preview.ID] = preview
}

// GetPreview retrieves a preview by its identifier.
func (m *WindowManager) GetPreview(id string) (EditPreview, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.previews[id]
	return p, ok
}

// ListPreviews returns all stored edit previews.
func (m *WindowManager) ListPreviews() []EditPreview {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]EditPreview, 0, len(m.previews))
	for _, p := range m.previews {
		result = append(result, p)
	}
	return result
}

// ClearPreviews removes all stored previews.
func (m *WindowManager) ClearPreviews() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.previews = make(map[string]EditPreview)
}

// NextPreviewID generates a unique identifier for an edit preview.
func (m *WindowManager) NextPreviewID() string {
	seq := m.previewSeq.Add(1)
	return fmt.Sprintf("prev-%d-%d", m.clock.Now().UnixNano(), seq)
}

// GetNow gets the current time of the manager's clock.
func (m *WindowManager) GetNow() time.Time {
	return m.clock.Now()
}
