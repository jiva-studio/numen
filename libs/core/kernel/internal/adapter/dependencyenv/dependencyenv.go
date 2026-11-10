// Package dependencyenv opens the registry of the dependencies the plugins provision, as the process
// environment and the machine's cache folder say.
package dependencyenv

import (
	"os"
	"path/filepath"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/onnxruntime"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// EnvMockML names the variable that runs the plugins on mock models.
const EnvMockML = "NUMEN_MOCK_ML"

// Open returns the registry of the dependencies the plugins provision, in the folder given unless the
// environment names another. With NUMEN_MOCK_ML set to true it provisions empty placeholder files.
func Open(defaultDir string) pluginport.DependencyRegistry {
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
		plugindomain.RuntimeONNX: func(primary string) map[string]string {
			return map[string]string{onnxruntime.EnvLibrary: primary, onnxruntime.EnvProvider: "cpu"}
		},
	})
}
