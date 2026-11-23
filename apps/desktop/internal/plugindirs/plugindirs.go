// Package plugindirs finds the folders a desktop build looks for plugins in.
package plugindirs

import (
	"os"
	"path/filepath"
)

// IsDev reports whether plugins are discovered from development directories: NUMEN_DEV is 1 or true.
func IsDev() bool {
	value := os.Getenv("NUMEN_DEV")
	return value == "1" || value == "true"
}

// GetDevDirs returns the folders searched only in a development run.
func GetDevDirs() []string { return []string{"plugins"} }

// GetDirs returns every folder plugins are discovered from, without duplicates.
func GetDirs() []string {
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
		for _, dir := range GetDevDirs() {
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
