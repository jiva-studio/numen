//go:build plugins_monolith

package platform

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/modules/libs/core/container"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

// builtInPlugin is a plugin linked into this binary: its manifest.json and what makes the handler
// of its backend.
type builtInPlugin struct {
	manifest []byte
	serve    func(vars map[string]string) (http.Handler, error)
}

// builtInPlugins are the plugins linked into this binary, in the order they were registered.
var builtInPlugins []builtInPlugin

// RegisterPlugin links a plugin into this binary from its manifest.json and what makes the handler
// of its backend. Every plugin host made afterwards serves it in-process.
func RegisterPlugin(manifest []byte, serve func(vars map[string]string) (http.Handler, error)) {
	SetMonolith(true)
	builtInPlugins = append(builtInPlugins, builtInPlugin{manifest: manifest, serve: serve})
}

// newPluginHost is the plugin host of this binary, with the plugins linked into it registered. A
// plugin that cannot be registered is reported and left out.
func newPluginHost(target string, dirs ...string) pluginport.PluginManager {
	host := container.NewPluginHost(target, dirs...)
	for _, builtIn := range builtInPlugins {
		if err := host.RegisterBuiltIn(builtIn.manifest, builtIn.serve); err != nil {
			fmt.Fprintf(os.Stderr, "numen: %v\n", err)
		}
	}
	return host
}
