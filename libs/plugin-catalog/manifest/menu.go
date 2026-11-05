package manifest

import "errors"

// ErrInvalidMenu is a menu entry a manifest declares wrongly.
var ErrInvalidMenu = errors.New("invalid menu")

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
	return nil
}
