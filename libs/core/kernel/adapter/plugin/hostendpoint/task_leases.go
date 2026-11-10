package hostendpoint

import "sync"

// taskLeases records the tasks each plugin reports as open. Each open task holds one lease of its
// plugin, so the host does not stop a plugin as idle while it works on its own.
type taskLeases struct {
	mu   sync.Mutex
	open map[string]map[string]bool
}

func (l *taskLeases) hold(callers Leases, pluginID, taskID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.open == nil {
		l.open = make(map[string]map[string]bool)
	}
	tasks := l.open[pluginID]
	if tasks == nil {
		tasks = make(map[string]bool)
		l.open[pluginID] = tasks
	}
	if tasks[taskID] {
		return
	}
	tasks[taskID] = true
	callers.Acquire(pluginID)
}

func (l *taskLeases) release(callers Leases, pluginID, taskID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	tasks := l.open[pluginID]
	if !tasks[taskID] {
		return
	}
	delete(tasks, taskID)
	if len(tasks) == 0 {
		delete(l.open, pluginID)
	}
	callers.Release(pluginID)
}

func (l *taskLeases) clear(pluginID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.open, pluginID)
}
