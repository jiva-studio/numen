package platform

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/jiva-studio/numen/libs/core/container"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// storageFolder is the folder of the application inside the cache and data folders that holds
// what the plugins store.
const storageFolder = "plugin-storage"

// newPluginHost is the plugin host of this binary. It grants the plugins storage under the cache
// and data folders of this machine.
func newPluginHost(target string, dirs ...string) pluginport.PluginManager {
	host := container.NewPluginHost(target, dirs...)
	cacheRoot, dataRoot := getStorageRoots(runtime.GOOS, os.Getenv, os.UserCacheDir, os.UserHomeDir)
	host.SetStorageRoots(cacheRoot, dataRoot)
	// gomutants:disable-next-line
	if models := host.GetModelsDir(); models != "" {
		// gomutants:disable-next-line
		host.SetDependencies(container.NewDependenciesIn(models))
	}
	return host
}

// getStorageRoots is the folders the plugin cache directories and the plugin data directories are
// created under. A folder this machine does not provide is empty, and the host grants no directory
// under it.
func getStorageRoots(goos string, getEnv func(string) string, getCacheDir, getHome func() (string, error)) (cacheRoot, dataRoot string) {
	if dir, err := getCacheDir(); err == nil && dir != "" {
		cacheRoot = filepath.Join(dir, appFolder, storageFolder)
	}
	if dir, err := getDataDir(goos, getEnv, getHome); err == nil {
		dataRoot = filepath.Join(dir, appFolder, storageFolder)
	}
	return cacheRoot, dataRoot
}
