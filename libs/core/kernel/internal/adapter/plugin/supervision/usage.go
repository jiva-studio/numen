package supervision

import (
	"cmp"
	"context"
	"slices"
	"sync"

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
// which EvictIdle stops it.
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
			PluginID: id, StartTime: usage.StartTime, LastUseTime: usage.LastActiveTime,
			LeaseCount: usage.LeaseCount, IsInProcess: usage.IsInProcess,
		}
		if usage.IsIdle {
			process.IdleStopTime = usage.LastActiveTime.Add(IdleThreshold)
		}
		processes = append(processes, process)
	}
	slices.SortFunc(processes, func(a, b plugindomain.Process) int { return cmp.Compare(a.PluginID, b.PluginID) })
	return processes
}

// WatchProcesses signals each time a plugin the supervisors registered starts, stops, or takes or
// returns a lease, until the context ends. Signals not received yet are merged into one.
func (r *RunningProcesses) WatchProcesses(ctx context.Context) <-chan struct{} {
	return r.sups.changes.watch(ctx)
}

// changeSignal is the watchers of the changes of the plugins.
type changeSignal struct {
	mu       sync.Mutex
	watchers []chan struct{}
}

func (c *changeSignal) watch(ctx context.Context) <-chan struct{} {
	watcher := make(chan struct{}, 1)
	c.mu.Lock()
	c.watchers = append(c.watchers, watcher)
	c.mu.Unlock()
	go func() {
		<-ctx.Done()
		c.mu.Lock()
		defer c.mu.Unlock()
		c.watchers = slices.DeleteFunc(c.watchers, func(one chan struct{}) bool { return one == watcher })
	}()
	return watcher
}

func (c *changeSignal) notify() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, watcher := range c.watchers {
		select {
		case watcher <- struct{}{}:
		default:
		}
	}
}
