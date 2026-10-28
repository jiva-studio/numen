package plugindir

import (
	"fmt"
	"os"
	"path/filepath"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// DiscoveredPlugin is a validated plugin located on disk.
type DiscoveredPlugin = pluginport.DiscoveredPlugin

// DiscoveryProblem describes a plugin directory that could not be registered.
type DiscoveryProblem struct {
	Dir string
	Err error
}

// DiscoverPlugins scans directories and loads every valid plugin manifest found; the rest are reported as problems.
func DiscoverPlugins(dirs []string) ([]DiscoveredPlugin, []DiscoveryProblem) {
	observed := make(map[string]string)
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

		plugins, dirProblems := scanDirectory(cleanDir, observed)
		found = append(found, plugins...)
		problems = append(problems, dirProblems...)
	}
	return found, problems
}

func scanDirectory(dir string, visited map[string]string) ([]DiscoveredPlugin, []DiscoveryProblem) {
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
		p, ok, err := scanPluginDir(pluginDir, visited)
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

func scanPluginDir(pluginDir string, seenCount map[string]string) (DiscoveredPlugin, bool, error) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	m, err := readManifestPath(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return EmptyDiscoveredPlugin(), false, nil
		}
		return EmptyDiscoveredPlugin(), false, fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}

	cleanPluginDir := filepath.Clean(pluginDir)
	if abs, err := filepath.Abs(cleanPluginDir); err == nil {
		cleanPluginDir = abs
	}

	if existingDir, ok := seenCount[m.ID]; ok {
		if existingDir == cleanPluginDir {
			return EmptyDiscoveredPlugin(), false, nil
		}
		return EmptyDiscoveredPlugin(), false, fmt.Errorf("duplicate plugin id %q in %s and %s", m.ID, existingDir, cleanPluginDir)
	}
	seenCount[m.ID] = cleanPluginDir
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

func EmptyDiscoveredPlugin() DiscoveredPlugin {
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
