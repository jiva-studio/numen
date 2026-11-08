package manifest

import (
	"fmt"
	"regexp"
)

// listenerProtocolTCP is the only supported listener protocol.
const listenerProtocolTCP = "tcp"

// MinListenerPort and MaxListenerPort bound the port of a listener: the host binds no privileged port.
const (
	MinListenerPort = 1024
	MaxListenerPort = 65535
)

var listenerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// Listener is a TCP port on 127.0.0.1 the host binds for the plugin while the plugin is enabled. It
// has a fixed Port or a PortSetting, exactly one.
type Listener struct {
	Name        string
	Port        int
	PortSetting string
	Protocol    string
}

func (m Manifest) listListenerIssues() []error {
	var issues []error
	listeners := m.Contributes.Listeners
	if len(listeners) > 0 && !m.HasBackend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares listeners and has no backend to serve them", ErrInvalidContribution))
	}
	seenNames := map[string]bool{}
	for _, declared := range listeners {
		switch {
		case !listenerNamePattern.MatchString(declared.Name):
			issues = append(issues, fmt.Errorf("%w: listener %q is not lowercase words joined by hyphens", ErrInvalidContribution, declared.Name))
		case seenNames[declared.Name]:
			issues = append(issues, fmt.Errorf("%w: listener %q is declared twice", ErrInvalidContribution, declared.Name))
		}
		seenNames[declared.Name] = true
		if err := m.checkListenerPort(declared); err != nil {
			issues = append(issues, err)
		}
		if declared.Protocol != listenerProtocolTCP {
			issues = append(issues, fmt.Errorf("%w: listener %q has protocol %q, want %q", ErrInvalidContribution, declared.Name, declared.Protocol, listenerProtocolTCP))
		}
	}
	return issues
}

func (m Manifest) checkListenerPort(declared Listener) error {
	switch {
	case declared.Port != 0 && declared.PortSetting != "":
		return fmt.Errorf("%w: listener %q has both a port and a port setting", ErrInvalidContribution, declared.Name)
	case declared.PortSetting != "":
		setting, ok := m.GetSetting(declared.PortSetting)
		if !ok || setting.Type != settingTypeNumber {
			return fmt.Errorf("%w: listener %q has port setting %q, which is not a number setting of the plugin", ErrInvalidContribution, declared.Name, declared.PortSetting)
		}
	case declared.Port < MinListenerPort || declared.Port > MaxListenerPort:
		return fmt.Errorf("%w: listener %q has port %d, outside %d to %d, and no port setting", ErrInvalidContribution, declared.Name, declared.Port, MinListenerPort, MaxListenerPort)
	}
	return nil
}
