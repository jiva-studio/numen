package pluginport

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
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

// PluginCommand is one command a manifest declares, which the plugin's backend runs.
type PluginCommand struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// FileExtensions are the extensions, without the dot, of the files the command runs over. None is
	// a command over no file.
	FileExtensions []string                `json:"fileExtensions,omitempty"`
	Arguments      []PluginCommandArgument `json:"arguments,omitempty"`
	// IsHiddenInPalette keeps the command out of the window's palette where the window offers the
	// same thing another way. The command line still offers it.
	IsHiddenInPalette bool `json:"hiddenInPalette,omitempty"`
}

// PluginCommandArgument is one argument a command takes.
type PluginCommandArgument struct {
	Name       string `json:"name"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	IsRequired bool   `json:"required,omitempty"`
}

// HasFiles reports whether the command runs over a file.
func (c PluginCommand) HasFiles() bool {
	return len(c.FileExtensions) > 0
}

// CanRunOn reports whether the command runs over the file at path, by its extension.
func (c PluginCommand) CanRunOn(path string) bool {
	dot := strings.LastIndexByte(path, '.')
	if dot < 0 || strings.ContainsRune(path[dot:], '/') {
		return false
	}
	return slices.Contains(c.FileExtensions, strings.ToLower(path[dot+1:]))
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
		for _, extension := range command.FileExtensions {
			if extension == "" || extension != strings.ToLower(extension) || strings.ContainsAny(extension, "./") {
				issues = append(issues, fmt.Errorf("%w: %q names extension %q, which is not lowercase without the dot", ErrInvalidCommand, command.ID, extension))
			}
		}
		issues = append(issues, listCommandArgumentIssues(command)...)
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
		if !slices.Contains(commandArgumentTypes, argument.Type) {
			issues = append(issues, fmt.Errorf("%w: %q takes %q of type %q, expected one of %s", ErrInvalidCommandArgument, command.ID, argument.Name, argument.Type, strings.Join(commandArgumentTypes, ", ")))
		}
	}
	return issues
}

// PluginCommandEntry is one command an enabled plugin declares, and the plugin that does.
type PluginCommandEntry struct {
	PluginID string
	Command  PluginCommand
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

// PluginCommandResult is what a command that ran answered.
type PluginCommandResult struct {
	// Message is what a person reads about the outcome.
	Message string
	// OpenedPath is the file a window opens once the command ran, relative to the vault folder. Empty
	// for a command that leaves nothing to open.
	OpenedPath string
}

// CommandFailedError is a command the plugin ran and that failed, with what the plugin said.
type CommandFailedError struct {
	Message string
}

// Error is what the plugin said.
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
