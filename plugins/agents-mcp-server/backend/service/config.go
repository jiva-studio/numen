package service

import (
	"os"
	"strings"
)

// Config configures the MCP server listener.
type Config struct {
	Addr  string
	Token string
}

// readConfig builds a configuration from values and environment variables.
func readConfig(values map[string]string) Config {
	addr := values["addr"]
	if addr == "" {
		addr = values["address"]
	}
	if addr == "" {
		if p := values["port"]; p != "" {
			addr = "127.0.0.1:" + p
		}
	}
	if addr == "" {
		if p := os.Getenv("MCP_PORT"); p != "" {
			addr = "127.0.0.1:" + p
		}
	}

	token := values["token"]
	if token == "" {
		token = os.Getenv("MCP_TOKEN")
	}

	return Config{
		Addr:  strings.TrimSpace(addr),
		Token: strings.TrimSpace(token),
	}
}
