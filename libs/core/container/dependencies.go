package container

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/onnxruntime"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/usecase/run"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const dependencyTaskPrefix = "provision:"

// envMockML names the variable that runs the plugins on mock models.
const envMockML = "NUMEN_MOCK_ML"

// newDependencies returns the provisioners together: the ONNX Runtime, and any
// other dependency that is a file with a pinned checksum. Fetching follows
// NUMEN_OFFLINE, and the folder follows NUMEN_MODELS_DIR when it is set. Plugins
// on mock models open no model and load no runtime, so then every dependency is
// an empty placeholder and nothing is fetched.
func newDependencies() *provision.Registry {
	if os.Getenv(envMockML) == "true" {
		return provision.NewRegistry(newPlaceholders())
	}
	dir := ""
	if models := os.Getenv(onnxruntime.EnvDir); models != "" {
		dir = filepath.Join(models, "dependencies")
	}
	return provision.NewRegistry(
		onnxruntime.NewONNXProvisioner(onnxruntime.FromEnvironment(clock.System{})),
		provision.NewFileProvisioner(provision.Config{Dir: dir, Offline: onnxruntime.IsOffline(), Clock: clock.System{}}),
	)
}

// newPlaceholders keeps the placeholders under the cache folder of the user.
func newPlaceholders() *provision.PlaceholderProvisioner {
	dir := filepath.Join(os.TempDir(), "numen-placeholder-dependencies")
	if cache, err := os.UserCacheDir(); err == nil {
		dir = filepath.Join(cache, "numen", "placeholder-dependencies")
	}
	return provision.NewPlaceholderProvisioner(dir, map[string]func(string) map[string]string{
		pluginport.RuntimeONNX: func(primary string) map[string]string {
			return map[string]string{onnxruntime.EnvLibrary: primary, onnxruntime.EnvProvider: "cpu"}
		},
	})
}

// NewDependencies returns the provisioners as the environment names them now: a
// host made before NUMEN_MOCK_ML was set is handed these again.
func NewDependencies() pluginport.DependencyCatalog { return newDependencies() }

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
