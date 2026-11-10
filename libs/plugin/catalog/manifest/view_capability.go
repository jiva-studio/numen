package manifest

import (
	"fmt"
	"strings"
)

// ApplicationViewCapabilities lists the kinds of view the application defines. A plugin draws one for
// others to mount only when its manifest lists the kind under contributes.viewCapabilities.
var ApplicationViewCapabilities = []string{"conversation", "note-editor", "vault-summary"}

func (m Manifest) listViewCapabilityIssues() []error {
	var issues []error
	if len(m.Contributes.ViewCapabilities) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares view capabilities and has no frontend to register them", ErrInvalidContribution))
	}
	observed := map[string]bool{}
	for _, capability := range m.Contributes.ViewCapabilities {
		if observed[capability] {
			issues = append(issues, fmt.Errorf("%w: view capability %q is declared twice", ErrInvalidContribution, capability))
		}
		observed[capability] = true
		if !m.isViewCapabilityAllowed(capability) {
			issues = append(issues, fmt.Errorf("%w: view capability %q is neither one of %s nor named %q followed by a name", ErrInvalidContribution, capability, strings.Join(ApplicationViewCapabilities, ", "), m.ID+"."))
		}
	}
	return issues
}

func (m Manifest) isViewCapabilityAllowed(capability string) bool {
	for _, known := range ApplicationViewCapabilities {
		if capability == known {
			return true
		}
	}
	rest, isOwn := strings.CutPrefix(capability, m.ID+".")
	return isOwn && rest != "" && tabKindPattern.MatchString(rest)
}
