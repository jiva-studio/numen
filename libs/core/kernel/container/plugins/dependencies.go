// Package plugins wires the plugin catalog, plugin host, agent provider and dependencies.
package plugins

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/onnxruntime"
	pluginhost "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const dependencyTaskPrefix = "provision:"

// EnvMockML names the variable that runs the plugins on mock models.
const EnvMockML = "NUMEN_MOCK_ML"

// NewDependenciesIn returns the provisioners as the environment names them now, keeping the
// dependencies in defaultDir where NUMEN_MODELS_DIR names no folder.
func NewDependenciesIn(defaultDir string) pluginhost.Dependencies {
	if os.Getenv(EnvMockML) == "true" {
		return provision.NewRegistry(newPlaceholders())
	}
	dir := defaultDir
	if models := os.Getenv(onnxruntime.EnvDir); models != "" {
		dir = models
	}
	runtimeConfig := onnxruntime.FromEnvironment(clock.System{})
	runtimeConfig.Dir = dir
	return provision.NewRegistry(
		onnxruntime.NewONNXProvisioner(runtimeConfig),
		provision.NewFileProvisioner(provision.Config{Dir: dir, Offline: onnxruntime.IsOffline(), Clock: clock.System{}}),
	)
}

func newPlaceholders() *provision.PlaceholderProvisioner {
	dir, err := os.UserCacheDir()
	if err == nil {
		dir = filepath.Join(dir, "numen", "placeholder-dependencies")
	} else if dir, err = os.MkdirTemp("", "numen-placeholder-dependencies-"); err != nil {
		// Placeholders written under the working folder still let a mock run start.
		dir = "numen-placeholder-dependencies"
	}
	return provision.NewPlaceholderProvisioner(dir, map[string]func(string) map[string]string{
		plugin.RuntimeONNX: func(primary string) map[string]string {
			return map[string]string{onnxruntime.EnvLibrary: primary, onnxruntime.EnvProvider: "cpu"}
		},
	})
}

// ReportDependencies keeps one task in the list for each dependency being provisioned. The
// task counts bytes, and is taken out when the dependency is ready; one that failed stays with its error.
func (c Config) ReportDependencies(tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
	c.PluginHost.ObserveDependencies(func(req manifest.DependencyRequirement, status plugin.DependencyStatus) {
		id := dependencyTaskPrefix + req.ID
		switch status.State {
		case plugin.DependencyStateReady:
			tasks.Remove(id)
		case plugin.DependencyStateMissing, plugin.DependencyStateProvisioning, plugin.DependencyStateFailed:
			tasks.Set(run.Task{
				ID: id, Title: "Downloading " + req.ID, Unit: run.Bytes,
				Count: status.BytesDownloaded, Total: status.BytesTotal, Error: status.Error, IsUserStarted: false,
			})
		}
	})
}
