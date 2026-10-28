package plugincli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/jiva-studio/numen/libs/core/container"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
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

	var m pluginport.PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return []string{fmt.Sprintf("invalid json in manifest: %v", err)}, nil
	}

	var issues []string
	for _, issue := range m.ListIssues() {
		issues = append(issues, issue.Error())
	}
	issues = append(issues, listBackendFileIssues(m.Backend, pluginDir)...)
	issues = append(issues, listFrontendFileIssues(m.Frontend, pluginDir)...)
	return issues, nil
}

func resolveManifestPath(targetPath string) (manifestPath, pluginDir string, err error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return "", "", fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		pluginDir = targetPath
		manifestPath = filepath.Join(targetPath, "manifest.json")
	} else {
		pluginDir = filepath.Dir(targetPath)
		manifestPath = targetPath
	}
	return manifestPath, pluginDir, nil
}

func listBackendFileIssues(b pluginport.PluginBackend, pluginDir string) []string {
	if strings.TrimSpace(b.Entrypoint) == "" {
		return nil
	}
	if hasAnyFile(filepath.Join(pluginDir, b.Entrypoint), filepath.Join(pluginDir, "backend", "main.go")) {
		return nil
	}
	return []string{fmt.Sprintf("backend entrypoint %q does not exist and no backend/main.go source found", b.Entrypoint)}
}

func listFrontendFileIssues(f pluginport.PluginFrontend, pluginDir string) []string {
	if strings.TrimSpace(f.Entrypoint) == "" {
		return nil
	}
	if hasAnyFile(filepath.Join(pluginDir, f.Entrypoint), filepath.Join(pluginDir, "frontend", "index.ts")) {
		return nil
	}
	return []string{fmt.Sprintf("frontend entrypoint %q does not exist and no frontend/index.ts source found", f.Entrypoint)}
}

func hasAnyFile(paths ...string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
