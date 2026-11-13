package supervision

import (
	"cmp"
	"context"
	"slices"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// ListProcesses returns the plugins that run, in the order of their ids. An idle one has the time from
// which EvictIdle stops it.
func (s *Supervisors) ListProcesses() []plugindomain.Process {
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

// WatchProcesses signals each time a plugin starts, stops, or takes or returns a lease, until the
// context ends. Signals not received yet are merged into one.
func (s *Supervisors) WatchProcesses(ctx context.Context) <-chan struct{} {
	watcher := make(chan struct{}, 1)
	s.watchMu.Lock()
	s.watchers = append(s.watchers, watcher)
	s.watchMu.Unlock()
	go func() {
		<-ctx.Done()
		s.watchMu.Lock()
		defer s.watchMu.Unlock()
		s.watchers = slices.DeleteFunc(s.watchers, func(one chan struct{}) bool { return one == watcher })
	}()
	return watcher
}

func (s *Supervisors) notifyChange() {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	for _, watcher := range s.watchers {
		select {
		case watcher <- struct{}{}:
		default:
		}
	}
}
