package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/jiva-studio/numen/libs/atomicfile"
)

func (c *Cache[V]) cachePath() (string, error) {
	// gomutants:disable-next-line reason="shouldUseDiskLocked returns false for a cache without a name, so the callers that reach the disk have a name and this guard repeats that check"
	if c.name == "" {
		return "", errors.New("cache name is empty")
	}
	dir, err := GetPluginDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, c.name+".json"), nil
}

const (
	// EnvCacheDir names the variable the host sets to the cache directory of the plugin.
	EnvCacheDir = "NUMEN_PLUGIN_CACHE_DIR"

	// EnvDataDir names the variable the host sets to the data directory of the plugin.
	EnvDataDir = "NUMEN_PLUGIN_DATA_DIR"

	// EnvSharedDir names the variable the host sets to the directory all plugins share.
	EnvSharedDir = "NUMEN_SHARED_CACHE_DIR"

	// EnvSharedDataDir names the variable the host sets to the directory all plugins share for what they cannot compute again.
	EnvSharedDataDir = "NUMEN_SHARED_DATA_DIR"
)

// GetPluginDir returns the cache directory the host granted the plugin. The host owns the directory:
// it creates it, keeps it apart from every other plugin, and removes it when the plugin is
// uninstalled or updated. What the plugin keeps there can be recomputed.
func GetPluginDir() (string, error) { return getGrantedDir(EnvCacheDir) }

// GetDataDir returns the data directory the host granted the plugin. What the plugin keeps there
// stays across updates of the plugin.
func GetDataDir() (string, error) { return getGrantedDir(EnvDataDir) }

// GetSharedDir returns the directory the host shares between all plugins for what can be computed again. Plugins write there only
// content that is named by its digest, write it to a temporary file in the same directory and rename it
// once its digest is verified, and never delete what is there: the host owns its lifetime.
func GetSharedDir() (string, error) { return getGrantedDir(EnvSharedDir) }

// GetSharedDataDir returns the directory the host shares between all plugins for what is downloaded or made
// and cannot be computed again, such as models. The same rules hold as for GetSharedDir. The OS never
// purges it as it may a cache.
func GetSharedDataDir() (string, error) { return getGrantedDir(EnvSharedDataDir) }

// GetSharedDataDirOrPluginDataDir returns the shared data directory, and the plugin's own data directory
// where the host granted no shared one.
func GetSharedDataDirOrPluginDataDir() (string, error) {
	if dir, err := GetSharedDataDir(); err == nil {
		return dir, nil
	}
	return GetDataDir()
}

// GetSharedDirOrPluginDir returns the shared directory, and the plugin's own cache directory where the
// host granted no shared one.
func GetSharedDirOrPluginDir() (string, error) {
	if dir, err := GetSharedDir(); err == nil {
		return dir, nil
	}
	return GetPluginDir()
}

func getGrantedDir(name string) (string, error) {
	dir := os.Getenv(name)
	if dir == "" {
		return "", fmt.Errorf("the host granted no directory: %s is not set", name)
	}
	return dir, nil
}

func (c *Cache[V]) readDiskEntry(key string) (V, bool) {
	var zero V
	path, err := c.cachePath()
	// gomutants:disable-next-line reason="without the return the empty path makes ReadFile fail and take the same return a few lines below"
	if err != nil {
		return zero, false
	}
	c.diskMu.Lock()
	data, err := os.ReadFile(path)
	c.diskMu.Unlock()
	// gomutants:disable-next-line reason="without the return the empty data makes the decode below fail and take the same return"
	if err != nil {
		return zero, false
	}
	var all map[string]V
	// gomutants:disable-next-line reason="without the return the lookup below reads a map that is empty when the input is not valid JSON and gives the same zero and false"
	if json.Unmarshal(data, &all) != nil {
		return zero, false
	}
	val, found := all[key]
	if !found || !isPersistable(val) {
		return zero, false
	}
	return val, true
}

func (c *Cache[V]) writeDiskEntry(key string, val V) {
	if !isPersistable(val) {
		return
	}
	// A value the disk does not take stays in memory; the disk only lets a later process start with it.
	_ = c.storeDiskEntry(key, val)
}

func (c *Cache[V]) storeDiskEntry(key string, val V) error {
	path, err := c.cachePath()
	if err != nil {
		return err
	}
	c.diskMu.Lock()
	defer c.diskMu.Unlock()

	all := make(map[string]V)
	if existing, readErr := os.ReadFile(path); readErr == nil {
		if json.Unmarshal(existing, &all) != nil {
			all = make(map[string]V)
		}
	}
	all[key] = val
	data, err := json.Marshal(all)
	// gomutants:disable-next-line reason="json.Marshal of a map keyed by strings fails only for a value type that JSON cannot encode, which no test stores"
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return atomicfile.WriteFile(path, bytes.NewReader(data), 0o600)
}

func isPersistable[V any](val V) bool {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() {
		return false
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map {
		return rv.Len() > 0
	}
	return !rv.IsZero()
}
