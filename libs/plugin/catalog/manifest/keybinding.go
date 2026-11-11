package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

// ErrInvalidKeybinding is an entry of contributes.keybindings a manifest declares wrongly.
var ErrInvalidKeybinding = errors.New("invalid keybinding")

// keyPattern is a chord: modifiers joined by plus signs, then one key name, such as mod+shift+p or n.
var keyPattern = regexp.MustCompile(
	`^((mod|ctrl|alt|shift|meta)\+)*([a-z0-9]|f([1-9]|1[0-2])|enter|escape|tab|space|backspace|delete|home|end|pageup|pagedown|up|down|left|right|comma|period|slash|backslash|semicolon|quote|minus|equal|backquote|bracketleft|bracketright)$`,
)

// Keybinding binds a key to a command, with values for its arguments, under a condition.
type Keybinding struct {
	Key     string
	Command string
	Values  map[string]string
	When    string
}

func (m Manifest) listKeybindingIssues() []error {
	declared := map[string]Command{}
	for _, command := range m.Contributes.Commands {
		declared[command.ID] = command
	}
	var issues []error
	type binding struct{ key, when string }
	observed := map[binding]bool{}
	for _, entry := range m.Contributes.Keybindings {
		issues = append(issues, listKeybindingEntryIssues(entry, declared)...)
		key := binding{entry.Key, strings.TrimSpace(entry.When)}
		if observed[key] {
			issues = append(issues, fmt.Errorf("%w: key %q is bound twice under the same condition", ErrInvalidKeybinding, entry.Key))
		}
		observed[key] = true
	}
	return issues
}

func listKeybindingEntryIssues(entry Keybinding, declared map[string]Command) []error {
	var issues []error
	if !keyPattern.MatchString(entry.Key) {
		issues = append(
			issues,
			fmt.Errorf("%w: key %q is not modifiers (mod, ctrl, alt, shift, meta) and one key name joined by plus signs", ErrInvalidKeybinding, entry.Key),
		)
	}
	if _, err := when.Parse(entry.When); err != nil {
		issues = append(issues, fmt.Errorf("%w: key %q: when: %w", ErrInvalidKeybinding, entry.Key, err))
	}
	command, isDeclared := declared[entry.Command]
	if !isDeclared && !fullCommandIDPattern.MatchString(entry.Command) {
		return append(
			issues,
			fmt.Errorf(
				"%w: key %q: command %q is neither declared by the manifest nor the full id of a command",
				ErrInvalidKeybinding,
				entry.Key,
				entry.Command,
			),
		)
	}
	target := SlotEntry{Command: entry.Command, Values: entry.Values}
	if isDeclared {
		return append(issues, listSlotValueIssues(target, command)...)
	}
	return append(issues, listForeignValueIssues(target)...)
}
