package hostendpoint

import (
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// taskLeases records the tasks each plugin reports as open. Each open task holds one lease of its
// plugin, so the host does not stop a plugin as idle while it works on its own.
type taskLeases struct {
	plugins sync.Map
}

type pluginTasks struct {
	mu   sync.Mutex
	open map[string]plugin.Holder
}

func (l *taskLeases) get(pluginID string) *pluginTasks {
	created := &pluginTasks{open: make(map[string]plugin.Holder)}
	stored, _ := l.plugins.LoadOrStore(pluginID, created)
	if tasks, ok := stored.(*pluginTasks); ok {
		return tasks
	}
	return created
}

func (l *taskLeases) hold(callers Leases, pluginID, taskID, title string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	if _, isOpen := tasks.open[taskID]; isOpen {
		return
	}
	holder := plugin.Holder{Who: "plugin " + pluginID, Why: "task " + title}
	tasks.open[taskID] = holder
	callers.Acquire(pluginID, holder)
}

func (l *taskLeases) release(callers Leases, pluginID, taskID string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	holder, isOpen := tasks.open[taskID]
	if !isOpen {
		return
	}
	delete(tasks.open, taskID)
	callers.Release(pluginID, holder)
}

func (l *taskLeases) clear(pluginID string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	clear(tasks.open)
}
