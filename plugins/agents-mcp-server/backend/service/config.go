package service

import "strings"

// Config configures the MCP server.
type Config struct {
	Token string
}

// readConfig reads the configuration from the setting values.
func readConfig(values map[string]string) Config {
	return Config{Token: strings.TrimSpace(values["token"])}
}
