package manifest

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

// ErrInvalidSlotEntry is an entry of contributes.slots a manifest declares wrongly.
var ErrInvalidSlotEntry = errors.New("invalid slot entry")

var (
	// submenuIDPattern is the id of a submenu: words joined by dots, such as fileTree.new.
	submenuIDPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*(\.[a-z][a-zA-Z0-9]*)+$`)
	// fullCommandIDPattern is the id of a command with its owner, such as app:settings.open.
	fullCommandIDPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9.]*:[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$`)
)

// AppCommandArgument is an argument of an app command; an argument of the type enum lists the values it
// allows in Options.
type AppCommandArgument struct {
	Name    string
	Title   string
	Type    string
	Options []string
}

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
	Values  map[string]string
}

func (e SlotEntry) getPlacement() (kind, id string) {
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

func (e SlotEntry) getValuesKey() string {
	parts := make([]string, 0, len(e.Values))
	for _, name := range slices.Sorted(maps.Keys(e.Values)) {
		parts = append(parts, strconv.Quote(name)+"="+strconv.Quote(e.Values[name]))
	}
	return strings.Join(parts, "&")
}

func (m Manifest) listSlotIssues() []error {
	declared := map[string]Command{}
	for _, command := range m.Contributes.Commands {
		declared[command.ID] = command
	}
	var issues []error
	type placement struct{ slot, kind, id, values string }
	observed := map[placement]bool{}
	submenus := map[string]bool{}
	windowViews := map[string]bool{}
	for _, entry := range m.Contributes.Slots {
		issues = append(issues, listSlotEntryIssues(entry, declared)...)
		kind, id := entry.getPlacement()
		key := placement{entry.Slot, kind, id, entry.getValuesKey()}
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

func listSlotCommandIssues(entry SlotEntry, declared map[string]Command) []error {
	var issues []error
	command, isDeclared := declared[entry.Command]
	if !isDeclared && !fullCommandIDPattern.MatchString(entry.Command) {
		issues = append(issues, fmt.Errorf("%w: command %q is neither declared by the manifest nor the full id of a command", ErrInvalidSlotEntry, entry.Command))
	}
	issues = append(issues, listSlotLookIssues(entry)...)
	if isDeclared {
		return append(issues, listSlotValueIssues(entry, command)...)
	}
	return append(issues, listForeignValueIssues(entry)...)
}

func listSlotEntryIssues(entry SlotEntry, declared map[string]Command) []error {
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
		issues = append(issues, listSlotCommandIssues(entry, declared)...)
	case entry.View != "":
		issues = append(issues, listUnplacedValuesIssues(entry, "view", entry.View)...)
		if !tabKindPattern.MatchString(entry.View) {
			issues = append(issues, fmt.Errorf("%w: view %q is not lowercase words", ErrInvalidSlotEntry, entry.View))
		}
	case entry.Submenu != "":
		issues = append(issues, listUnplacedValuesIssues(entry, "submenu", entry.Submenu)...)
		issues = append(issues, listSubmenuIssues(entry)...)
	default:
		issues = append(issues, fmt.Errorf("%w: the entry in %q places no command, view or submenu", ErrInvalidSlotEntry, entry.Slot))
	}
	return issues
}

func listSlotValueIssues(entry SlotEntry, command Command) []error {
	return wrapSlotIssues(ListDeclaredValueIssues(command, entry.Command, entry.Values))
}

// ListDeclaredValueIssues returns an error for each name of values that the command, run under the
// id named, does not declare as an argument.
func ListDeclaredValueIssues(command Command, named string, values map[string]string) []error {
	var issues []error
	for _, name := range slices.Sorted(maps.Keys(values)) {
		if _, isDeclared := command.GetArgument(name); !isDeclared {
			issues = append(issues, fmt.Errorf("values of command %q name the argument %q, which the command does not declare", named, name))
		}
	}
	return issues
}

func wrapSlotIssues(issues []error) []error {
	for i, issue := range issues {
		issues[i] = fmt.Errorf("%w: %w", ErrInvalidSlotEntry, issue)
	}
	return issues
}

func listSlotLookIssues(entry SlotEntry) []error {
	var issues []error
	if entry.Title != "" && strings.TrimSpace(entry.Title) == "" {
		issues = append(issues, fmt.Errorf("%w: command %q has a blank title", ErrInvalidSlotEntry, entry.Command))
	}
	if entry.Icon != "" && !iconNamePattern.MatchString(entry.Icon) {
		issues = append(issues, fmt.Errorf("%w: command %q has icon %q, which is not lowercase words joined by hyphens", ErrInvalidSlotEntry, entry.Command, entry.Icon))
	}
	return issues
}

func listForeignValueIssues(entry SlotEntry) []error {
	if len(entry.Values) == 0 || !strings.HasPrefix(entry.Command, "app:") {
		return nil
	}
	return wrapSlotIssues(ListAppValueIssues(entry.Command, entry.Values))
}

// ListAppValueIssues returns an error for each name of values that the app: command does not accept,
// and for each value of an enum argument that is not one of its options. The arguments are those
// commands.json declares for the app command and for the shared command of the id.
func ListAppValueIssues(command string, values map[string]string) []error {
	allowed := map[string][]string{}
	if appArguments, isAppCommand := GetAppCommandArguments(command); isAppCommand {
		for _, argument := range appArguments {
			if argument.Type == SettingTypeEnum {
				allowed[argument.Name] = argument.Options
			} else {
				allowed[argument.Name] = nil
			}
		}
	}
	for _, argument := range GetSharedCommandArguments(command) {
		allowed[argument.Name] = nil
	}
	var issues []error
	for _, name := range slices.Sorted(maps.Keys(values)) {
		options, isDeclared := allowed[name]
		switch {
		case !isDeclared:
			issues = append(issues, fmt.Errorf("values of command %q name the argument %q, which the command does not take", command, name))
		case len(options) > 0 && !slices.Contains(options, values[name]):
			issues = append(issues, fmt.Errorf("value %q of the argument %q of command %q is not one of %s", values[name], name, command, strings.Join(options, ", ")))
		}
	}
	return issues
}

func listUnplacedValuesIssues(entry SlotEntry, kind, id string) []error {
	if len(entry.Values) == 0 {
		return nil
	}
	return []error{fmt.Errorf("%w: %s %q holds values, and only a command takes them", ErrInvalidSlotEntry, kind, id)}
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
