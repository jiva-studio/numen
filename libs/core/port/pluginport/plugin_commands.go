package pluginport

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

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

// manifestCommandPattern is the shape of a command id and of an argument name: lowercase words
// joined by hyphens, as a command line spells a subcommand and a flag.
var manifestCommandPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

var commandArgumentTypes = []string{CommandArgumentString, CommandArgumentNumber, CommandArgumentBoolean}

// CommandArgumentVersion names the argument the host gives a command that has a When condition: the
// id of the text version that met the condition.
const CommandArgumentVersion = "version-id"

// PluginCommandWhen is the state of a file's text versions in which a command is offered over the
// file. Every condition a command names has to hold. A command with a condition is offered by the
// menu of the file, and not by the palette, which knows no file's versions.
type PluginCommandWhen struct {
	// LiveTextKinds are the kinds of text, as the plugin that made it names them, one of which the
	// file's live version has; `*` is any kind. The command is offered over that version.
	LiveTextKinds []domain.TextKind
	// HasOwnText is set when the file has a version the plugin of the command made. The command is
	// offered over each of them.
	HasOwnText bool
}

// PluginCommand is one command a manifest declares, which the plugin's backend runs.
type PluginCommand struct {
	ID    string
	Title string
	Icon  string
	// ContentTypes are the types of the files the command runs over; `*/*` is a file of any type.
	// A command that names no content type and no note type runs over no file.
	ContentTypes []domain.ContentType
	// NoteTypes are the plugin's own note types of the notes the command runs over.
	NoteTypes []string
	// When is the condition under which the menu of a file offers the command. A command with one
	// runs over any file, and names no content type and no note type.
	When      *PluginCommandWhen
	Arguments []PluginCommandArgument
	// IsHiddenInPalette keeps the command out of the window's palette where the window offers the
	// same thing another way. The command line still offers it.
	IsHiddenInPalette bool
}

// PluginCommandArgument is one argument a command takes.
type PluginCommandArgument struct {
	Name       string
	Title      string
	Type       string
	IsRequired bool
	// IsHiddenInPalette keeps an optional argument out of the window's palette, which runs the
	// command without it. The command line still takes it.
	IsHiddenInPalette bool
}

// HasFiles reports whether the command runs over a file.
func (c PluginCommand) HasFiles() bool {
	return len(c.ContentTypes) > 0 || len(c.NoteTypes) > 0 || c.When != nil
}

// CanRunOn reports whether the command runs over a file of the content type, or a note of the note
// type.
func (c PluginCommand) CanRunOn(contentType domain.ContentType, noteType domain.NoteType) bool {
	return c.When != nil || isAnyMatching(c.ContentTypes, contentType) || slices.Contains(c.NoteTypes, string(noteType))
}

// GetArgument returns the declaration of the argument of the name, and false where none is declared.
func (c PluginCommand) GetArgument(name string) (PluginCommandArgument, bool) {
	for _, one := range c.Arguments {
		if one.Name == name {
			return one, true
		}
	}
	return PluginCommandArgument{}, false
}

// listCommandIssues checks the commands, and that a plugin declaring any has a backend to run them.
func (m PluginManifest) listCommandIssues() []error {
	var issues []error
	if len(m.Commands) > 0 && m.Backend == (PluginBackend{}) {
		issues = append(issues, fmt.Errorf("%w: the plugin declares commands and has no backend to run them", ErrInvalidCommand))
	}
	observed := map[string]bool{}
	for _, command := range m.Commands {
		switch {
		case !manifestCommandPattern.MatchString(command.ID):
			issues = append(issues, fmt.Errorf("%w: id %q is not lowercase words joined by hyphens", ErrInvalidCommand, command.ID))
		case observed[command.ID]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", ErrInvalidCommand, command.ID))
		}
		observed[command.ID] = true
		if strings.TrimSpace(command.Title) == "" {
			issues = append(issues, fmt.Errorf("%w: %q has no title", ErrInvalidCommand, command.ID))
		}
		issues = append(issues, listContentTypesIssues(ErrInvalidCommand, command.ContentTypes)...)
		issues = append(issues, m.listUndeclaredNoteTypes(ErrInvalidCommand, command.NoteTypes)...)
		issues = append(issues, listCommandWhenIssues(command)...)
		issues = append(issues, listCommandArgumentIssues(command)...)
	}
	return issues
}

func listCommandWhenIssues(command PluginCommand) []error {
	when := command.When
	if when == nil {
		return nil
	}
	var issues []error
	if len(when.LiveTextKinds) == 0 && !when.HasOwnText {
		issues = append(issues, fmt.Errorf("%w: %q has a condition that names nothing", ErrInvalidCommand, command.ID))
	}
	if len(command.ContentTypes) > 0 || len(command.NoteTypes) > 0 {
		issues = append(issues, fmt.Errorf("%w: %q has a condition and names a content type or a note type", ErrInvalidCommand, command.ID))
	}
	for _, kind := range when.LiveTextKinds {
		if strings.TrimSpace(string(kind)) == "" {
			issues = append(issues, fmt.Errorf("%w: %q names a kind with no name", ErrInvalidCommand, command.ID))
		}
	}
	if _, isDeclared := command.GetArgument(CommandArgumentVersion); isDeclared {
		issues = append(issues, fmt.Errorf("%w: %q declares %q, which the host gives", ErrInvalidCommandArgument, command.ID, CommandArgumentVersion))
	}
	return issues
}

func listCommandArgumentIssues(command PluginCommand) []error {
	var issues []error
	observed := map[string]bool{}
	for _, argument := range command.Arguments {
		switch {
		case !manifestCommandPattern.MatchString(argument.Name):
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
		if !slices.Contains(commandArgumentTypes, argument.Type) {
			issues = append(issues, fmt.Errorf("%w: %q takes %q of type %q, expected one of %s", ErrInvalidCommandArgument, command.ID, argument.Name, argument.Type, strings.Join(commandArgumentTypes, ", ")))
		}
	}
	return issues
}

// PluginCommandEntry is one command an enabled plugin declares, and the plugin that does, by its id
// and by its name as a person reads it.
type PluginCommandEntry struct {
	PluginID   string
	PluginName string
	Command    PluginCommand
}

// PluginCommandCall is one run of a command, checked against its declaration.
type PluginCommandCall struct {
	PluginID string
	Command  string
	// Path is the file the command runs over, relative to the vault folder. Empty for a command over
	// no file.
	Path string
	// Arguments are the values by argument name, each as text.
	Arguments map[string]string
}

// PluginCommandResult is the result of a command that ran.
type PluginCommandResult struct {
	// Message is the outcome text to show to the user.
	Message string
	// OpenedPath is the file a window opens once the command ran, relative to the vault folder. Empty
	// for a command that leaves nothing to open.
	OpenedPath string
}

// CommandFailedError is a command the plugin ran and that failed, with the plugin's message.
type CommandFailedError struct {
	Message string
}

// Error returns the plugin's message.
func (e *CommandFailedError) Error() string {
	return e.Message
}

// PluginCommands is the commands the plugins declare and the backends that run them.
type PluginCommands interface {
	// ListCommands lists the commands of every enabled plugin, in the order of the plugins' ids and
	// then of each manifest.
	ListCommands() []PluginCommandEntry
	// RunCommand runs the command in the backend of its plugin. A plugin that cannot be started or
	// reached is an *UnavailableError, and a command that ran and failed a *CommandFailedError.
	RunCommand(ctx context.Context, call PluginCommandCall) (PluginCommandResult, error)
}
