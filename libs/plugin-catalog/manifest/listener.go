package manifest

import (
	"fmt"
	"regexp"
)

// listenerProtocolTCP is the one transport a listener has.
const listenerProtocolTCP = "tcp"

const (
	minListenerPort = 1024
	maxListenerPort = 65535
)

var listenerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// Listener is a TCP port on 127.0.0.1 the host binds for the plugin while the plugin is enabled.
type Listener struct {
	// Name identifies the listener within the plugin and in the ready line of the plugin process.
	Name string
	// Port is the port the host binds.
	Port int
	// Protocol is the transport, "tcp".
	Protocol string
}

// listListenerIssues checks that each listener has a unique name of lowercase words joined by
// hyphens, a port from 1024 to 65535 and the protocol tcp, and that the plugin has a backend to
// forward the connections to.
func (m Manifest) listListenerIssues() []error {
	var issues []error
	listeners := m.Contributes.Listeners
	if len(listeners) > 0 && !m.HasBackend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares listeners and has no backend to serve them", ErrInvalidContribution))
	}
	observed := map[string]bool{}
	for _, declared := range listeners {
		switch {
		case !listenerNamePattern.MatchString(declared.Name):
			issues = append(issues, fmt.Errorf("%w: listener %q is not lowercase words joined by hyphens", ErrInvalidContribution, declared.Name))
		case observed[declared.Name]:
			issues = append(issues, fmt.Errorf("%w: listener %q is declared twice", ErrInvalidContribution, declared.Name))
		}
		observed[declared.Name] = true
		if declared.Port < minListenerPort || declared.Port > maxListenerPort {
			issues = append(issues, fmt.Errorf("%w: listener %q has port %d, outside %d to %d", ErrInvalidContribution, declared.Name, declared.Port, minListenerPort, maxListenerPort))
		}
		if declared.Protocol != listenerProtocolTCP {
			issues = append(issues, fmt.Errorf("%w: listener %q has protocol %q, want %q", ErrInvalidContribution, declared.Name, declared.Protocol, listenerProtocolTCP))
		}
	}
	return issues
}
