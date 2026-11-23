// Package hostcall is how a plugin calls the host: where the host endpoint is, and the token that
// identifies the calling plugin.
package hostcall

import (
	"errors"
	"os"
	"strings"
)

const (
	// HostPortEnv names the environment variable holding the host endpoint port.
	HostPortEnv = "NUMEN_HOST_PORT"
	// PluginTokenEnv names the environment variable holding the plugin launch token.
	PluginTokenEnv = "NUMEN_PLUGIN_TOKEN"
)

// ErrNoHost means the environment names no host endpoint.
var ErrNoHost = errors.New("host: " + HostPortEnv + " is not set")

// Endpoint is where the host is, and the token the plugin calls it with.
type Endpoint struct {
	BaseURL string
	Token   string
}

// ReadEnv returns the endpoint the environment of the process names.
func ReadEnv() (Endpoint, error) {
	return ReadVars(map[string]string{
		HostPortEnv:    os.Getenv(HostPortEnv),
		PluginTokenEnv: os.Getenv(PluginTokenEnv),
	})
}

// ReadVars returns the endpoint the NUMEN_HOST_PORT and NUMEN_PLUGIN_TOKEN entries of vars name. They are
// the variables a host gives a plugin it links into its own process.
func ReadVars(vars map[string]string) (Endpoint, error) {
	port := strings.TrimSpace(vars[HostPortEnv])
	if port == "" {
		return Endpoint{}, ErrNoHost
	}
	return Endpoint{BaseURL: "http://127.0.0.1:" + port, Token: vars[PluginTokenEnv]}, nil
}
