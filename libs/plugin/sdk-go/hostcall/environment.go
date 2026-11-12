package hostcall

import (
	"os"
	"slices"
	"strings"
)

const (
	// PluginPortEnv names the environment variable holding the port the plugin listens on.
	PluginPortEnv = "NUMEN_PLUGIN_PORT"
	// PluginIDEnv names the environment variable holding the plugin id.
	PluginIDEnv = "NUMEN_PLUGIN_ID"
	// VaultPathEnv names the environment variable holding the vault folder.
	VaultPathEnv = "NUMEN_VAULT_PATH"
)

var hostEnvs = []string{HostPortEnv, PluginTokenEnv, PluginPortEnv, PluginIDEnv, VaultPathEnv}

// GetChildEnvironment returns the environment of the process as name=value entries without the
// variables the host sets for the plugin, so a child process does not inherit the plugin token or
// the host endpoint.
func GetChildEnvironment() []string {
	return slices.DeleteFunc(os.Environ(), func(entry string) bool {
		name, _, _ := strings.Cut(entry, "=")
		return slices.Contains(hostEnvs, name)
	})
}
