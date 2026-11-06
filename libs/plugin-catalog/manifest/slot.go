package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidSlotEntry is an entry of contributes.slots a manifest declares wrongly.
var ErrInvalidSlotEntry = errors.New("invalid slot entry")

var (
	// submenuIDPattern is the id of a submenu: words joined by dots, such as fileTree.new.
	submenuIDPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*(\.[a-z][a-zA-Z0-9]*)+$`)
	// fullCommandIDPattern is the id of a command with its owner, such as app:settings.open.
	fullCommandIDPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9.]*:[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$`)
)

// SlotEntry places one thing of the plugin in a slot of a window, or in a submenu a plugin declares.
// Exactly one of Command, View and Submenu is set.
type SlotEntry struct {
	// Slot is a slot of the list of Slots, or the id of a submenu.
	Slot string
	// Command is the id the manifest declares the command under, or the full id of a command of
	// another owner, such as app:settings.open. The owner of the slot draws it.
	Command string
	// View is the id the frontend registers the component under. The component draws itself.
	View string
	// Submenu is the id of the submenu the entry declares, which becomes a place of its own.
	Submenu string
	// Title is the title of a submenu.
	Title string
	// Icon is the kebab-case icon name of a submenu; empty for none.
	Icon string
	// Group names the entries a slot draws together; empty is the last group.
	Group string
	// Order sorts the entries of a group, lowest first; nil sorts after every order.
	Order *int
}

// placed is what the entry places, as its kind and id, such as command ledger.open.
func (e SlotEntry) placed() (kind, id string) {
	switch {
	case e.Command != "":
		return "command", e.Command
	case e.View != "":
		return "view", e.View
	case e.Submenu != "":
		return "submenu", e.Submenu
	}
	return "", ""
}

// listSlotIssues checks the entries of contributes.slots. A slot outside the list may be the submenu of
// another plugin, so it is checked once every manifest is read, not here.
func (m Manifest) listSlotIssues() []error {
	declared := map[string]bool{}
	for _, command := range m.Contributes.Commands {
		declared[command.ID] = true
	}
	var issues []error
	type placement struct{ slot, kind, id string }
	observed := map[placement]bool{}
	submenus := map[string]bool{}
	windowViews := map[string]bool{}
	for _, entry := range m.Contributes.Slots {
		issues = append(issues, listSlotEntryIssues(entry, declared)...)
		kind, id := entry.placed()
		key := placement{entry.Slot, kind, id}
		if kind != "" && observed[key] {
			issues = append(issues, fmt.Errorf("%w: %s %q is placed twice in %q", ErrInvalidSlotEntry, kind, id, entry.Slot))
		}
		observed[key] = true
		if entry.Submenu != "" && submenus[entry.Submenu] {
			issues = append(issues, fmt.Errorf("%w: submenu %q is declared twice", ErrInvalidSlotEntry, entry.Submenu))
		}
		submenus[entry.Submenu] = true
		if entry.View != "" && IsWindowSlot(entry.Slot) {
			if windowViews[entry.Slot] {
				issues = append(issues, fmt.Errorf("%w: the window slot %q holds one view, and the plugin places two", ErrInvalidSlotEntry, entry.Slot))
			}
			windowViews[entry.Slot] = true
		}
	}
	return issues
}

// listSlotEntryIssues checks one entry against the rules every entry follows.
func listSlotEntryIssues(entry SlotEntry, declared map[string]bool) []error {
	var issues []error
	if strings.TrimSpace(entry.Slot) == "" {
		issues = append(issues, fmt.Errorf("%w: an entry has no slot", ErrInvalidSlotEntry))
	}
	if entry.Group != strings.TrimSpace(entry.Group) {
		issues = append(issues, fmt.Errorf("%w: group %q has surrounding space", ErrInvalidSlotEntry, entry.Group))
	}
	switch {
	case entry.Command != "":
		if !declared[entry.Command] && !fullCommandIDPattern.MatchString(entry.Command) {
			issues = append(issues, fmt.Errorf("%w: command %q is neither declared by the manifest nor the full id of a command", ErrInvalidSlotEntry, entry.Command))
		}
	case entry.View != "":
		if !tabKindPattern.MatchString(entry.View) {
			issues = append(issues, fmt.Errorf("%w: view %q is not lowercase words", ErrInvalidSlotEntry, entry.View))
		}
	case entry.Submenu != "":
		issues = append(issues, listSubmenuIssues(entry)...)
	default:
		issues = append(issues, fmt.Errorf("%w: the entry in %q places no command, view or submenu", ErrInvalidSlotEntry, entry.Slot))
	}
	return issues
}

// listSubmenuIssues checks the id, title and icon of a submenu an entry declares.
func listSubmenuIssues(entry SlotEntry) []error {
	var issues []error
	switch {
	case !submenuIDPattern.MatchString(entry.Submenu):
		issues = append(issues, fmt.Errorf("%w: submenu %q is not words joined by dots", ErrInvalidSlotEntry, entry.Submenu))
	case IsSlot(entry.Submenu):
		issues = append(issues, fmt.Errorf("%w: submenu %q is named as a slot of the window", ErrInvalidSlotEntry, entry.Submenu))
	}
	if strings.TrimSpace(entry.Title) == "" {
		issues = append(issues, fmt.Errorf("%w: submenu %q has no title", ErrInvalidSlotEntry, entry.Submenu))
	}
	if entry.Icon != "" && !iconNamePattern.MatchString(entry.Icon) {
		issues = append(issues, fmt.Errorf("%w: submenu %q has icon %q, which is not lowercase words joined by hyphens", ErrInvalidSlotEntry, entry.Submenu, entry.Icon))
	}
	return issues
}

// hasSlotViews reports whether the manifest places a view, which only a frontend registers.
func (c Contributes) hasSlotViews() bool {
	for _, entry := range c.Slots {
		if entry.View != "" {
			return true
		}
	}
	return false
}
