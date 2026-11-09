package plugincli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// Validate inspects a plugin manifest and entrypoints on disk.
func Validate(targetPath string) ([]string, error) {
	manifestPath, pluginDir, err := resolveManifestPath(targetPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest at %s: %w", manifestPath, err)
	}

	m, err := manifestjson.Decode(data)
	if err != nil {
		return []string{fmt.Sprintf("invalid json in manifest: %v", err)}, nil
	}

	var issues []string
	for _, issue := range m.ListIssues() {
		issues = append(issues, issue.Error())
	}
	issues = append(issues, listBackendFileIssues(m.Backend, pluginDir)...)
	issues = append(issues, listFrontendFileIssues(m.HasFrontend(), pluginDir)...)
	issues = append(issues, listThemeFileIssues(m.Contributes.Themes, pluginDir)...)
	return issues, nil
}

func listThemeFileIssues(themes []manifest.Theme, pluginDir string) []string {
	var issues []string
	for _, theme := range themes {
		if _, err := manifestjson.ReadThemeFile(pluginDir, theme.File); err != nil {
			issues = append(issues, fmt.Sprintf("theme %q: %v", theme.ID, err))
		}
	}
	return issues
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

func listBackendFileIssues(b manifest.Backend, pluginDir string) []string {
	if strings.TrimSpace(b.Entrypoint) == "" {
		return nil
	}
	if hasAnyFile(filepath.Join(pluginDir, b.Entrypoint), filepath.Join(pluginDir, "backend", "main.go")) {
		return nil
	}
	return []string{fmt.Sprintf("backend entrypoint %q does not exist and no backend/main.go source found", b.Entrypoint)}
}

func listFrontendFileIssues(hasFrontend bool, pluginDir string) []string {
	if !hasFrontend {
		return nil
	}
	if hasAnyFile(filepath.Join(pluginDir, "dist", "index.js"), filepath.Join(pluginDir, "frontend", "index.ts")) {
		return nil
	}
	return []string{"frontend bundle dist/index.js does not exist and no frontend/index.ts source found"}
}

func hasAnyFile(paths ...string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
