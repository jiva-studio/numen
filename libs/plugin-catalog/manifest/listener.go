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

// Listener is a TCP port on 127.0.0.1 the host binds for the plugin while the plugin is enabled. It
// has a fixed Port or a PortSetting, exactly one.
type Listener struct {
	// Name identifies the listener within the plugin and in the ready line of the plugin process.
	Name string
	// Port is the fixed port the host binds. Zero for a listener with a PortSetting.
	Port int
	// PortSetting is the key of the number setting of the plugin that holds the port the host binds.
	// Empty for a listener with a fixed Port.
	PortSetting string
	// Protocol is the transport, "tcp".
	Protocol string
}

// listListenerIssues checks the listeners of the manifest: a unique name of lowercase words joined
// by hyphens, a fixed port from 1024 to 65535 or a port setting of type number, exactly one, the
// protocol tcp, and a backend to forward the connections to.
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
		if err := m.checkListenerPort(declared); err != nil {
			issues = append(issues, err)
		}
		if declared.Protocol != listenerProtocolTCP {
			issues = append(issues, fmt.Errorf("%w: listener %q has protocol %q, want %q", ErrInvalidContribution, declared.Name, declared.Protocol, listenerProtocolTCP))
		}
	}
	return issues
}

// checkListenerPort checks that the listener has a fixed port from 1024 to 65535 or a port setting
// that names a number setting of the manifest, exactly one.
func (m Manifest) checkListenerPort(declared Listener) error {
	switch {
	case declared.Port != 0 && declared.PortSetting != "":
		return fmt.Errorf("%w: listener %q has both a port and a port setting", ErrInvalidContribution, declared.Name)
	case declared.PortSetting != "":
		setting, ok := m.GetSetting(declared.PortSetting)
		if !ok || setting.Type != settingTypeNumber {
			return fmt.Errorf("%w: listener %q has port setting %q, which is not a number setting of the plugin", ErrInvalidContribution, declared.Name, declared.PortSetting)
		}
	case declared.Port < minListenerPort || declared.Port > maxListenerPort:
		return fmt.Errorf("%w: listener %q has port %d, outside %d to %d, and no port setting", ErrInvalidContribution, declared.Name, declared.Port, minListenerPort, maxListenerPort)
	}
	return nil
}
