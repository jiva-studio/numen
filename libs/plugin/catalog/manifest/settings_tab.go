package manifest

import (
	"fmt"
	"strings"
)

// TabSettings is the id of the default tab of the settings page. It always exists, comes first and holds the
// plain settings rows, and a plugin does not declare it.
const TabSettings = "settings"

// SettingsTab is a tab of the settings page besides the default one.
type SettingsTab struct {
	ID    string
	Title string
}

func (m Manifest) listSettingsTabIssues() []error {
	var issues []error
	declared := map[string]bool{TabSettings: true}
	for _, tab := range m.SettingsTabs {
		switch {
		case strings.TrimSpace(tab.ID) == "":
			issues = append(issues, fmt.Errorf("%w: a tab has no id", ErrInvalidSettingsView))
		case declared[tab.ID]:
			issues = append(issues, fmt.Errorf("%w: the tab %q is declared twice or is the default tab", ErrInvalidSettingsView, tab.ID))
		}
		declared[tab.ID] = true
		if strings.TrimSpace(tab.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: the tab %q has no title", ErrInvalidSettingsView, tab.ID))
		}
	}
	for _, view := range m.SettingsViews {
		if view.Tab != "" && !declared[view.Tab] {
			issues = append(issues, fmt.Errorf("%w: %q names the tab %q, which the manifest does not declare", ErrInvalidSettingsView, view.ID, view.Tab))
		}
	}
	return issues
}
