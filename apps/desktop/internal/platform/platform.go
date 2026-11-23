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
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/identityfile"
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

// GetConfig returns the desktop launch paths.
func GetConfig() container.Config {
	var cfg container.Config
	if path, err := GetIndexPath(); err == nil {
		cfg.IndexPath = path
	}
	return cfg
}

// Installation holds the roles built when a desktop application starts.
type Installation struct {
	Execution   *container.PluginExecution
	Preferences *container.PluginPreferences
	Endpoints   *container.PluginEndpoints
	Resources   *container.Resources
}

// NewPluginServices builds the plugin host and shared resources of a desktop window or command.
func NewPluginServices(target string) Installation {
	dirs := getPluginsDirs()
	res := container.NewResources()
	res.KeyStore = getKeyStore()
	res.ChainStore = getChainStore()
	execution, preferences, endpoints, discoverErr := container.NewPluginHost(GetConfig().IndexPath, res, reportPluginHost, target, dirs...)
	reportPluginFailure("discover plugins", discoverErr)
	cacheRoot, dataRoot := getStorageRoots(runtime.GOOS, os.Getenv, os.UserCacheDir, os.UserHomeDir)
	preferences.SetStorage(cacheRoot, dataRoot)
	res.PluginCatalog = getCatalogConfig(target, dirs, execution.Runtime.Listing.IsPluginEnabled)
	return Installation{Execution: execution, Preferences: preferences, Endpoints: endpoints, Resources: res}
}

func reportPluginHost(err error) {
	reportPluginFailure("plugin host", err)
}

func reportPluginFailure(stage string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "numen: %s: %v\n", stage, err)
	}
}

func getChainStore() *identityfile.Chain {
	path, err := settings.Path()
	if err != nil {
		return identityfile.Open(os.TempDir())
	}
	return identityfile.Open(filepath.Dir(path))
}

func getKeyStore() *keychain.Store {
	path, err := settings.Path()
	if err != nil {
		return &keychain.Store{}
	}
	return keychain.NewSystemStore(filepath.Dir(path))
}

func getDevPluginsDirs() []string { return []string{"plugins"} }

func getCatalogConfig(target string, pluginDirs []string, isEnabled localsource.SettingsFunc) catalog.Config {
	local := localsource.New(slices.Clone(pluginDirs), isEnabled)
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
