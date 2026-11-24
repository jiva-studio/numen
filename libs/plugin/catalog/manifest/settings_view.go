package manifest

import (
	"errors"
	"fmt"
	"strings"
)

// pageIdentity is the page "Identity and licenses", the one page besides its own settings that a settings view may be drawn on.
const pageIdentity = "identity"

// ErrInvalidSettingsView is a settings view declared wrongly.
var ErrInvalidSettingsView = errors.New("invalid settings view")

// SettingsView is a component of the plugin's frontend that the settings page draws among the
// settings of the plugin.
type SettingsView struct {
	ID    string
	Title string
	Page  string
	Tab   string
}

func (m Manifest) listSettingsViewIssues() []error {
	var issues []error
	if len(m.SettingsViews) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares settings views and has no frontend to draw them", ErrInvalidSettingsView))
	}
	observed := map[string]bool{}
	for _, view := range m.SettingsViews {
		switch {
		case strings.TrimSpace(view.ID) == "":
			issues = append(issues, fmt.Errorf("%w: a view has no id", ErrInvalidSettingsView))
		case observed[view.ID]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", ErrInvalidSettingsView, view.ID))
		}
		observed[view.ID] = true
		if strings.TrimSpace(view.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: %q has no title", ErrInvalidSettingsView, view.ID))
		}
		if view.Page != "" && view.Page != pageIdentity {
			issues = append(issues, fmt.Errorf("%w: %q has the page %q, want none or %q", ErrInvalidSettingsView, view.ID, view.Page, pageIdentity))
		}
	}
	return issues
}
