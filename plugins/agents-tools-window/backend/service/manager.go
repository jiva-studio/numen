package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errMissingFocusTarget = errors.New("specify tabId, path, or title to focus")
	errTabNotFound        = errors.New("no open tab matches criteria")
)

// WindowManager coordinates open tabs and edit previews.
type WindowManager struct {
	mu         sync.RWMutex
	tabs       []TabInfo
	frontID    string
	previews   map[string]EditPreview
	previewSeq atomic.Uint64
}

// NewManager creates a window manager instance.
func NewManager() *WindowManager {
	return &WindowManager{
		previews: make(map[string]EditPreview),
	}
}

// SetOpenTabs sets the currently open tabs.
func (m *WindowManager) SetOpenTabs(tabs []TabInfo, frontID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tabs = make([]TabInfo, len(tabs))
	copy(m.tabs, tabs)
	m.frontID = frontID

	for i := range m.tabs {
		m.tabs[i].IsFront = (m.tabs[i].ID != "" && m.tabs[i].ID == frontID)
	}
}

// GetOpenTabs returns the currently open tabs and the front tab ID.
func (m *WindowManager) GetOpenTabs() ([]TabInfo, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copied := make([]TabInfo, len(m.tabs))
	copy(copied, m.tabs)
	return copied, m.frontID
}

// FocusTab focuses or switches to a tab matching the input.
func (m *WindowManager) FocusTab(_ context.Context, in WindowFocusTabInput) (TabInfo, error) {
	if in.TabID == "" && in.Path == "" && in.Title == "" {
		return TabInfo{}, errMissingFocusTarget
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if idx := findMatchingTabIndex(m.tabs, in); idx >= 0 {
		return m.activateTab(idx), nil
	}

	if in.Path != "" {
		newTab := createTabFromPath(in.Path, in.TabID, in.Title, len(m.tabs)+1)
		for j := range m.tabs {
			m.tabs[j].IsFront = false
		}
		m.tabs = append(m.tabs, newTab)
		m.frontID = newTab.ID
		return newTab, nil
	}

	return TabInfo{}, errTabNotFound
}

func (m *WindowManager) activateTab(idx int) TabInfo {
	m.frontID = m.tabs[idx].ID
	for j := range m.tabs {
		m.tabs[j].IsFront = (j == idx)
	}
	m.tabs[idx].IsFront = true
	return m.tabs[idx]
}

func findMatchingTabIndex(tabs []TabInfo, in WindowFocusTabInput) int {
	for i, tab := range tabs {
		if in.TabID != "" && tab.ID == in.TabID {
			return i
		}
		if in.Path != "" && tab.Path == in.Path {
			return i
		}
		if in.Title != "" && tab.Title == in.Title {
			return i
		}
	}
	return -1
}

func createTabFromPath(path, tabID, title string, nextIndex int) TabInfo {
	if tabID == "" {
		tabID = fmt.Sprintf("tab-%d", nextIndex)
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return TabInfo{
		ID:      tabID,
		Kind:    "note",
		Path:    path,
		Title:   title,
		IsFront: true,
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
	return fmt.Sprintf("prev-%d-%d", time.Now().UnixNano(), seq)
}
