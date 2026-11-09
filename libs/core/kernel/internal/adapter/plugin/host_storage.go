package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// versionMarker is the file inside a plugin cache directory that records the version of the plugin
// the directory was filled by.
const versionMarker = ".plugin-version"

// storageRoots are the folders the host grants plugin storage under. An empty root grants none.
type storageRoots struct {
	cache string
	data  string
}

// SetStorageRoots sets the folders the plugin cache directories and plugin data directories are
// created under, as <cacheRoot>/plugins/<plugin id> and <dataRoot>/plugins/<plugin id>. The
// application chooses them from the platform's own folders.
func (h *Host) SetStorageRoots(cacheRoot, dataRoot string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.storage = storageRoots{cache: cacheRoot, data: dataRoot}
}

func (h *Host) getStorageVars(pluginID string) ([]string, error) {
	h.mu.RLock()
	roots := h.storage
	version := h.plugins[pluginID].Manifest.Version
	h.mu.RUnlock()
	if err := checkStorageID(pluginID); err != nil {
		return nil, err
	}
	var vars []string
	sharedCache, err := prepareSharedDir(roots.cache)
	if err != nil {
		return nil, fmt.Errorf("prepare shared cache directory: %w", err)
	}
	if sharedCache != "" {
		vars = append(vars, "NUMEN_SHARED_CACHE_DIR="+sharedCache)
	}
	sharedData, err := prepareSharedDir(roots.data)
	if err != nil {
		return nil, fmt.Errorf("prepare shared data directory: %w", err)
	}
	if sharedData != "" {
		vars = append(vars, "NUMEN_SHARED_DATA_DIR="+sharedData)
		vars = append(vars, "NUMEN_MODELS_DIR="+h.GetModelsDir())
	}
	if roots.cache != "" {
		dir := getPluginStorageDir(roots.cache, pluginID)
		if err := prepareCacheDir(dir, version); err != nil {
			return nil, fmt.Errorf("prepare cache directory of plugin %s: %w", pluginID, err)
		}
		vars = append(vars, "NUMEN_PLUGIN_CACHE_DIR="+dir)
	}
	if roots.data != "" {
		dir := getPluginStorageDir(roots.data, pluginID)
		if err := createPrivateDir(dir); err != nil {
			return nil, fmt.Errorf("prepare data directory of plugin %s: %w", pluginID, err)
		}
		vars = append(vars, "NUMEN_PLUGIN_DATA_DIR="+dir)
	}
	return vars, nil
}

func (h *Host) removeCacheDir(pluginID string) {
	h.mu.RLock()
	root := h.storage.cache
	h.mu.RUnlock()
	if root == "" || checkStorageID(pluginID) != nil {
		return
	}
	if err := os.RemoveAll(getPluginStorageDir(root, pluginID)); err != nil {
		h.reportError(fmt.Errorf("remove cache directory of plugin %s: %w", pluginID, err))
	}
}

func getPluginStorageDir(root, pluginID string) string {
	return filepath.Join(root, "plugins", pluginID)
}

func checkStorageID(pluginID string) error {
	if pluginID == "" || pluginID == "." || pluginID == ".." || strings.ContainsAny(pluginID, `/\`) {
		return fmt.Errorf("plugin id %q does not name a storage directory", pluginID)
	}
	return nil
}

func prepareCacheDir(dir, version string) error {
	marker := filepath.Join(dir, versionMarker)
	recorded, err := os.ReadFile(marker)
	if err == nil && string(recorded) != version {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	if err := createPrivateDir(dir); err != nil {
		return err
	}
	if err == nil && string(recorded) == version {
		return nil
	}
	return os.WriteFile(marker, []byte(version), 0o600)
}

func createPrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	return os.Chmod(dir, 0o700)
}

// sharedFolder is the folder inside the cache root that every plugin may read and write.
const sharedFolder = "shared"

// modelsFolder is the folder inside the shared directory that holds the models of the machine.
const modelsFolder = "models"

// envModelsDir is the variable a user sets to move the models off the shared data directory. The host
// passes the folder it resolved to every plugin under the same name.
const envModelsDir = "NUMEN_MODELS_DIR"

// GetSharedDir returns the directory all plugins share for what can be computed again, or empty where
// the host has no cache root. The host never deletes it: a plugin's version change leaves it alone.
func (h *Host) GetSharedDir() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.storage.cache == "" {
		return ""
	}
	return filepath.Join(h.storage.cache, sharedFolder)
}

// GetSharedDataDir returns the directory all plugins share for what is downloaded or made and cannot be
// computed again, or empty where the host has no data root. The host never deletes it.
func (h *Host) GetSharedDataDir() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.storage.data == "" {
		return ""
	}
	return filepath.Join(h.storage.data, sharedFolder)
}

// GetModelsDir returns the folder where models are kept once for every plugin: NUMEN_MODELS_DIR when the
// user sets it, else models inside the shared data directory. It is empty where the host has no data root.
func (h *Host) GetModelsDir() string {
	if dir := os.Getenv(envModelsDir); dir != "" {
		return dir
	}
	shared := h.GetSharedDataDir()
	if shared == "" {
		return ""
	}
	return filepath.Join(shared, modelsFolder)
}

func prepareSharedDir(cacheRoot string) (string, error) {
	if cacheRoot == "" {
		return "", nil
	}
	dir := filepath.Join(cacheRoot, sharedFolder)
	if err := createPrivateDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}
