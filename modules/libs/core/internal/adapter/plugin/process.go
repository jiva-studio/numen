package plugin

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
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
	if upper == "" || isSecretEnv(upper) {
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

// isSecretEnv reports whether an upper-case variable name looks like it holds a secret.
func isSecretEnv(upper string) bool {
	for _, word := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "AUTH"} {
		if strings.Contains(upper, word) {
			return true
		}
	}
	return false
}

// onnxRuntimeEnv are the variables of the host a plugin that declares the ONNX
// Runtime is also launched with: where the loader finds shared libraries, where
// the accelerator toolkits are installed and which devices the plugin may use.
var onnxRuntimeEnv = []string{
	"LD_LIBRARY_PATH",
	"CUDA_VISIBLE_DEVICES", "CUDA_PATH", "CUDA_HOME",
	"NVIDIA_VISIBLE_DEVICES", "NVIDIA_DRIVER_CAPABILITIES",
	"ROCR_VISIBLE_DEVICES", "HIP_VISIBLE_DEVICES",
}

// onnxRuntimeEnvPrefix names the versioned toolkit folders the CUDA installer sets on windows.
const onnxRuntimeEnvPrefix = "CUDA_PATH_V"

// darwinRuntimeEnv are the further loader variables passed on darwin.
var darwinRuntimeEnv = []string{"DYLD_LIBRARY_PATH", "DYLD_FALLBACK_LIBRARY_PATH"}

// isRuntimeEnv reports whether a variable is one a plugin that declares the ONNX
// Runtime is launched with on goos.
func isRuntimeEnv(goos, key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" || isSecretEnv(upper) {
		return false
	}
	if slices.Contains(onnxRuntimeEnv, upper) || strings.HasPrefix(upper, onnxRuntimeEnvPrefix) {
		return true
	}
	return goos == "darwin" && slices.Contains(darwinRuntimeEnv, upper)
}

// sanitizeEnvironment returns the host variables a plugin process is launched
// with, followed by pluginVars. A plugin that declares the ONNX Runtime among
// runtimes also gets the variables isRuntimeEnv names.
func sanitizeEnvironment(environ []string, pluginVars []string, runtimes []string) []string {
	hasONNX := slices.Contains(runtimes, port.RuntimeONNX)
	var env []string
	for _, entry := range environ {
		key, _, found := strings.Cut(entry, "=")
		if found && (isWhitelistedEnv(key) || hasONNX && isRuntimeEnv(runtime.GOOS, key)) {
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

// launchEnvironment is what the host tells a plugin process about its launch.
// Neither a secret nor a setting is part of it: both reach a plugin through
// Configure.
type launchEnvironment struct {
	vaultPath   string
	port        int
	hostPort    int
	token       string
	runtimeVars []string
}

// getPluginVars is the environment variables the host sets for a plugin
// process. The host port is set only when the host endpoint is serving, and the
// runtime variables only for a plugin that declares a runtime.
func getPluginVars(pluginID string, launch launchEnvironment) []string {
	vars := []string{
		"NUMEN_VAULT_PATH=" + launch.vaultPath,
		"NUMEN_PLUGIN_PORT=" + strconv.Itoa(launch.port),
		// This builds the launch environment of a plugin process; no credential is written here.
		// skipcq: SCT-A000
		"NUMEN_PLUGIN_TOKEN=" + launch.token,
		"NUMEN_PLUGIN_ID=" + pluginID,
	}
	if launch.hostPort > 0 {
		vars = append(vars, "NUMEN_HOST_PORT="+strconv.Itoa(launch.hostPort))
	}
	return append(vars, launch.runtimeVars...)
}

// getPluginArgs is the command line the host starts a plugin process with.
func getPluginArgs(port int, token string) []string {
	return []string{"--port=" + strconv.Itoa(port), "--token=" + token}
}
