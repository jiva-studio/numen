// Package failureevents dispatches changes to plugin failure state.
package failureevents

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"sync"
)

type Events struct {
	mu            sync.RWMutex
	handleCleared func(string)
	handleFailure func(plugin.Failure)
}

func (e *Events) SetOnFailure(fn func(plugin.Failure)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handleFailure = fn
}
func (e *Events) SetOnCleared(fn func(string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handleCleared = fn
}
func (e *Events) ReportFailure(failure plugin.Failure) {
	e.mu.RLock()
	fn := e.handleFailure
	e.mu.RUnlock()
	if fn != nil {
		fn(failure)
	}
}
func (e *Events) ReportCleared(id string) {
	e.mu.RLock()
	fn := e.handleCleared
	e.mu.RUnlock()
	if fn != nil {
		fn(id)
	}
}
