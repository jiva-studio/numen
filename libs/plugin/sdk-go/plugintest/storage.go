package plugintest

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
)

// GrantStorage gives the test the directories the host gives a plugin process: it sets
// NUMEN_PLUGIN_CACHE_DIR, NUMEN_PLUGIN_DATA_DIR and NUMEN_SHARED_CACHE_DIR and NUMEN_SHARED_DATA_DIR to new temporary directories
// and returns the first two. GrantSharedStorage returns the third.
func GrantStorage(t *testing.T) (cacheDir, dataDir string) {
	t.Helper()
	cacheDir, dataDir = t.TempDir(), t.TempDir()
	t.Setenv(cache.EnvCacheDir, cacheDir)
	t.Setenv(cache.EnvDataDir, dataDir)
	t.Setenv(cache.EnvSharedDir, t.TempDir())
	t.Setenv(cache.EnvSharedDataDir, t.TempDir())
	return cacheDir, dataDir
}

// GrantSharedStorage is GrantStorage that returns the directory shared between plugins.
func GrantSharedStorage(t *testing.T) string {
	t.Helper()
	GrantStorage(t)
	dir, err := cache.GetSharedDir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
