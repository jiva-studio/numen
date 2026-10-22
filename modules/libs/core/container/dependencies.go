package container

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/provision"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

const dependencyTaskPrefix = "provision:"

// newDependencies returns the provisioners together: the ONNX Runtime, and any
// other dependency that is a file with a pinned checksum. Fetching follows
// NUMEN_OFFLINE, and the folder follows NUMEN_MODELS_DIR when it is set.
func newDependencies() *provision.Registry {
	dir := ""
	if models := os.Getenv(onnxruntime.EnvDir); models != "" {
		dir = filepath.Join(models, "dependencies")
	}
	return provision.NewRegistry(
		onnxruntime.NewONNXProvisioner(onnxruntime.FromEnvironment()),
		provision.NewFileProvisioner(provision.Config{Dir: dir, Offline: onnxruntime.IsOffline()}),
	)
}

// ReportDependencies keeps one task in the list for each dependency being provisioned. The
// task counts bytes, and is taken out when the dependency is ready; one that failed stays with its error.
func (c Config) ReportDependencies(tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
	catalog := c.PluginHost.GetDependencies()
	if catalog == nil {
		return
	}
	catalog.SetObserver(func(req pluginport.DependencyRequirement, status pluginport.DependencyStatus) {
		id := dependencyTaskPrefix + req.ID
		switch status.State {
		case pluginport.DependencyStateReady:
			tasks.Remove(id)
		case pluginport.DependencyStateMissing, pluginport.DependencyStateProvisioning, pluginport.DependencyStateFailed:
			tasks.Set(run.Task{
				ID: id, Title: "Downloading " + req.ID, Unit: run.Bytes,
				Count: status.BytesDownloaded, Total: status.BytesTotal, Error: status.Error, IsUserStarted: false,
			})
		}
	})
}
