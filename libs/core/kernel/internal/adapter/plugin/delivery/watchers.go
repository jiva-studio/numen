package delivery

import "sync"

// watchers are the functions to run each time a plugin has been given its configuration.
type watchers struct {
	mu   sync.RWMutex
	byID map[string]map[int]func()
	next int
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

func (w *watchers) run(pluginID string) {
	w.mu.RLock()
	fns := make([]func(), 0, len(w.byID[pluginID]))
	for _, fn := range w.byID[pluginID] {
		fns = append(fns, fn)
	}
	w.mu.RUnlock()
	for _, fn := range fns {
		go fn()
	}
}
