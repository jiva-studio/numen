package supervision

import (
	"cmp"
	"context"
	"slices"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// RunningProcesses lists the plugins the supervisors run and signals each change of them.
type RunningProcesses struct {
	sups *Supervisors
}

// NewRunningProcesses returns the list over the supervisors.
func NewRunningProcesses(sups *Supervisors) *RunningProcesses {
	return &RunningProcesses{sups: sups}
}

// ListProcesses returns the plugins that run, in the order of their ids. An idle one has the time from
// which EvictIdle stops it: its last use plus its idle limit, or its start for one never used.
func (r *RunningProcesses) ListProcesses() []plugindomain.Process {
	s := r.sups
	s.mu.RLock()
	defer s.mu.RUnlock()
	processes := make([]plugindomain.Process, 0, len(s.byPlugin))
	for id, sup := range s.byPlugin {
		usage, isRunning := sup.GetUsage()
		if !isRunning {
			continue
		}
		process := plugindomain.Process{
			PluginID: id, StartTime: getStartTimeLocked(s, id), LastUseTime: usage.LastActiveTime,
			LeaseCount: usage.LeaseCount, IsInProcess: usage.IsInProcess,
		}
		switch {
		case usage.IsIdle && usage.IsUsed:
			process.IdleStopTime = usage.LastActiveTime.Add(getIdleLimitLocked(s, id, IdleThreshold))
		case usage.IsIdle:
			process.IdleStopTime = usage.LastActiveTime
		}
		processes = append(processes, process)
	}
	slices.SortFunc(processes, func(a, b plugindomain.Process) int { return cmp.Compare(a.PluginID, b.PluginID) })
	return processes
}

// WatchProcesses signals each time a plugin starts, stops, or takes or returns a lease, until the
// context ends. Signals not received yet are merged into one.
func (r *RunningProcesses) WatchProcesses(ctx context.Context) <-chan struct{} {
	s := r.sups
	watcher := make(chan struct{}, 1)
	s.mu.Lock()
	s.watchers = append(s.watchers, watcher)
	s.mu.Unlock()
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.watchers = slices.DeleteFunc(s.watchers, func(one chan struct{}) bool { return one == watcher })
	}()
	return watcher
}
