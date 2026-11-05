package manifest

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidMenu is a menu entry a manifest declares wrongly.
var ErrInvalidMenu = errors.New("invalid menu")

// editorCommandOwner prefixes the id of a command of the editor.
const editorCommandOwner = "app:"

// Menu places a command in a place the window draws.
type Menu struct {
	// Location is the name of the place.
	Location string
	// Command is the short id of a command of this manifest, or the full id of an editor command,
	// whose owner is "app".
	Command string
	// Group is the name that tells the place to draw entries together. Empty is no group.
	Group string
	// Order sorts the entries of a group, lowest first. Nil is no order.
	Order *int
}

// listMenuIssues checks the menu entries the manifest declares.
func (m Manifest) listMenuIssues() []error {
	var issues []error
	type placement struct{ location, command string }
	observed := map[placement]bool{}
	for _, entry := range m.Contributes.Menus {
		if strings.TrimSpace(entry.Location) == "" {
			issues = append(issues, fmt.Errorf("%w: an entry of command %q has no location", ErrInvalidMenu, entry.Command))
		}
		switch {
		case strings.TrimSpace(entry.Command) == "":
			issues = append(issues, fmt.Errorf("%w: an entry in %q has no command", ErrInvalidMenu, entry.Location))
		case !isMenuCommand(entry.Command):
			issues = append(issues, fmt.Errorf("%w: command %q is neither the id of a command of this plugin nor the id of a command of the editor", ErrInvalidMenu, entry.Command))
		}
		if entry.Group != strings.TrimSpace(entry.Group) {
			issues = append(issues, fmt.Errorf("%w: group %q has surrounding space", ErrInvalidMenu, entry.Group))
		}
		key := placement{entry.Location, entry.Command}
		if observed[key] {
			issues = append(issues, fmt.Errorf("%w: command %q is declared twice in %q", ErrInvalidMenu, entry.Command, entry.Location))
		}
		observed[key] = true
	}
	return issues
}

// isMenuCommand reports whether the command is a short id or the id of a command of the editor.
func isMenuCommand(command string) bool {
	return manifestCommandPattern.MatchString(strings.TrimPrefix(command, editorCommandOwner))
}
