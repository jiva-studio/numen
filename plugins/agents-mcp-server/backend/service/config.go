package service

import "strings"

// Config configures the MCP server.
type Config struct {
	Token string
}

func readConfig(values map[string]string) Config {
	return Config{Token: strings.TrimSpace(values["token"])}
}
