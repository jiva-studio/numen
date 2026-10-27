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
	"github.com/jiva-studio/numen/libs/core/container"
)

var isDev bool

func init() {
	if os.Getenv("NUMEN_DEV") == "1" || os.Getenv("NUMEN_DEV") == "true" {
		isDev = true
	}
}

// IsDev reports whether plugins are discovered from development directories.
func IsDev() bool {
	return isDev
}

// SetDev sets whether plugins are discovered from development directories.
func SetDev(v bool) {
	isDev = v
}

// The windows a plugin names as the places it works in.
const (
	// TargetEditor is the window a person writes in.
	TargetEditor = "desktop.editor"
	// TargetFlashcards is the window a person runs their cards in.
	TargetFlashcards = "desktop.flashcards"
)

// GetConfig is the core's configuration with what this machine supplies already in
// it. What a settings file says is read over it.
func GetConfig() container.Config { return GetConfigFor(TargetEditor) }

// GetConfigFor is GetConfig for the window of a target, whose plugin host runs the plugins that name
// that target.
func GetConfigFor(target string) container.Config {
	cfg := container.Config{
		PluginHost: newPluginHost(target, getPluginsDirs()...),
		KeyStore:   keychain.Store{},
	}
	// A machine with no data folder leaves the path empty, and the core refuses to start.
	if path, err := GetIndexPath(); err == nil {
		cfg.IndexPath = path
	}
	return cfg
}

func getPluginsDirs() []string {
	var dirs []string
	observed := make(map[string]bool)
	addDir := func(dir string) {
		if dir == "" {
			return
		}
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
		addDir(filepath.Join(execDir, "modules", "plugins"))
		addDir(filepath.Join(execDir, "..", "PlugIns"))
		addDir(filepath.Join(execDir, "..", "Resources", "plugins"))
		addDir(filepath.Join(execDir, "..", "lib", "numen", "plugins"))
		addDir(filepath.Join(execDir, "..", "share", "numen", "plugins"))
	}

	if IsDev() {
		addDir("modules/plugins")
		addDir("plugins")
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
