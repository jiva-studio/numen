package process

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/security"
)

var (
	// ErrEmptyWorkingDirectory indicates that an empty working directory was provided.
	ErrEmptyWorkingDirectory = errors.New("working directory cannot be empty")

	// ErrWorkingDirectoryEscape indicates that the working directory escapes the plugin directory and vault.
	ErrWorkingDirectoryEscape = errors.New("plugin working directory escapes plugin sandbox and vault")
)

// Limits defines resource bounds for out-of-process plugins.
// CPULimitSeconds is RLIMIT_CPU, the CPU time the process may use over its whole life; zero sets none.
type Limits struct {
	CPULimitSeconds uint64
}

// DefaultLimits returns standard resource limits for plugin processes. A plugin that computes
// runs for as long as it is kept, so the defaults set no CPU limit.
func DefaultLimits() Limits {
	return Limits{}
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

	if security.IsSystemTempDir(cleanWork) {
		return "", security.ErrProhibitedTempDir
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

	if !isRelativeInside(cleanBase, cleanChild) {
		return false
	}

	// A path that cannot be resolved cannot be shown to stay inside.
	realBase, err := filepath.EvalSymlinks(cleanBase)
	if err != nil {
		return false
	}
	realChild, err := filepath.EvalSymlinks(cleanChild)
	if err != nil {
		return false
	}
	return isRelativeInside(realBase, realChild)
}

func isRelativeInside(base, child string) bool {
	rel, err := filepath.Rel(base, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// IsWhitelistedEnv reports whether a variable of the host environment is
// passed to a plugin process: the system's own variables, the proxy the machine
// uses for network access, and every variable that starts with testEnvPrefix,
// which tests set to run a plugin on its mock models.
func IsWhitelistedEnv(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" || isSecretEnv(upper) {
		return false
	}
	switch upper {
	case "PATH", "TMPDIR", "TMP", "TEMP", "HOME", "USER",
		"SYSTEMROOT", "SYSTEMDRIVE", "APPDATA", "LOCALAPPDATA",
		"LANG", "SSL_CERT_FILE", "SSL_CERT_DIR", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY":
		return true
	}
	return strings.HasPrefix(upper, "LC_") || strings.HasPrefix(upper, testEnvPrefix)
}

// testEnvPrefix starts the variables tests set for a plugin process.
const testEnvPrefix = "NUMEN_TEST_"

func isSecretEnv(upper string) bool {
	for _, word := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "AUTH"} {
		if strings.Contains(upper, word) {
			return true
		}
	}
	return false
}

func isDeclaredEnv(declared []string, key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" || isSecretEnv(upper) {
		return false
	}
	for _, entry := range declared {
		prefix, isPrefix := strings.CutSuffix(strings.ToUpper(entry), "*")
		if (isPrefix && strings.HasPrefix(upper, prefix)) || (!isPrefix && prefix == upper) {
			return true
		}
	}
	return false
}

// SanitizeEnvironment returns the host variables a plugin process is launched
// with, followed by pluginVars. The variables declared, which the dependencies of
// the plugin name, are passed besides those IsWhitelistedEnv passes.
func SanitizeEnvironment(environ []string, pluginVars []string, declared []string) []string {
	var env []string
	for _, entry := range environ {
		key, _, found := strings.Cut(entry, "=")
		if found && (IsWhitelistedEnv(key) || isDeclaredEnv(declared, key)) {
			env = append(env, entry)
		}
	}
	return append(env, pluginVars...)
}

// LaunchEnvironment is what the host passes to a plugin process about its
// launch. It contains neither secrets nor settings: both arrive through
// Configure.
type LaunchEnvironment struct {
	VaultPath   string
	Port        int
	HostPort    int
	Token       string
	RuntimeVars []string
	StorageVars []string
}

// GetPluginVars is the environment variables the host sets for a plugin
// process. The host port is set only when the host endpoint is serving, and the
// runtime variables only for a plugin that declares a runtime.
func GetPluginVars(pluginID string, launch LaunchEnvironment) []string {
	vars := []string{
		"NUMEN_VAULT_PATH=" + launch.VaultPath,
		"NUMEN_PLUGIN_PORT=" + strconv.Itoa(launch.Port),
		// This builds the launch environment of a plugin process; no credential is written here.
		// skipcq: SCT-A000
		"NUMEN_PLUGIN_TOKEN=" + launch.Token,
		"NUMEN_PLUGIN_ID=" + pluginID,
	}
	if launch.HostPort > 0 {
		vars = append(vars, "NUMEN_HOST_PORT="+strconv.Itoa(launch.HostPort))
	}
	vars = append(vars, launch.StorageVars...)
	return append(vars, launch.RuntimeVars...)
}

// NewCommand returns the command that starts the plugin binary at binPath listening on port, with the process
// attributes of the platform.
func NewCommand(ctx context.Context, binPath string, port int) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binPath, "--port="+strconv.Itoa(port))
	configureSysProcAttr(cmd)
	return cmd
}
