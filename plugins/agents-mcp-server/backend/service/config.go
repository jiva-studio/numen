package service

import (
	"os"
	"strings"
)

// Config configures the MCP server.
type Config struct {
	Token string
}

// readConfig builds a configuration from values and environment variables.
func readConfig(values map[string]string) Config {
	token := values["token"]
	if token == "" {
		token = os.Getenv("MCP_TOKEN")
	}
	return Config{Token: strings.TrimSpace(token)}
}
