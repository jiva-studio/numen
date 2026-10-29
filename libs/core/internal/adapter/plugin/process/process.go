package process

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/security"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var (
	// ErrEmptyWorkingDirectory indicates that an empty working directory was provided.
	ErrEmptyWorkingDirectory = errors.New("working directory cannot be empty")

	// ErrWorkingDirectoryEscape indicates that the working directory escapes the plugin directory and vault.
	ErrWorkingDirectoryEscape = errors.New("plugin working directory escapes plugin sandbox and vault")
)

// Limits defines resource bounds for out-of-process plugins.
// CPULimitSeconds is RLIMIT_CPU, the CPU time the process may use over its whole life; zero sets none.
// MemoryLimitBytes specifies resident memory threshold monitored by the supervisor.
type Limits struct {
	CPULimitSeconds  uint64
	MemoryLimitBytes uint64
}

// DefaultLimits returns standard resource limits for plugin processes. A plugin that computes
// runs for as long as it is kept, so the defaults set no CPU limit.
func DefaultLimits() Limits {
	return Limits{
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

// IsWhitelistedEnv reports whether a variable of the host environment is
// passed to a plugin process: the system's own variables, the proxy the machine
// uses for network access, and NUMEN_MOCK_ML, which runs a plugin on its mock
// models.
func IsWhitelistedEnv(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" || isSecretEnv(upper) {
		return false
	}
	switch upper {
	case "PATH", "TMPDIR", "TMP", "TEMP", "HOME", "USER",
		"SYSTEMROOT", "SYSTEMDRIVE", "APPDATA", "LOCALAPPDATA",
		"LANG", "SSL_CERT_FILE", "SSL_CERT_DIR", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "NUMEN_MOCK_ML":
		return true
	}
	return strings.HasPrefix(upper, "LC_")
}

// isSecretEnv reports whether an upper-case variable name looks like it holds a secret.
func isSecretEnv(upper string) bool {
	for _, word := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "AUTH"} {
		if strings.Contains(upper, word) {
			return true
		}
	}
	return false
}

// onnxRuntimeEnvPrefix names the versioned toolkit folders the CUDA installer sets on windows.
const onnxRuntimeEnvPrefix = "CUDA_PATH_V"

// isRuntimeEnv reports whether a variable is one of the host variables a plugin
// that declares the ONNX Runtime is also launched with on goos: where the loader
// finds shared libraries, where the accelerator toolkits are installed and which
// devices the plugin may use.
func isRuntimeEnv(goos, key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" || isSecretEnv(upper) {
		return false
	}
	switch upper {
	case "LD_LIBRARY_PATH",
		"CUDA_VISIBLE_DEVICES", "CUDA_PATH", "CUDA_HOME",
		"NVIDIA_VISIBLE_DEVICES", "NVIDIA_DRIVER_CAPABILITIES",
		"ROCR_VISIBLE_DEVICES", "HIP_VISIBLE_DEVICES":
		return true
	case "DYLD_LIBRARY_PATH", "DYLD_FALLBACK_LIBRARY_PATH":
		return goos == "darwin"
	}
	return strings.HasPrefix(upper, onnxRuntimeEnvPrefix)
}

// SanitizeEnvironment returns the host variables a plugin process is launched
// with, followed by pluginVars. A plugin that declares the ONNX Runtime among
// runtimes also gets the variables isRuntimeEnv names.
func SanitizeEnvironment(environ []string, pluginVars []string, runtimes []string) []string {
	hasONNX := slices.Contains(runtimes, pluginport.RuntimeONNX)
	var env []string
	for _, entry := range environ {
		key, _, found := strings.Cut(entry, "=")
		if found && (IsWhitelistedEnv(key) || hasONNX && isRuntimeEnv(runtime.GOOS, key)) {
			env = append(env, entry)
		}
	}
	return append(env, pluginVars...)
}

const (
	// EnvONNXRuntimeLib is the absolute path of the ONNX Runtime library the host provisioned.
	EnvONNXRuntimeLib = "NUMEN_ONNX_RUNTIME_LIB"

	// EnvONNXProvider is the execution provider a plugin opens its sessions on.
	EnvONNXProvider = "NUMEN_ONNX_PROVIDER"
)

// LaunchEnvironment is what the host passes to a plugin process about its
// launch. It contains neither secrets nor settings: both arrive through
// Configure.
type LaunchEnvironment struct {
	VaultPath   string
	Port        int
	HostPort    int
	Token       string
	RuntimeVars []string
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
	return append(vars, launch.RuntimeVars...)
}

// GetPluginArgs is the command line the host starts a plugin process with.
func GetPluginArgs(port int) []string {
	return []string{"--port=" + strconv.Itoa(port)}
}
