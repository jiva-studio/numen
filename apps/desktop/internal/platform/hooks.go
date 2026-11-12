package platform

import "path/filepath"

// storageFolder is the folder of the application inside the cache and data folders that holds
// what the plugins store.
const storageFolder = "plugin-storage"

func getStorageRoots(goos string, getEnv func(string) string, getCacheDir, getHome func() (string, error)) (cacheRoot, dataRoot string) {
	if dir, err := getCacheDir(); err == nil && dir != "" {
		cacheRoot = filepath.Join(dir, appFolder, storageFolder)
	}
	if dir, err := getDataDir(goos, getEnv, getHome); err == nil {
		dataRoot = filepath.Join(dir, appFolder, storageFolder)
	}
	return cacheRoot, dataRoot
}
