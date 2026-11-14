package delivery

import "sync"

// watchers are the functions to run each time a plugin has been given its configuration, with the
// revision of the configuration they last ran for, by plugin.
type watchers struct {
	mu       sync.RWMutex
	byID     map[string]map[int]func()
	notified map[string]string
	next     int
}

func (w *watchers) add(pluginID string, fn func()) (stop func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.byID == nil {
		w.byID = make(map[string]map[int]func())
	}
	if w.byID[pluginID] == nil {
		w.byID[pluginID] = make(map[int]func())
	}
	id := w.next
	w.next++
	w.byID[pluginID][id] = fn
	return func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		delete(w.byID[pluginID], id)
	}
}

func (w *watchers) run(pluginID, revision string) {
	w.mu.Lock()
	if w.notified == nil {
		w.notified = make(map[string]string)
	}
	w.notified[pluginID] = revision
	fns := make([]func(), 0, len(w.byID[pluginID]))
	for _, fn := range w.byID[pluginID] {
		fns = append(fns, fn)
	}
	w.mu.Unlock()
	for _, fn := range fns {
		go fn()
	}
}

func (w *watchers) runIfChanged(pluginID, revision string) {
	w.mu.RLock()
	last, isNotified := w.notified[pluginID]
	w.mu.RUnlock()
	if isNotified && last == revision {
		return
	}
	w.run(pluginID, revision)
}
