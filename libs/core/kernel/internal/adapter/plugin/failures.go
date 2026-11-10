package plugin

import (
	"context"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Failures keeps the plugins whose start failed, reports them to the callbacks registered, and
// forgets a failure once the plugin starts or is switched on or off.
type Failures struct {
	t     *table
	clock machineport.Clock
	errs  *errorSink

	mu              sync.Mutex
	pending         []plugindomain.Failure
	onPluginFailed  func(plugindomain.Failure)
	onCleared       func(pluginID string)
	reportedTargets map[string]struct{}
}

func newFailures(t *table, clock machineport.Clock, errs *errorSink) *Failures {
	return &Failures{t: t, clock: clock, errs: errs, reportedTargets: make(map[string]struct{})}
}

// SetErrorHandler registers a callback for problems the host meets outside a call, such as a failed shutdown.
func (f *Failures) SetErrorHandler(handler eventbus.ErrorHandler) {
	f.errs.set(handler)
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
	f.t.mu.RLock()
	defer f.t.mu.RUnlock()
	sup, ok := f.t.supervisors[pluginID]
	return ok && sup.GetActivationFailure() != nil
}

func (f *Failures) newFailureLocked(pluginID string, sup *lifecycle.Supervisor, cause error) plugindomain.Failure {
	return plugindomain.Failure{
		PluginID: pluginID,
		Name:     f.t.plugins[pluginID].Manifest.Name,
		Reason:   lifecycle.DescribeFailure(cause),
		Detail:   cause.Error(),
		At:       sup.GetFailedAt(),
	}
}

func (f *Failures) getFailureLocked(pluginID string) *plugindomain.Failure {
	sup, ok := f.t.supervisors[pluginID]
	if !ok {
		return nil
	}
	cause := sup.GetActivationFailure()
	if cause == nil {
		return nil
	}
	failure := f.newFailureLocked(pluginID, sup, cause)
	return &failure
}

func (f *Failures) enqueue(failure plugindomain.Failure) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending = append(f.pending, failure)
}

func (f *Failures) failActivation(ctx context.Context, pluginID string, sup *lifecycle.Supervisor, cause error) {
	f.t.mu.Lock()
	defer f.t.mu.Unlock()
	if sup.GetState() != lifecycle.StateActivating {
		return
	}
	sup.SetDormant(cause)
	if ctx.Err() == nil {
		sup.RecordActivationFailure(f.clock.Now(), cause)
		f.enqueue(f.newFailureLocked(pluginID, sup, cause))
	}
}

func (f *Failures) recordActivationFailure(pluginID string, cause error) *plugindomain.Failure {
	f.t.mu.Lock()
	defer f.t.mu.Unlock()
	sup, ok := f.t.supervisors[pluginID]
	if !ok || !sup.RecordActivationFailure(f.clock.Now(), cause) {
		return nil
	}
	failure := f.newFailureLocked(pluginID, sup, cause)
	f.enqueue(failure)
	return &failure
}

func (f *Failures) deliver() {
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
	f.t.mu.RLock()
	sup := f.t.supervisors[pluginID]
	f.t.mu.RUnlock()
	if sup != nil {
		f.clear(pluginID, sup)
	}
}

func (f *Failures) clear(pluginID string, sup *lifecycle.Supervisor) {
	f.forgetTargetReport(pluginID)
	if !sup.ClearActivationFailure() {
		return
	}
	f.mu.Lock()
	notify := f.onCleared
	f.mu.Unlock()
	if notify != nil {
		notify(pluginID)
	}
}

func (f *Failures) forgetTargetReport(pluginID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.reportedTargets, pluginID)
}

func (f *Failures) markTargetResult(pluginID string, err error) (wasReported bool) {
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
