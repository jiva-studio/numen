package plugincli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// Validate inspects a plugin manifest and entrypoints on disk.
func Validate(targetPath string) ([]string, error) {
	manifestPath, pluginDir, err := resolveManifestPath(targetPath)
	if err != nil {
		return nil, err
	}
	_, issues, err := readManifest(manifestPath, pluginDir)
	return issues, err
}

func readManifest(manifestPath, pluginDir string) (manifest.Manifest, []string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return manifest.Manifest{}, nil, fmt.Errorf("read manifest at %s: %w", manifestPath, err)
	}

	m, err := manifestjson.Decode(data)
	if err != nil {
		return manifest.Manifest{}, []string{fmt.Sprintf("invalid json in manifest: %v", err)}, nil
	}

	var issues []string
	for _, issue := range m.ListIssues() {
		issues = append(issues, issue.Error())
	}
	return m, append(issues, manifestjson.ListPackageIssues(m, pluginDir)...), nil
}

func resolveManifestPath(targetPath string) (manifestPath, pluginDir string, err error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return "", "", fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		pluginDir = targetPath
		manifestPath = filepath.Join(targetPath, manifest.FileName)
	} else {
		pluginDir = filepath.Dir(targetPath)
		manifestPath = targetPath
	}
	return manifestPath, pluginDir, nil
}
