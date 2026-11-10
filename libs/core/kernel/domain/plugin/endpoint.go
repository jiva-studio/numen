package plugin

import "fmt"

// PersistHandler is called when a plugin modifies its own settings.
type PersistHandler func(pluginID string, key string, value any) error

// Endpoint holds connection parameters and credentials for an active plugin.
type Endpoint struct {
	Host  string
	Port  int
	Token string
}

// Address returns host and port joined with a colon.
func (e Endpoint) Address() string {
	if e.Port == 0 {
		return e.Host
	}
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}
