// Package failureevents dispatches changes to plugin failure state.
package failureevents

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"sync"
)

type Events struct {
	mu        sync.RWMutex
	onCleared func(string)
	onFailure func(plugin.Failure)
}

func (e *Events) SetOnFailure(fn func(plugin.Failure)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onFailure = fn
}
func (e *Events) SetOnCleared(fn func(string)) { e.mu.Lock(); defer e.mu.Unlock(); e.onCleared = fn }
func (e *Events) ReportFailure(failure plugin.Failure) {
	e.mu.RLock()
	fn := e.onFailure
	e.mu.RUnlock()
	if fn != nil {
		fn(failure)
	}
}
func (e *Events) ReportCleared(id string) {
	e.mu.RLock()
	fn := e.onCleared
	e.mu.RUnlock()
	if fn != nil {
		fn(id)
	}
}
