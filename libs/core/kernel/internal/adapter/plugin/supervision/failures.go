package supervision

import (
	"context"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/errsink"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Failures keeps the plugins whose start failed, reports them to the callbacks registered, and
// forgets a failure once the plugin starts or is switched on or off.
type Failures struct {
	sups  *Supervisors
	clock machineport.Clock
	errs  *errsink.Sink

	mu              sync.Mutex
	pending         []plugindomain.Failure
	onPluginFailed  func(plugindomain.Failure)
	onCleared       func(pluginID string)
	reportedTargets map[string]struct{}
}

// NewFailures returns the failures of the plugins the supervisors track. Problems the host meets go to errs.
func NewFailures(sups *Supervisors, clock machineport.Clock, errs *errsink.Sink) *Failures {
	return &Failures{sups: sups, clock: clock, errs: errs, reportedTargets: make(map[string]struct{})}
}

// SetErrorHandler registers a callback for problems the host meets outside a call, such as a failed shutdown.
func (f *Failures) SetErrorHandler(handler eventbus.ErrorHandler) {
	f.errs.Set(handler)
}

// SetOnPluginFailed registers a callback invoked each time a start of a plugin fails.
func (f *Failures) SetOnPluginFailed(fn func(plugindomain.Failure)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onPluginFailed = fn
}

// SetOnFailureCleared registers a callback invoked with the id of a plugin whose failure the host forgets, because the plugin started, was enabled or was disabled.
func (f *Failures) SetOnFailureCleared(fn func(pluginID string)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onCleared = fn
}

// IsPluginSuspended reports whether the last start of the plugin failed, which holds it suspended until
// its failure is cleared.
func (f *Failures) IsPluginSuspended(pluginID string) bool {
	sup, ok := f.sups.Get(pluginID)
	return ok && sup.Failure.GetCause() != nil
}

func newFailure(pluginID, name string, sup *lifecycle.Supervisor, cause error) plugindomain.Failure {
	return plugindomain.Failure{
		PluginID: pluginID,
		Name:     name,
		Reason:   lifecycle.DescribeFailure(cause),
		Detail:   cause.Error(),
		At:       sup.Failure.GetFailedAt(),
	}
}

// Get returns the failure of the plugin's last start, or nil.
func (f *Failures) Get(pluginID, name string) *plugindomain.Failure {
	sup, ok := f.sups.Get(pluginID)
	if !ok {
		return nil
	}
	cause := sup.Failure.GetCause()
	if cause == nil {
		return nil
	}
	failure := newFailure(pluginID, name, sup, cause)
	return &failure
}

func (f *Failures) enqueue(failure plugindomain.Failure) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending = append(f.pending, failure)
}

// FailActivation sets a plugin whose start failed dormant, and records and queues the failure unless ctx ended.
func (f *Failures) FailActivation(ctx context.Context, pluginID, name string, sup *lifecycle.Supervisor, cause error) {
	if sup.GetState() != lifecycle.StateActivating {
		return
	}
	sup.SetDormant(cause)
	if ctx.Err() == nil {
		sup.Failure.Record(f.clock.Now(), cause)
		f.enqueue(newFailure(pluginID, name, sup, cause))
	}
}

// RecordActivationFailure records why a plugin failed outside its start, and returns the failure unless the plugin had one.
func (f *Failures) RecordActivationFailure(pluginID, name string, cause error) *plugindomain.Failure {
	sup, ok := f.sups.Get(pluginID)
	if !ok || !sup.Failure.Record(f.clock.Now(), cause) {
		return nil
	}
	failure := newFailure(pluginID, name, sup, cause)
	f.enqueue(failure)
	return &failure
}

// Deliver hands the failures queued to the callback registered.
func (f *Failures) Deliver() {
	f.mu.Lock()
	failures := f.pending
	f.pending = nil
	notify := f.onPluginFailed
	f.mu.Unlock()

	if notify == nil {
		return
	}
	for _, failure := range failures {
		notify(failure)
	}
}

// ClearFailure forgets the failure of the plugin, which makes it startable again.
func (f *Failures) ClearFailure(pluginID string) {
	if sup, ok := f.sups.Get(pluginID); ok && sup != nil {
		f.Clear(pluginID, sup)
	}
}

// Clear forgets the failure of the supervisor of a plugin and tells the callback registered.
func (f *Failures) Clear(pluginID string, sup *lifecycle.Supervisor) {
	f.ForgetTargetReport(pluginID)
	if !sup.Failure.Clear() {
		return
	}
	f.mu.Lock()
	notify := f.onCleared
	f.mu.Unlock()
	if notify != nil {
		notify(pluginID)
	}
}

// ForgetTargetReport forgets that the failure of a plugin was reported through a listener.
func (f *Failures) ForgetTargetReport(pluginID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.reportedTargets, pluginID)
}

// MarkTargetResult records the result of starting a plugin for a listener and returns whether a failure was reported before.
func (f *Failures) MarkTargetResult(pluginID string, err error) (wasReported bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, wasReported = f.reportedTargets[pluginID]
	if err == nil {
		delete(f.reportedTargets, pluginID)
	} else {
		f.reportedTargets[pluginID] = struct{}{}
	}
	return wasReported
}
