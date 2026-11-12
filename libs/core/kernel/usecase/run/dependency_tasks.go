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

// Execute sets the task of the dependency with the id while it is missing, being provisioned or failed, and
// removes it once the dependency is ready.
func (u DependencyTasks) Execute(id string, status plugindomain.DependencyStatus) {
	taskID := dependencyTaskPrefix + id
	switch status.State {
	case plugindomain.DependencyStateReady:
		u.tasks.Remove(taskID)
	case plugindomain.DependencyStateMissing, plugindomain.DependencyStateProvisioning, plugindomain.DependencyStateFailed:
		u.tasks.Set(work.Task{
			ID: taskID, Title: "Downloading " + id, Unit: work.Bytes,
			Count: status.BytesDownloaded, Total: status.BytesTotal, Error: status.Error, IsUserStarted: false,
		})
	}
}
