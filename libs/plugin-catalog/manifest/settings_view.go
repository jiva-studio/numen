package manifest

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidSettingsView is a settings view declared wrongly.
var ErrInvalidSettingsView = errors.New("invalid settings view")

// SettingsView is a component of the plugin's frontend that the settings page draws among the
// settings of the plugin.
type SettingsView struct {
	// ID is the id the frontend registers the component under.
	ID    string
	Title string
	// After is the key of the setting the component is drawn after. Empty draws it at the top.
	After string
}

// listSettingsViewIssues checks that each view has an id of its own and a title, names a setting
// declared by the manifest as the one it follows, and that the plugin has a frontend to draw it.
func (m Manifest) listSettingsViewIssues() []error {
	var issues []error
	if len(m.SettingsViews) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares settings views and has no frontend to draw them", ErrInvalidSettingsView))
	}
	keys := make(map[string]bool, len(m.Settings))
	for _, s := range m.Settings {
		keys[strings.TrimSpace(s.Key)] = true
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
		if view.After != "" && !keys[view.After] {
			issues = append(issues, fmt.Errorf("%w: %q follows %q, which is no setting of the plugin", ErrInvalidSettingsView, view.ID, view.After))
		}
	}
	return issues
}
