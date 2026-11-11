package plugin_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func getVar(t *testing.T, vars []string, name string) string {
	t.Helper()
	for _, entry := range vars {
		if value, found := strings.CutPrefix(entry, name+"="); found {
			return value
		}
	}
	t.Fatalf("environment %v has no %s", vars, name)
	return ""
}

func checkPrivateDir(t *testing.T, dir string) {
	t.Helper()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory %s: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Errorf("directory %s has mode %o, want 700", dir, info.Mode().Perm())
	}
}

func newStorageHost(t *testing.T) (*container.PluginRoles, string, string) {
	t.Helper()
	host := newDiscoveredHost(t)
	cacheRoot, dataRoot := t.TempDir(), t.TempDir()
	host.Storage.SetStorageRoots(cacheRoot, dataRoot)
	setPlugin(host, "org.example.other", manifest.Discovered{Manifest: manifest.Manifest{ID: "org.example.other", Version: "2.0.0"}})
	return host, cacheRoot, dataRoot
}

func TestThePluginEnvironmentNamesItsCacheAndDataDirectories(t *testing.T) {
	host, cacheRoot, dataRoot := newStorageHost(t)

	vars, err := host.Processes.GetLaunchVars(t.Context(), "org.example.sample", nil, host.Mode.GetVaultPath(), 5, "t")
	if err != nil {
		t.Fatal(err)
	}

	cacheDir := getVar(t, vars, "NUMEN_PLUGIN_CACHE_DIR")
	dataDir := getVar(t, vars, "NUMEN_PLUGIN_DATA_DIR")
	if want := filepath.Join(cacheRoot, "plugins", "org.example.sample"); cacheDir != want {
		t.Errorf("cache dir %s, want %s", cacheDir, want)
	}
	if want := filepath.Join(dataRoot, "plugins", "org.example.sample"); dataDir != want {
		t.Errorf("data dir %s, want %s", dataDir, want)
	}
	checkPrivateDir(t, cacheDir)
	checkPrivateDir(t, dataDir)
	checkPrivateDir(t, getVar(t, vars, "NUMEN_SHARED_CACHE_DIR"))
}

func TestTwoPluginsGetDifferentDirectories(t *testing.T) {
	host, _, _ := newStorageHost(t)

	first, err := host.Processes.GetLaunchVars(t.Context(), "org.example.sample", nil, host.Mode.GetVaultPath(), 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	second, err := host.Processes.GetLaunchVars(t.Context(), "org.example.other", nil, host.Mode.GetVaultPath(), 6, "t")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"NUMEN_PLUGIN_CACHE_DIR", "NUMEN_PLUGIN_DATA_DIR"} {
		if getVar(t, first, name) == getVar(t, second, name) {
			t.Errorf("both plugins got %s %s", name, getVar(t, first, name))
		}
	}
}

func TestAHostWithoutStorageRootsGrantsNoDirectory(t *testing.T) {
	host := newDiscoveredHost(t)

	vars, err := host.Processes.GetLaunchVars(t.Context(), "org.example.sample", nil, host.Mode.GetVaultPath(), 5, "t")
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range vars {
		if strings.HasPrefix(entry, "NUMEN_PLUGIN_CACHE_DIR=") || strings.HasPrefix(entry, "NUMEN_PLUGIN_DATA_DIR=") {
			t.Errorf("environment names a directory the host has no root for: %s", entry)
		}
	}
}

func TestTheCacheDirectoryIsEmptiedWhenThePluginVersionChanges(t *testing.T) {
	host, _, _ := newStorageHost(t)
	launch := func() (cacheDir, dataDir string) {
		vars, err := host.Processes.GetLaunchVars(t.Context(), "org.example.sample", nil, host.Mode.GetVaultPath(), 5, "t")
		if err != nil {
			t.Fatal(err)
		}
		return getVar(t, vars, "NUMEN_PLUGIN_CACHE_DIR"), getVar(t, vars, "NUMEN_PLUGIN_DATA_DIR")
	}
	cacheDir, dataDir := launch()
	cached := filepath.Join(cacheDir, "compiled")
	notesDir := filepath.Join(dataDir, "notes")
	for _, path := range []string{cached, notesDir} {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	launch()
	if _, err := os.Stat(cached); err != nil {
		t.Errorf("the cache of the same version was removed: %v", err)
	}
	p := getRawPlugin(host, "org.example.sample")
	p.Manifest.Version = "1.1.0"
	setPlugin(host, "org.example.sample", p)
	launch()

	if _, err := os.Stat(cached); !os.IsNotExist(err) {
		t.Errorf("the cache of the old version stays: %v", err)
	}
	if _, err := os.Stat(notesDir); err != nil {
		t.Errorf("the data directory lost a file: %v", err)
	}
	checkPrivateDir(t, cacheDir)
}

func TestTheCacheDirectoryIsRemovedWhenThePluginIsRemovedFromTheDisk(t *testing.T) {
	host, _, _ := newStorageHost(t)
	vars, err := host.Processes.GetLaunchVars(t.Context(), "org.example.sample", nil, host.Mode.GetVaultPath(), 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	cacheDir := getVar(t, vars, "NUMEN_PLUGIN_CACHE_DIR")
	dataDir := getVar(t, vars, "NUMEN_PLUGIN_DATA_DIR")
	pluginDir := getRawPlugin(host, "org.example.sample").Dir
	if err := os.RemoveAll(pluginDir); err != nil {
		t.Fatal(err)
	}

	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Errorf("the cache directory of an uninstalled plugin stays: %v", err)
	}
	if _, err := os.Stat(dataDir); err != nil {
		t.Errorf("the data directory of an uninstalled plugin was removed: %v", err)
	}
}

func TestAPluginIdThatIsNotAFolderNameGetsNoDirectory(t *testing.T) {
	host, cacheRoot, _ := newStorageHost(t)

	vars, err := host.Storage.GetVars("../escape")
	if err == nil || len(vars) != 0 {
		t.Errorf("GetVars() = %v, %v, want an error", vars, err)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(cacheRoot), "escape")); statErr == nil {
		t.Error("a directory was created outside the root")
	}
}

func TestAllPluginsAreGivenTheSameSharedDirectoriesThatSurviveAVersionChange(t *testing.T) {
	t.Setenv("NUMEN_MODELS_DIR", "")
	host, cacheRoot, dataRoot := newStorageHost(t)

	first, err := host.Processes.GetLaunchVars(t.Context(), "org.example.sample", nil, host.Mode.GetVaultPath(), 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	second, err := host.Processes.GetLaunchVars(t.Context(), "org.example.other", nil, host.Mode.GetVaultPath(), 6, "t")
	if err != nil {
		t.Fatal(err)
	}

	sharedCache := getVar(t, first, "NUMEN_SHARED_CACHE_DIR")
	sharedData := getVar(t, first, "NUMEN_SHARED_DATA_DIR")
	if sharedCache != getVar(t, second, "NUMEN_SHARED_CACHE_DIR") || sharedData != getVar(t, second, "NUMEN_SHARED_DATA_DIR") {
		t.Errorf("the plugins got different shared directories")
	}
	if want := filepath.Join(cacheRoot, "shared"); sharedCache != want || host.Storage.GetSharedDir() != want {
		t.Errorf("shared cache directory %s and %s, want %s", sharedCache, host.Storage.GetSharedDir(), want)
	}
	if want := filepath.Join(dataRoot, "shared"); sharedData != want || host.Storage.GetSharedDataDir() != want {
		t.Errorf("shared data directory %s and %s, want %s", sharedData, host.Storage.GetSharedDataDir(), want)
	}
	models := filepath.Join(sharedData, "models")
	if host.Storage.GetModelsDir() != models || getVar(t, first, "NUMEN_MODELS_DIR") != models {
		t.Errorf("models directory %s and %s, want %s", host.Storage.GetModelsDir(), getVar(t, first, "NUMEN_MODELS_DIR"), models)
	}
	checkPrivateDir(t, sharedCache)
	checkPrivateDir(t, sharedData)
	for _, name := range []string{"NUMEN_PLUGIN_CACHE_DIR", "NUMEN_PLUGIN_DATA_DIR"} {
		if got := getVar(t, first, name); got == sharedCache || got == sharedData {
			t.Errorf("%s is a shared directory", name)
		}
	}

	blobs := []string{filepath.Join(sharedCache, "blob"), filepath.Join(sharedData, "blob")}
	for _, blob := range blobs {
		if err := os.WriteFile(blob, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p := getRawPlugin(host, "org.example.sample")
	p.Manifest.Version = "9.0.0"
	setPlugin(host, "org.example.sample", p)
	if _, err := host.Processes.GetLaunchVars(t.Context(), "org.example.sample", nil, host.Mode.GetVaultPath(), 5, "t"); err != nil {
		t.Fatal(err)
	}
	for _, blob := range blobs {
		if _, err := os.Stat(blob); err != nil {
			t.Errorf("a version change removed what the plugins share: %v", err)
		}
	}
}

func TestTheUserCanMoveTheModelsFolder(t *testing.T) {
	moved := t.TempDir()
	t.Setenv("NUMEN_MODELS_DIR", moved)
	host, _, _ := newStorageHost(t)

	vars, err := host.Processes.GetLaunchVars(t.Context(), "org.example.sample", nil, host.Mode.GetVaultPath(), 5, "t")
	if err != nil {
		t.Fatal(err)
	}

	if host.Storage.GetModelsDir() != moved || getVar(t, vars, "NUMEN_MODELS_DIR") != moved {
		t.Errorf("models directory %s and %s, want %s", host.Storage.GetModelsDir(), getVar(t, vars, "NUMEN_MODELS_DIR"), moved)
	}
}
