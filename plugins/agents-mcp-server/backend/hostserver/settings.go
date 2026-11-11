package hostserver

import "strings"

// Settings are the settings of the MCP server.
type Settings struct {
	Token string
}

// ParseSettings reads the settings from the values the host sends.
func ParseSettings(values map[string]string) Settings {
	return Settings{Token: strings.TrimSpace(values["token"])}
}
