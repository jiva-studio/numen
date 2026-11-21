package run

import (
	"net/url"
	"path"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const dependencyTaskPrefix = "provision:"

// DependencyTasks shows the provisioning of a plugin's dependency as a task until the dependency is ready.
type DependencyTasks struct {
	tasks *Tasks
}

func NewDependencyTasks(tasks *Tasks) DependencyTasks {
	return DependencyTasks{tasks: tasks}
}

// Execute sets the task of the dependency while it is being provisioned or failed, titled "Downloading" with the
// readable name of the dependency below it, with cancel as the stop of its download, and removes it once the
// dependency is ready or its download is canceled.
func (u DependencyTasks) Execute(req manifest.DependencyRequirement, status plugindomain.DependencyStatus, cancel func()) {
	taskID := dependencyTaskPrefix + req.ID
	switch status.State {
	case plugindomain.DependencyStateReady, plugindomain.DependencyStateMissing:
		u.tasks.Remove(taskID)
	case plugindomain.DependencyStateProvisioning, plugindomain.DependencyStateFailed:
		u.tasks.Set(work.Task{
			ID: taskID, Title: "Downloading", About: getDependencyName(req), Unit: work.Bytes,
			Count: status.BytesDownloaded, Total: status.BytesTotal, Error: status.Error, IsUserStarted: false, Cancel: cancel,
		})
	}
}

func getDependencyName(req manifest.DependencyRequirement) string {
	if named := req.Metadata["fileName"]; named != "" {
		return named
	}
	address := req.SourceURL
	if len(req.SourceURLs) > 0 {
		address = req.SourceURLs[0]
	}
	if parsed, err := url.Parse(address); err == nil {
		if base := path.Base(parsed.Path); base != "" && base != "." && base != "/" {
			return base
		}
	}
	return req.ID
}
