package plugin

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	// ErrEmptyWorkingDirectory indicates that an empty working directory was provided.
	ErrEmptyWorkingDirectory = errors.New("working directory cannot be empty")

	// ErrWorkingDirectoryEscape indicates that the working directory escapes the plugin directory and vault.
	ErrWorkingDirectoryEscape = errors.New("plugin working directory escapes plugin sandbox and vault")
)

// ProcessLimits defines resource bounds for out-of-process plugins.
// MemoryLimitBytes specifies resident memory threshold monitored by the supervisor.
type ProcessLimits struct {
	CPULimitSeconds  uint64
	MemoryLimitBytes uint64
}

// DefaultProcessLimits returns standard resource limits for plugin processes.
func DefaultProcessLimits() ProcessLimits {
	return ProcessLimits{
		CPULimitSeconds:  1800,
		MemoryLimitBytes: 4 * 1024 * 1024 * 1024,
	}
}

// ValidateWorkingDirectory verifies that workDir resides within either pluginDir or vaultRoot.
func ValidateWorkingDirectory(workDir, pluginDir, vaultRoot string) (string, error) {
	if strings.TrimSpace(workDir) == "" {
		return "", ErrEmptyWorkingDirectory
	}

	cleanWork, err := filepath.Abs(filepath.Clean(workDir))
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	if isSystemTempDir(cleanWork) {
		return "", errProhibitedTempDir
	}

	if isPathWithin(pluginDir, cleanWork) {
		return cleanWork, nil
	}
	if strings.TrimSpace(vaultRoot) != "" && isPathWithin(vaultRoot, cleanWork) {
		return cleanWork, nil
	}

	return "", ErrWorkingDirectoryEscape
}

func isPathWithin(base, child string) bool {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(child) == "" {
		return false
	}
	cleanBase, err := filepath.Abs(filepath.Clean(base))
	if err != nil {
		return false
	}
	cleanChild, err := filepath.Abs(filepath.Clean(child))
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(cleanBase, cleanChild)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}

	realBase, errB := filepath.EvalSymlinks(cleanBase)
	realChild, errC := filepath.EvalSymlinks(cleanChild)
	if errB == nil && errC == nil {
		relReal, errRel := filepath.Rel(realBase, realChild)
		if errRel != nil || relReal == ".." || strings.HasPrefix(relReal, ".."+string(filepath.Separator)) {
			return false
		}
	}
	return true
}

func isWhitelistedEnv(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" {
		return false
	}
	if strings.Contains(upper, "KEY") ||
		strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "SECRET") ||
		strings.Contains(upper, "PASSWORD") ||
		strings.Contains(upper, "CREDENTIAL") ||
		strings.Contains(upper, "AUTH") {
		return false
	}
	switch upper {
	case "PATH", "TMPDIR", "TMP", "TEMP", "HOME", "USER",
		"SYSTEMROOT", "SYSTEMDRIVE", "APPDATA", "LOCALAPPDATA",
		"LANG", "SSL_CERT_FILE", "SSL_CERT_DIR":
		return true
	}
	return strings.HasPrefix(upper, "LC_")
}

func sanitizeEnvironment(environ []string, pluginVars []string) []string {
	var env []string
	for _, entry := range environ {
		key, _, found := strings.Cut(entry, "=")
		if found && isWhitelistedEnv(key) {
			env = append(env, entry)
		}
	}
	return append(env, pluginVars...)
}

// launchEnvironment is what the host tells a plugin process about its launch.
// A secret is never part of it: a key reaches a plugin through Configure.
type launchEnvironment struct {
	vaultPath  string
	port       int
	hostPort   int
	token      string
	configJSON string
}

// getPluginVars is the environment variables the host sets for a plugin
// process. The host port is set only when the host endpoint is serving.
func getPluginVars(pluginID string, launch launchEnvironment) []string {
	vars := []string{
		"NUMEN_VAULT_PATH=" + launch.vaultPath,
		"NUMEN_PLUGIN_PORT=" + strconv.Itoa(launch.port),
		// This builds the launch environment of a plugin process; no credential is written here.
		// skipcq: SCT-A000
		"NUMEN_PLUGIN_TOKEN=" + launch.token,
		"NUMEN_PLUGIN_ID=" + pluginID,
		"NUMEN_PLUGIN_CONFIG=" + launch.configJSON,
	}
	if launch.hostPort > 0 {
		vars = append(vars, "NUMEN_HOST_PORT="+strconv.Itoa(launch.hostPort))
	}
	return vars
}
