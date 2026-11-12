package hostendpoint

import "sync"

// taskLeases records the tasks each plugin reports as open. Each open task holds one lease of its
// plugin, so the host does not stop a plugin as idle while it works on its own.
type taskLeases struct {
	plugins sync.Map
}

type pluginTasks struct {
	mu   sync.Mutex
	open map[string]bool
}

func (l *taskLeases) get(pluginID string) *pluginTasks {
	created := &pluginTasks{open: make(map[string]bool)}
	stored, _ := l.plugins.LoadOrStore(pluginID, created)
	if tasks, ok := stored.(*pluginTasks); ok {
		return tasks
	}
	return created
}

func (l *taskLeases) hold(callers Leases, pluginID, taskID string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	if tasks.open[taskID] {
		return
	}
	tasks.open[taskID] = true
	callers.Acquire(pluginID)
}

func (l *taskLeases) release(callers Leases, pluginID, taskID string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	if !tasks.open[taskID] {
		return
	}
	delete(tasks.open, taskID)
	callers.Release(pluginID)
}

func (l *taskLeases) clear(pluginID string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	clear(tasks.open)
}
