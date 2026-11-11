package load

import (
	"strings"
)

// PluginsPrefix is where the plugins are compiled from. A package under it is a
// plugin, and a module under it is one too.
const PluginsPrefix = "github.com/jiva-studio/numen/plugins/"

func IsPlugin(importPath string) bool {
	return strings.HasPrefix(importPath, PluginsPrefix)
}
