package plugincli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	validIDPattern      = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)
	validSemverPattern  = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
	allowedPermissions  = []string{"vault:read", "vault:write", "network:outbound"}
	allowedModes        = []string{"on-demand", "persistent", "manual", "daemon"}
	allowedBackendTypes = []string{"process", "wasm"}
	allowedSettingTypes = []string{"string", "boolean", "number", "range", "enum", "select"}
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

	var m manifestJSON
	if err := json.Unmarshal(data, &m); err != nil {
		return []string{fmt.Sprintf("invalid json in manifest: %v", err)}, nil
	}

	var issues []string
	checkIdentity(&m, &issues)
	checkLifecycle(&m, &issues)
	checkPermissions(&m, &issues)
	checkBackend(&m, pluginDir, &issues)
	checkFrontend(&m, pluginDir, &issues)
	checkSettings(&m, &issues)

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

func checkIdentity(m *manifestJSON, issues *[]string) {
	if strings.TrimSpace(m.ID) == "" {
		*issues = append(*issues, "field 'id' is required")
	} else if !validIDPattern.MatchString(m.ID) {
		*issues = append(*issues, fmt.Sprintf("field 'id' %q must use reverse-DNS notation (e.g. md.numen.sample)", m.ID))
	}

	if strings.TrimSpace(m.Name) == "" {
		*issues = append(*issues, "field 'name' is required")
	}

	if strings.TrimSpace(m.Version) == "" {
		*issues = append(*issues, "field 'version' is required")
	} else if !validSemverPattern.MatchString(m.Version) {
		*issues = append(*issues, fmt.Sprintf("field 'version' %q is not valid semver (e.g. 1.0.0)", m.Version))
	}
}

func checkLifecycle(m *manifestJSON, issues *[]string) {
	mode := m.Lifecycle.Mode
	if mode != "" && !slices.Contains(allowedModes, mode) {
		*issues = append(*issues, fmt.Sprintf("lifecycle.mode %q is invalid; must be one of %v", mode, allowedModes))
	}
	if m.Lifecycle.IdleTimeoutSeconds < 0 {
		*issues = append(*issues, "lifecycle.idleTimeoutSeconds cannot be negative")
	}
}

func normalizePermission(perm string) string {
	if perm == "network" {
		return "network:outbound"
	}
	return perm
}

func checkPermissions(m *manifestJSON, issues *[]string) {
	for i, perm := range m.Permissions {
		norm := normalizePermission(perm)
		m.Permissions[i] = norm
		if !slices.Contains(allowedPermissions, norm) {
			*issues = append(*issues, fmt.Sprintf("permission %q is unknown; allowed permissions: %v", perm, allowedPermissions))
		}
	}
}

func checkBackend(m *manifestJSON, pluginDir string, issues *[]string) {
	if m.Backend == nil {
		return
	}

	if !slices.Contains(allowedBackendTypes, m.Backend.Type) {
		*issues = append(*issues, fmt.Sprintf("backend.type %q must be one of %v", m.Backend.Type, allowedBackendTypes))
	}

	if strings.TrimSpace(m.Backend.Entrypoint) == "" {
		*issues = append(*issues, "backend.entrypoint cannot be empty")
		return
	}

	entrypointPath := filepath.Join(pluginDir, m.Backend.Entrypoint)
	backendSrcPath := filepath.Join(pluginDir, "backend", "main.go")
	_, errBin := os.Stat(entrypointPath)
	_, errSrc := os.Stat(backendSrcPath)

	if errBin != nil && errSrc != nil {
		*issues = append(*issues, fmt.Sprintf("backend entrypoint %q does not exist and no backend/main.go source found", m.Backend.Entrypoint))
	}
}

func checkFrontend(m *manifestJSON, pluginDir string, issues *[]string) {
	if m.Frontend == nil {
		return
	}

	if strings.TrimSpace(m.Frontend.Entrypoint) == "" {
		*issues = append(*issues, "frontend.entrypoint cannot be empty")
		return
	}

	entrypointPath := filepath.Join(pluginDir, m.Frontend.Entrypoint)
	frontendSrcPath := filepath.Join(pluginDir, "frontend", "index.ts")
	_, errDist := os.Stat(entrypointPath)
	_, errSrc := os.Stat(frontendSrcPath)

	if errDist != nil && errSrc != nil {
		*issues = append(*issues, fmt.Sprintf("frontend entrypoint %q does not exist and no frontend/index.ts source found", m.Frontend.Entrypoint))
	}
}

func checkSettings(m *manifestJSON, issues *[]string) {
	seenKeys := make(map[string]bool)
	for _, s := range m.Settings {
		if strings.TrimSpace(s.Key) == "" {
			*issues = append(*issues, "setting key cannot be empty")
		} else if seenKeys[s.Key] {
			*issues = append(*issues, fmt.Sprintf("duplicate setting key %q", s.Key))
		}
		seenKeys[s.Key] = true

		if strings.TrimSpace(s.Title) == "" {
			*issues = append(*issues, fmt.Sprintf("setting %q title cannot be empty", s.Key))
		}
		if !slices.Contains(allowedSettingTypes, s.Type) {
			*issues = append(*issues, fmt.Sprintf("setting %q has invalid type %q; must be one of %v", s.Key, s.Type, allowedSettingTypes))
		}
	}
}
