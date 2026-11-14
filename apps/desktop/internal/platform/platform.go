// Package platform is what this machine supplies to the core that a settings
// file cannot name: the folders its plugins are found in and the keychain keys are kept in.
//
// Every entry point of this application begins from the configuration here, so
// a window and the command line reach the same machine.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/jiva-studio/numen/apps/desktop/internal/keychain"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/indexsource"
	"github.com/jiva-studio/numen/libs/plugin/catalog/localsource"
	"github.com/jiva-studio/numen/libs/plugin/catalog/packagecache"
)

// IsDev reports whether plugins are discovered from development directories: NUMEN_DEV is 1 or true.
func IsDev() bool {
	value := os.Getenv("NUMEN_DEV")
	return value == "1" || value == "true"
}

// The windows a plugin names as the places it works in.
const (
	// TargetEditor is the editor window.
	TargetEditor = "desktop.editor"
	// TargetFlashcards is the flashcards window.
	TargetFlashcards = "desktop.flashcards"
)

// GetConfig returns the core's configuration with the values this machine
// supplies. Values from a settings file are applied over it.
func GetConfig() container.Config { return GetConfigFor(TargetEditor) }

// GetConfigFor is GetConfig for the window of a target, whose plugin host runs the plugins that name
// that target.
func GetConfigFor(target string) container.Config {
	dirs := getPluginsDirs()
	base := container.Config{KeyStore: getKeyStore(), Resources: container.NewResources()}
	host, discoverErr := container.NewPluginHost(base, reportPluginHost, target, dirs...)
	if discoverErr != nil {
		fmt.Fprintf(os.Stderr, "numen: discover plugins: %v\n", discoverErr)
	}
	cacheRoot, dataRoot := getStorageRoots(runtime.GOOS, os.Getenv, os.UserCacheDir, os.UserHomeDir)
	host.SetStorage(cacheRoot, dataRoot)
	base.PluginHost, base.PluginCatalog = host, getCatalogConfig(target, dirs, host)
	cfg := container.NewConfig(base)
	// A machine with no data folder leaves the path empty, and the core fails to start.
	if path, err := GetIndexPath(); err == nil {
		cfg.IndexPath = path
	}
	return cfg
}

func reportPluginHost(err error) {
	fmt.Fprintf(os.Stderr, "numen: plugin host: %v\n", err)
}

func getKeyStore() *keychain.Store {
	path, err := settings.Path()
	if err != nil {
		return &keychain.Store{}
	}
	return keychain.NewSystemStore(filepath.Dir(path))
}

func getDevPluginsDirs() []string { return []string{"plugins"} }

func getCatalogConfig(target string, pluginDirs []string, host *container.PluginHost) catalog.Config {
	local := localsource.New(slices.Clone(pluginDirs), localsource.SettingsFunc(host.Runtime.IsPluginEnabled))
	config := catalog.Config{Target: target, Sources: []catalog.Source{local, getIndexSource(pluginDirs)}, InstallDir: "", CacheDir: ""}
	if configDir, err := os.UserConfigDir(); err == nil {
		config.InstallDir = filepath.Join(configDir, "numen", "plugins")
	}
	if cacheDir, err := packagecache.DefaultDir(); err == nil {
		config.CacheDir = cacheDir
	}
	return config
}

func getIndexSource(pluginDirs []string) *indexsource.Source {
	// gomutants:disable-next-line reason="the capacity of the made value does not change what the function returns"
	indexPaths := make([]string, 0, len(pluginDirs))
	for _, dir := range pluginDirs {
		indexPaths = append(indexPaths, filepath.Join(dir, indexsource.FileName))
	}
	var manifestDirs []string
	if IsDev() {
		manifestDirs = getDevPluginsDirs()
	}
	return indexsource.New(indexPaths, manifestDirs)
}

func getPluginsDirs() []string {
	var dirs []string
	observed := make(map[string]bool)
	addDir := func(dir string) {
		clean := filepath.Clean(dir)
		// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
		if abs, err := filepath.Abs(clean); err == nil {
			clean = abs
		}
		if !observed[clean] {
			observed[clean] = true
			dirs = append(dirs, clean)
		}
	}

	if custom := os.Getenv("NUMEN_PLUGINS_PATH"); custom != "" {
		addDir(custom)
	}

	execPath, err := os.Executable()
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err == nil {
		execDir := filepath.Dir(execPath)
		addDir(filepath.Join(execDir, "plugins"))
		addDir(filepath.Join(execDir, "..", "PlugIns"))
		addDir(filepath.Join(execDir, "..", "Resources", "plugins"))
		addDir(filepath.Join(execDir, "..", "lib", "numen", "plugins"))
		addDir(filepath.Join(execDir, "..", "share", "numen", "plugins"))
	}

	if IsDev() {
		for _, dir := range getDevPluginsDirs() {
			addDir(dir)
		}
	}

	configDir, err := os.UserConfigDir()
	if err == nil {
		addDir(filepath.Join(configDir, "numen", "plugins"))
	}

	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		addDir(filepath.Join(localAppData, "Programs", "Numen", "plugins"))
	}

	return dirs
}
