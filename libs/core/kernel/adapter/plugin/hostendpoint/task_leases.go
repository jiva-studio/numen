package hostendpoint

import (
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// taskLeases records the tasks each plugin reports as open. Each open task holds one own lease of the
// process of its plugin that reported it, so the host does not stop a plugin as idle while it works on its
// own.
type taskLeases struct {
	plugins sync.Map
}

type pluginTasks struct {
	mu   sync.Mutex
	open map[string]taskLease
}

// taskLease is the lease of one open task and the token of the process it was taken on.
type taskLease struct {
	holder plugin.Holder
	token  string
}

func (l *taskLeases) get(pluginID string) *pluginTasks {
	created := &pluginTasks{open: make(map[string]taskLease)}
	stored, _ := l.plugins.LoadOrStore(pluginID, created)
	if tasks, ok := stored.(*pluginTasks); ok {
		return tasks
	}
	return created
}

func (l *taskLeases) hold(callers Leases, pluginID, token, taskID, title string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	if held, isOpen := tasks.open[taskID]; isOpen && held.token == token {
		return
	}
	holder := plugin.Holder{PluginID: pluginID, Why: "task: " + title}
	if callers.AcquireOwn(pluginID, token, holder) {
		tasks.open[taskID] = taskLease{holder: holder, token: token}
	}
}

func (l *taskLeases) release(callers Leases, pluginID, taskID string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	held, isOpen := tasks.open[taskID]
	if !isOpen {
		return
	}
	delete(tasks.open, taskID)
	callers.ReleaseOwn(pluginID, held.token, held.holder)
}

func (l *taskLeases) clear(pluginID string) {
	tasks := l.get(pluginID)
	tasks.mu.Lock()
	defer tasks.mu.Unlock()
	clear(tasks.open)
}
