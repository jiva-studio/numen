package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-catalog/when"
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
	Slot    string
	Command string
	View    string
	Submenu string
	Title   string
	Icon    string
	Group   string
	Order   *int
	When    string
}

func (e SlotEntry) getPlaced() (kind, id string) {
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

func (m Manifest) listSlotIssues() []error {
	declared := map[string]bool{}
	for _, command := range m.Contributes.Commands {
		declared[command.ID] = true
	}
	var issues []error
	type placement struct{ slot, kind, id string }
	observed := map[placement]bool{}
	// kinds maps a slot and the id of a command or view in it to the kind placed first; the layout
	// key of the entry is made of the slot, the plugin and the id, whatever the kind.
	kinds := map[struct{ slot, id string }]string{}
	submenus := map[string]bool{}
	windowViews := map[string]bool{}
	for _, entry := range m.Contributes.Slots {
		issues = append(issues, listSlotEntryIssues(entry, declared)...)
		kind, id := entry.getPlaced()
		key := placement{entry.Slot, kind, id}
		if kind != "" && observed[key] {
			issues = append(issues, fmt.Errorf("%w: %s %q is placed twice in %q", ErrInvalidSlotEntry, kind, id, entry.Slot))
		}
		observed[key] = true
		if kind == "command" || kind == "view" {
			entryKey := struct{ slot, id string }{entry.Slot, id}
			if first, found := kinds[entryKey]; !found {
				kinds[entryKey] = kind
			} else if first != kind {
				issues = append(issues, fmt.Errorf("%w: view and command %q share one key in %q", ErrInvalidSlotEntry, id, entry.Slot))
			}
		}
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

func listSlotEntryIssues(entry SlotEntry, declared map[string]bool) []error {
	var issues []error
	if strings.TrimSpace(entry.Slot) == "" {
		issues = append(issues, fmt.Errorf("%w: an entry has no slot", ErrInvalidSlotEntry))
	}
	if entry.Group != strings.TrimSpace(entry.Group) {
		issues = append(issues, fmt.Errorf("%w: group %q has surrounding space", ErrInvalidSlotEntry, entry.Group))
	}
	if entry.When != "" {
		if _, err := when.Parse(entry.When); err != nil {
			issues = append(issues, fmt.Errorf("%w: when %q: %w", ErrInvalidSlotEntry, entry.When, err))
		}
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

func (c Contributes) hasSlotViews() bool {
	for _, entry := range c.Slots {
		if entry.View != "" {
			return true
		}
	}
	return false
}
