// Package platform is what this machine supplies to the core that a settings
// file cannot name: an adapter that starts a process.
//
// Every entry point of this application begins from the configuration here, so
// a window and the command line reach the same machine.
package platform

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetConfig is the core's configuration with what this machine supplies already in
// it. What a settings file says is read over it.
func GetConfig() container.Config {
	return container.Config{
		AgentProofreader: openProofreader,
		PluginHost:       container.NewPluginHost("desktop.editor", getPluginsDirs()...),
	}
}

func getPluginsDirs() []string {
	var dirs []string
	seen := make(map[string]bool)
	addDir := func(dir string) {
		if dir == "" {
			return
		}
		clean := filepath.Clean(dir)
		if abs, err := filepath.Abs(clean); err == nil {
			clean = abs
		}
		if !seen[clean] {
			seen[clean] = true
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

	addDir("modules/plugins")
	addDir("plugins")

	configDir, err := os.UserConfigDir()
	if err == nil {
		addDir(filepath.Join(configDir, "numen", "plugins"))
	}

	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		addDir(filepath.Join(localAppData, "Programs", "Numen", "plugins"))
	}

	return dirs
}

// openProofreader opens a proofreading profile that reaches the command line
// the person already has installed.
func openProofreader(spec container.ProofreaderSpec) (port.Proofreader, error) {
	return &claudecode.Proofreader{
		Command:  spec.Command,
		Model:    spec.Model,
		InFlight: spec.InFlight,
	}, nil
}
