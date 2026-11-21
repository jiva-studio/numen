package run

import (
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

const dependencyTaskPrefix = "provision:"

// DependencyTasks shows the provisioning of a plugin's dependency as a task until the dependency is ready.
type DependencyTasks struct {
	tasks *Tasks
}

func NewDependencyTasks(tasks *Tasks) DependencyTasks {
	return DependencyTasks{tasks: tasks}
}

// Execute sets the task of the dependency with the id, shown under name, while it is being provisioned or failed, with cancel as the
// stop of its download, and removes it once the dependency is ready or its download is canceled.
func (u DependencyTasks) Execute(id, name string, status plugindomain.DependencyStatus, cancel func()) {
	taskID := dependencyTaskPrefix + id
	switch status.State {
	case plugindomain.DependencyStateReady, plugindomain.DependencyStateMissing:
		u.tasks.Remove(taskID)
	case plugindomain.DependencyStateProvisioning, plugindomain.DependencyStateFailed:
		u.tasks.Set(work.Task{
			ID: taskID, Title: "Downloading " + name, Unit: work.Bytes,
			Count: status.BytesDownloaded, Total: status.BytesTotal, Error: status.Error, IsUserStarted: false, Cancel: cancel,
		})
	}
}
