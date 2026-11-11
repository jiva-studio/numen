package transport

import "fmt"

// Endpoint holds connection parameters and credentials for an active plugin.
type Endpoint struct {
	Host  string
	Port  int
	Token string
}

// GetAddress returns host and port joined with a colon.
func (e Endpoint) GetAddress() string {
	if e.Port == 0 {
		return e.Host
	}
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}
