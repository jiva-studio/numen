package platform

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// storageFolder is the folder of the application inside the cache and data folders that holds
// what the plugins store.
const storageFolder = "plugin-storage"

func newPluginHost(target string, dirs ...string) *container.PluginHost {
	host := container.NewPluginHost(target, dirs...)
	cacheRoot, dataRoot := getStorageRoots(runtime.GOOS, os.Getenv, os.UserCacheDir, os.UserHomeDir)
	host.SetStorage(cacheRoot, dataRoot)
	return host
}

func getStorageRoots(goos string, getEnv func(string) string, getCacheDir, getHome func() (string, error)) (cacheRoot, dataRoot string) {
	if dir, err := getCacheDir(); err == nil && dir != "" {
		cacheRoot = filepath.Join(dir, appFolder, storageFolder)
	}
	if dir, err := getDataDir(goos, getEnv, getHome); err == nil {
		dataRoot = filepath.Join(dir, appFolder, storageFolder)
	}
	return cacheRoot, dataRoot
}
