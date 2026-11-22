package manifest

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidSettingGroup is a setting group declared wrongly, or a setting that names a group the
// manifest does not declare.
var ErrInvalidSettingGroup = errors.New("invalid setting group")

// SettingGroup is an island of the settings page: the settings that name its id are drawn under its
// title. A setting that names no group is drawn in the island without a heading.
type SettingGroup struct {
	ID    string
	Title string
}

func (m Manifest) listSettingGroupIssues() []error {
	var issues []error
	declared := map[string]bool{}
	for _, group := range m.SettingGroups {
		switch {
		case strings.TrimSpace(group.ID) == "":
			issues = append(issues, fmt.Errorf("%w: a group has no id", ErrInvalidSettingGroup))
		case declared[group.ID]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", ErrInvalidSettingGroup, group.ID))
		}
		declared[group.ID] = true
		if strings.TrimSpace(group.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: %q has no title", ErrInvalidSettingGroup, group.ID))
		}
	}
	for _, setting := range m.Settings {
		if setting.Group != "" && !declared[setting.Group] {
			issues = append(issues, fmt.Errorf("%w: %s names the group %q, which the manifest does not declare", ErrInvalidSettingGroup, setting.Key, setting.Group))
		}
	}
	return issues
}
