package run

import (
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

const pluginFailureTaskPrefix = "plugin-failed:"

// PluginFailureTasks shows a plugin that stopped as a task until its failure is cleared.
type PluginFailureTasks struct {
	tasks *Tasks
}

func NewPluginFailureTasks(tasks *Tasks) PluginFailureTasks {
	return PluginFailureTasks{tasks: tasks}
}

// Report sets the task of the plugin that failed.
func (u PluginFailureTasks) Report(failure plugindomain.Failure) {
	u.tasks.Set(work.Task{
		ID:         pluginFailureTaskPrefix + failure.PluginID,
		Title:      "Plugin stopped",
		About:      failure.Name,
		Error:      failure.Reason,
		PluginID:   failure.PluginID,
		PluginName: failure.Name,
	})
}

// Clear removes the task of the plugin.
func (u PluginFailureTasks) Clear(pluginID string) {
	u.tasks.Remove(pluginFailureTaskPrefix + pluginID)
}
