package manifestjson

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ListPackageIssues lists what the plugin folder dir lacks for its manifest: the backend entrypoint
// or its source backend/main.go, the frontend bundle dist/index.js or its source frontend/index.ts,
// and the stylesheets of the themes it declares.
func ListPackageIssues(m manifest.Manifest, dir string) []string {
	var issues []string
	issues = append(issues, listBackendFileIssues(m.Backend, dir)...)
	issues = append(issues, listFrontendFileIssues(m.HasFrontend(), dir)...)
	issues = append(issues, listThemeFileIssues(m.Contributes.Themes, dir)...)
	return issues
}

// ListShippedPaths lists the files and folders of a plugin folder a package of the plugin holds,
// besides the stylesheets of its themes.
func ListShippedPaths() []string { return []string{manifest.FileName, "bin", "dist"} }

func listThemeFileIssues(themes []manifest.Theme, dir string) []string {
	var issues []string
	for _, theme := range themes {
		if _, err := ReadThemeFile(dir, theme.File); err != nil {
			issues = append(issues, fmt.Sprintf("theme %q: %v", theme.ID, err))
		}
	}
	return issues
}

func listBackendFileIssues(b manifest.Backend, dir string) []string {
	if strings.TrimSpace(b.Entrypoint) == "" {
		return nil
	}
	if hasAnyFile(filepath.Join(dir, b.Entrypoint), filepath.Join(dir, "backend", "main.go")) {
		return nil
	}
	return []string{fmt.Sprintf("backend entrypoint %q does not exist and no backend/main.go source found", b.Entrypoint)}
}

func listFrontendFileIssues(hasFrontend bool, dir string) []string {
	if !hasFrontend {
		return nil
	}
	if hasAnyFile(filepath.Join(dir, "dist", "index.js"), filepath.Join(dir, "frontend", "index.ts")) {
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
