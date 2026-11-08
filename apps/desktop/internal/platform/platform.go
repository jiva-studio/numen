// Package platform is what this machine supplies to the core that a settings
// file cannot name: the folders its plugins are found in and the keychain keys are kept in.
//
// Every entry point of this application begins from the configuration here, so
// a window and the command line reach the same machine.
package platform

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/apps/desktop/internal/keychain"
	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/indexsource"
	"github.com/jiva-studio/numen/libs/plugin-catalog/packagecache"
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
	cfg := container.NewConfig(container.Config{
		PluginHost:    newPluginHost(target, dirs...),
		PluginDirs:    dirs,
		PluginCatalog: getCatalogConfig(target, dirs),
		KeyStore:      getKeyStore(),
		Resources:     container.NewResources(),
	})
	// A machine with no data folder leaves the path empty, and the core fails to start.
	if path, err := GetIndexPath(); err == nil {
		cfg.IndexPath = path
	}
	return cfg
}

func getKeyStore() *keychain.Store {
	path, err := settings.Path()
	if err != nil {
		return &keychain.Store{}
	}
	return keychain.NewStore(filepath.Dir(path), container.Config{}.GetWait())
}

func getDevPluginsDirs() []string { return []string{"plugins"} }

func getCatalogConfig(target string, pluginDirs []string) catalog.Config {
	config := catalog.Config{Target: target, Sources: []catalog.Source{getIndexSource(pluginDirs)}, InstallDir: "", CacheDir: ""}
	if configDir, err := os.UserConfigDir(); err == nil {
		config.InstallDir = filepath.Join(configDir, "numen", "plugins")
	}
	if cacheDir, err := packagecache.DefaultDir(); err == nil {
		config.CacheDir = cacheDir
	}
	return config
}

func getIndexSource(pluginDirs []string) *indexsource.Source {
	// gomutants:disable-next-line
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
		// gomutants:disable-next-line
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
	// gomutants:disable-next-line
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
