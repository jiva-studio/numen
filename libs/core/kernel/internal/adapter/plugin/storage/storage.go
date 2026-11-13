package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// versionMarker is the file inside a plugin cache directory that records the version of the plugin
// the directory was filled by.
const versionMarker = ".plugin-version"

// storageRoots are the folders the host grants plugin storage under. An empty root grants none.
type storageRoots struct {
	cache string
	data  string
}

// Storage grants the plugins folders to keep files in: one cache folder and one data folder each,
// and a cache folder and a data folder every plugin shares.
type Storage struct {
	report    func(error)
	versionOf func(pluginID string) string
	mu        sync.RWMutex
	roots     storageRoots
}

// New creates a storage that reports the errors it meets outside a call to report and reads the version
// of a plugin from versionOf.
func New(report func(error), versionOf func(pluginID string) string) *Storage {
	return &Storage{report: report, versionOf: versionOf}
}

// SetStorageRoots sets the folders the plugin cache directories and plugin data directories are
// created under, as <cacheRoot>/plugins/<plugin id> and <dataRoot>/plugins/<plugin id>. The
// application chooses them from the platform's own folders.
func (s *Storage) SetStorageRoots(cacheRoot, dataRoot string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roots = storageRoots{cache: cacheRoot, data: dataRoot}
}

// GetVars returns the variables that name the folders of the plugin, creating the folders.
func (s *Storage) GetVars(pluginID string) ([]string, error) {
	s.mu.RLock()
	roots := s.roots
	s.mu.RUnlock()
	version := s.versionOf(pluginID)
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
		vars = append(vars, "NUMEN_MODELS_DIR="+s.GetModelsDir())
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

func (s *Storage) RemoveCacheDir(pluginID string) {
	s.mu.RLock()
	root := s.roots.cache
	s.mu.RUnlock()
	if root == "" || checkStorageID(pluginID) != nil {
		return
	}
	if err := os.RemoveAll(getPluginStorageDir(root, pluginID)); err != nil {
		s.report(fmt.Errorf("remove cache directory of plugin %s: %w", pluginID, err))
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
func (s *Storage) GetSharedDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.roots.cache == "" {
		return ""
	}
	return filepath.Join(s.roots.cache, sharedFolder)
}

// toolsFolder is the folder inside the cache root where the host keeps the tools each plugin listed.
const toolsFolder = "tools"

// GetToolsDir returns the directory the host keeps the tool listings of the plugins in, or empty where
// the host has no cache root.
func (s *Storage) GetToolsDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.roots.cache == "" {
		return ""
	}
	return filepath.Join(s.roots.cache, toolsFolder)
}

// GetSharedDataDir returns the directory all plugins share for what is downloaded or made and cannot be
// computed again, or empty where the host has no data root. The host never deletes it.
func (s *Storage) GetSharedDataDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.roots.data == "" {
		return ""
	}
	return filepath.Join(s.roots.data, sharedFolder)
}

// GetModelsDir returns the folder where models are kept once for every plugin: NUMEN_MODELS_DIR when the
// user sets it, else models inside the shared data directory. It is empty where the host has no data root.
func (s *Storage) GetModelsDir() string {
	if dir := os.Getenv(envModelsDir); dir != "" {
		return dir
	}
	shared := s.GetSharedDataDir()
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
