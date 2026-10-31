package manifestjson

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// DiscoveryProblem describes a plugin directory that could not be registered.
type DiscoveryProblem struct {
	Dir string
	Err error
}

// ReadFile reads and validates the manifest at the path.
func ReadFile(path string) (m manifest.Manifest, err error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest.Manifest{}, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return Read(file)
}

// Discover scans directories and loads every valid plugin manifest found; the rest are reported as problems.
func Discover(dirs []string) ([]manifest.Discovered, []DiscoveryProblem) {
	observed := make(map[string]string)
	seenDirs := make(map[string]bool)
	var found []manifest.Discovered
	var problems []DiscoveryProblem

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		cleanDir := absolutePath(dir)
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

func absolutePath(path string) string {
	clean := filepath.Clean(path)
	if abs, err := filepath.Abs(clean); err == nil {
		return abs
	}
	return clean
}

func scanDirectory(dir string, visited map[string]string) ([]manifest.Discovered, []DiscoveryProblem) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []DiscoveryProblem{{Dir: dir, Err: fmt.Errorf("read plugin dir %s: %w", dir, err)}}
	}

	var found []manifest.Discovered
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

func scanPluginDir(pluginDir string, seenCount map[string]string) (manifest.Discovered, bool, error) {
	manifestPath := filepath.Join(pluginDir, manifest.FileName)
	m, err := ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return manifest.Discovered{}, false, nil
		}
		return manifest.Discovered{}, false, fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}

	cleanPluginDir := absolutePath(pluginDir)
	if existingDir, ok := seenCount[m.ID]; ok {
		if existingDir == cleanPluginDir {
			return manifest.Discovered{}, false, nil
		}
		return manifest.Discovered{}, false, fmt.Errorf("duplicate plugin id %q in %s and %s", m.ID, existingDir, cleanPluginDir)
	}
	seenCount[m.ID] = cleanPluginDir
	return manifest.Discovered{Dir: cleanPluginDir, Manifest: m}, true, nil
}
