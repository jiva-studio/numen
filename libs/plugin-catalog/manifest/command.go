package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// CommandScopeNote is the scope of a command listed only while a note is in front.
const CommandScopeNote = "note"

// The types an argument of a command holds.
const (
	CommandArgumentString  = "string"
	CommandArgumentNumber  = "number"
	CommandArgumentBoolean = "boolean"
)

var (
	ErrInvalidCommand         = errors.New("invalid command")
	ErrInvalidCommandArgument = errors.New("invalid command argument")
)

// manifestArgumentPattern is the shape of an argument name: lowercase words joined by hyphens, as a
// command line spells a flag.
var manifestArgumentPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// manifestCommandPattern is the shape of a command id: an object and an action, each in camelCase
// (a lowercase first letter, then letters and digits), with a dot between them (`cardModel.create`).
// The host files the command under the id of the plugin and this id.
var manifestCommandPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$`)

// listCommandArgumentTypes returns the types a command argument may have.
func listCommandArgumentTypes() []string {
	return []string{CommandArgumentString, CommandArgumentNumber, CommandArgumentBoolean}
}

// CommandArgumentVersion names the argument the host gives a command that has a When condition: the
// id of the context the command is offered over.
const CommandArgumentVersion = "version-id"

// ContextAnyValue is the value of a condition that any value of the key meets.
const ContextAnyValue = "*"

// Context is one subject a command can be offered over, described by named keys. The application
// that offers the command supplies the keys and their values.
type Context map[string]string

// ContextCondition holds in a context that has the key with one of the values listed.
type ContextCondition struct {
	Key string
	// Is are the values of the key that meet the condition; ContextAnyValue is any value.
	Is []string
}

// IsMet reports whether the context has the key with one of the values of the condition.
func (c ContextCondition) IsMet(context Context) bool {
	value, isSet := context[c.Key]
	return isSet && (slices.Contains(c.Is, ContextAnyValue) || slices.Contains(c.Is, value))
}

// Command is one command a manifest declares, which the plugin's backend runs.
type Command struct {
	ID    string
	Title string
	Icon  string
	// ContentTypes are the types of the files the command runs over; `*/*` is a file of any type.
	// A command that names no content type and no note type runs over no file.
	ContentTypes []domain.ContentType
	// NoteTypes are the plugin's own note types of the notes the command runs over.
	NoteTypes []string
	// When are the conditions, all of which hold in one context, under which the menu of a file offers
	// the command over that context. A command with conditions runs over any file, and names no
	// content type and no note type. The palette has no file context and does not offer it.
	When      []ContextCondition
	Arguments []CommandArgument
	// Scope is CommandScopeNote for a command listed only while a note is in front. Empty is a
	// command over the window.
	Scope string
	// Shortcut is the keystroke the plugin suggests for the command, such as "mod+shift+e".
	Shortcut string
}

// CommandArgument is one argument a command takes.
type CommandArgument struct {
	Name       string
	Title      string
	Type       string
	IsRequired bool
	// IsHiddenInPalette keeps an optional argument out of the window's palette, which runs the
	// command without it. The command line still takes it.
	IsHiddenInPalette bool
}

// HasFiles reports whether the command runs over a file.
func (c Command) HasFiles() bool {
	return len(c.ContentTypes) > 0 || len(c.NoteTypes) > 0 || len(c.When) > 0
}

// CanRunIn reports whether every condition of the command holds in the context. A command with no
// condition holds in every context.
func (c Command) CanRunIn(context Context) bool {
	return !slices.ContainsFunc(c.When, func(condition ContextCondition) bool { return !condition.IsMet(context) })
}

// CanRunOn reports whether the command runs over a file of the content type, or a note of the note
// type.
func (c Command) CanRunOn(contentType domain.ContentType, noteType domain.NoteType) bool {
	return len(c.When) > 0 || isAnyMatching(c.ContentTypes, contentType) || slices.Contains(c.NoteTypes, string(noteType))
}

// GetArgument returns the declaration of the argument of the name, and false where none is declared.
func (c Command) GetArgument(name string) (CommandArgument, bool) {
	for _, one := range c.Arguments {
		if one.Name == name {
			return one, true
		}
	}
	return CommandArgument{}, false
}

// listCommandIssues checks the commands the manifest declares.
func (m Manifest) listCommandIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, command := range m.Contributes.Commands {
		switch {
		case !manifestCommandPattern.MatchString(command.ID):
			issues = append(issues, fmt.Errorf("%w: id %q is not an object and an action, each in camelCase, with a dot between them", ErrInvalidCommand, command.ID))
		case observed[command.ID]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", ErrInvalidCommand, command.ID))
		}
		observed[command.ID] = true
		if strings.TrimSpace(command.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: %q has no title", ErrInvalidCommand, command.ID))
		}
		if m.Backend != (Backend{}) && !m.HasFrontend() && (command.Scope != "" || command.Shortcut != "") {
			issues = append(issues, fmt.Errorf("%w: %q declares a scope or a shortcut, and its plugin has no frontend to run it", ErrInvalidCommand, command.ID))
		}
		if command.Scope != "" && command.Scope != CommandScopeNote {
			issues = append(issues, fmt.Errorf("%w: %q has scope %q, which is none of the scopes %q", ErrInvalidCommand, command.ID, command.Scope, CommandScopeNote))
		}
		issues = append(issues, listContentTypesIssues(ErrInvalidCommand, command.ContentTypes)...)
		issues = append(issues, m.listUndeclaredNoteTypes(ErrInvalidCommand, command.NoteTypes)...)
		issues = append(issues, listCommandWhenIssues(command)...)
		issues = append(issues, listCommandArgumentIssues(command)...)
	}
	return issues
}

func listCommandWhenIssues(command Command) []error {
	if len(command.When) == 0 {
		return nil
	}
	var issues []error
	if len(command.ContentTypes) > 0 || len(command.NoteTypes) > 0 {
		issues = append(issues, fmt.Errorf("%w: %q has a condition and names a content type or a note type", ErrInvalidCommand, command.ID))
	}
	for _, condition := range command.When {
		if strings.TrimSpace(condition.Key) == "" {
			issues = append(issues, fmt.Errorf("%w: %q has a condition with no key", ErrInvalidCommand, command.ID))
		}
		if len(condition.Is) == 0 {
			issues = append(issues, fmt.Errorf("%w: %q has a condition on %q that names no value", ErrInvalidCommand, command.ID, condition.Key))
		}
		for _, value := range condition.Is {
			if strings.TrimSpace(value) == "" {
				issues = append(issues, fmt.Errorf("%w: %q has a condition on %q with an empty value", ErrInvalidCommand, command.ID, condition.Key))
			}
		}
	}
	if _, isDeclared := command.GetArgument(CommandArgumentVersion); isDeclared {
		issues = append(issues, fmt.Errorf("%w: %q declares %q, which the host gives", ErrInvalidCommandArgument, command.ID, CommandArgumentVersion))
	}
	return issues
}

func listCommandArgumentIssues(command Command) []error {
	var issues []error
	observed := map[string]bool{}
	for _, argument := range command.Arguments {
		switch {
		case !manifestArgumentPattern.MatchString(argument.Name):
			issues = append(issues, fmt.Errorf("%w: %q takes %q, which is not lowercase words joined by hyphens", ErrInvalidCommandArgument, command.ID, argument.Name))
		case observed[argument.Name]:
			issues = append(issues, fmt.Errorf("%w: %q takes %q twice", ErrInvalidCommandArgument, command.ID, argument.Name))
		}
		observed[argument.Name] = true
		if strings.TrimSpace(argument.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: %q takes %q with no title", ErrInvalidCommandArgument, command.ID, argument.Name))
		}
		if argument.IsHiddenInPalette && argument.IsRequired {
			issues = append(issues, fmt.Errorf("%w: %q takes %q, which is required and hidden in the palette", ErrInvalidCommandArgument, command.ID, argument.Name))
		}
		if types := listCommandArgumentTypes(); !slices.Contains(types, argument.Type) {
			issues = append(issues, fmt.Errorf("%w: %q takes %q of type %q, expected one of %s", ErrInvalidCommandArgument, command.ID, argument.Name, argument.Type, strings.Join(types, ", ")))
		}
	}
	return issues
}
