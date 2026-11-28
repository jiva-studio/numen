// Package failureevents dispatches changes to plugin failure state.
package failureevents

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"sync"
)

type Events struct {
	mu            sync.RWMutex
	reported      map[string]struct{}
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

// ForgetTargetReport forgets that the failure of a plugin was reported through a listener.
func (e *Events) ForgetTargetReport(pluginID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.reported, pluginID)
}

// MarkTargetResult records the result of starting a plugin for a listener and returns whether a failure was
// reported before.
func (e *Events) MarkTargetResult(pluginID string, err error) (wasReported bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, wasReported = e.reported[pluginID]
	if err == nil {
		delete(e.reported, pluginID)
		return wasReported
	}
	if e.reported == nil {
		e.reported = make(map[string]struct{})
	}
	e.reported[pluginID] = struct{}{}
	return wasReported
}
