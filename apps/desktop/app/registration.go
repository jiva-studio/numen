//go:build plugins_monolith

package app

import (
	"net/http"

	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
)

// RegisterPlugin links a plugin into this binary from its manifest.json and what makes the handler
// of its backend from the variables a launched process of the plugin is given.
func RegisterPlugin(manifest []byte, serve func(vars map[string]string) (http.Handler, error)) {
	platform.RegisterPlugin(manifest, serve)
}
