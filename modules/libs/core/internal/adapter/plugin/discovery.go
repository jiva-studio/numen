package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// DiscoveredPlugin is a validated plugin located on disk.
type DiscoveredPlugin = port.DiscoveredPlugin

// DiscoveryProblem describes a plugin directory that could not be registered.
type DiscoveryProblem struct {
	Dir string
	Err error
}

// DiscoverPlugins scans directories and loads every valid plugin manifest found; the rest are reported as problems.
func DiscoverPlugins(dirs []string) ([]DiscoveredPlugin, []DiscoveryProblem) {
	seen := make(map[string]string)
	seenDirs := make(map[string]bool)
	var found []DiscoveredPlugin
	var problems []DiscoveryProblem

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		cleanDir := filepath.Clean(dir)
		if abs, err := filepath.Abs(cleanDir); err == nil {
			cleanDir = abs
		}
		if seenDirs[cleanDir] {
			continue
		}
		seenDirs[cleanDir] = true

		plugins, dirProblems := scanDirectory(cleanDir, seen)
		found = append(found, plugins...)
		problems = append(problems, dirProblems...)
	}
	return found, problems
}

func scanDirectory(dir string, seen map[string]string) ([]DiscoveredPlugin, []DiscoveryProblem) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []DiscoveryProblem{{Dir: dir, Err: fmt.Errorf("read plugin dir %s: %w", dir, err)}}
	}

	var found []DiscoveredPlugin
	var problems []DiscoveryProblem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginDir := filepath.Join(dir, entry.Name())
		p, ok, err := scanPluginDir(pluginDir, seen)
		if err != nil {
			problems = append(problems, DiscoveryProblem{Dir: pluginDir, Err: err})
			continue
		}
		if ok {
			found = append(found, p)
		}
	}
	return found, problems
}

func scanPluginDir(pluginDir string, seen map[string]string) (DiscoveredPlugin, bool, error) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	m, err := readManifestPath(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyDiscoveredPlugin(), false, nil
		}
		return emptyDiscoveredPlugin(), false, fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}

	cleanPluginDir := filepath.Clean(pluginDir)
	if abs, err := filepath.Abs(cleanPluginDir); err == nil {
		cleanPluginDir = abs
	}

	if existingDir, ok := seen[m.ID]; ok {
		if existingDir == cleanPluginDir {
			return emptyDiscoveredPlugin(), false, nil
		}
		return emptyDiscoveredPlugin(), false, fmt.Errorf("duplicate plugin id %q in %s and %s", m.ID, existingDir, cleanPluginDir)
	}
	seen[m.ID] = cleanPluginDir
	return DiscoveredPlugin{
		Dir:      cleanPluginDir,
		Manifest: m,
	}, true, nil
}

func readManifestPath(path string) (Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, err
	}
	defer file.Close()
	return ReadManifest(file)
}

func emptyDiscoveredPlugin() DiscoveredPlugin {
	return DiscoveredPlugin{
		Dir: "",
		Manifest: Manifest{
			ID:               "",
			Name:             "",
			Version:          "",
			Description:      "",
			Capabilities:     nil,
			Targets:          nil,
			ActivationEvents: nil,
			Permissions:      nil,
			Lifecycle:        Lifecycle{IdleTimeoutSeconds: 0, KeepAliveWhileVisible: false},
			Backend:          Backend{Type: "", Entrypoint: "", ServiceName: ""},
			Frontend:         Frontend{Entrypoint: "", Styles: ""},
			Settings:         nil,
		},
	}
}

// FilterPluginsByTarget returns only plugins that can run in the target workspace.
func FilterPluginsByTarget(plugins []DiscoveredPlugin, target string) []DiscoveredPlugin {
	var matched []DiscoveredPlugin
	for _, p := range plugins {
		if p.Manifest.CanTarget(target) {
			matched = append(matched, p)
		}
	}
	return matched
}

// FindPluginByCapability returns the first plugin providing the specified capability.
func FindPluginByCapability(plugins []DiscoveredPlugin, capability string) (DiscoveredPlugin, bool) {
	for _, p := range plugins {
		if p.Manifest.HasCapability(capability) {
			return p, true
		}
	}
	return emptyDiscoveredPlugin(), false
}
