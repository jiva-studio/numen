package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
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

// manifestArgumentPattern is the shape of an argument name: lowercase words joined by hyphens, as a
// command line spells a flag.
var manifestArgumentPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// manifestCommandPattern is the shape of a command id: an object and an action, each in camelCase
// (a lowercase first letter, then letters and digits), with a dot between them (`cardModel.create`).
// The host files the command under the id of the plugin and this id.
var manifestCommandPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$`)

func listCommandArgumentTypes() []string {
	return []string{CommandArgumentString, CommandArgumentNumber, CommandArgumentBoolean}
}

// CommandArgumentVersion names the argument the host gives a command whose condition names version.*:
// the id of the text version it runs over.
const CommandArgumentVersion = "version-id"

// Command is one command a manifest declares, which the plugin's backend runs.
type Command struct {
	ID         string
	Title      string
	Icon       string
	When       string
	Permission string
	Arguments  []CommandArgument
}

// CommandArgument is one argument a command takes.
type CommandArgument struct {
	Name       string
	Title      string
	Type       string
	IsRequired bool
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
		issues = append(issues, listCommandWhenIssues(command)...)
		issues = append(issues, m.listCommandPermissionIssues(command)...)
		issues = append(issues, listCommandArgumentIssues(command)...)
	}
	return issues
}

func listCommandWhenIssues(command Command) []error {
	condition, err := when.Parse(command.When)
	if err != nil {
		return []error{fmt.Errorf("%w: %q: when: %w", ErrInvalidCommand, command.ID, err)}
	}
	if _, isDeclared := command.GetArgument(CommandArgumentVersion); isDeclared && condition.HasItems() {
		return []error{fmt.Errorf("%w: %q declares %q, which the host gives", ErrInvalidCommandArgument, command.ID, CommandArgumentVersion)}
	}
	return nil
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
		if types := listCommandArgumentTypes(); !slices.Contains(types, argument.Type) {
			issues = append(issues, fmt.Errorf("%w: %q takes %q of type %q, expected one of %s", ErrInvalidCommandArgument, command.ID, argument.Name, argument.Type, strings.Join(types, ", ")))
		}
	}
	return issues
}

func (m Manifest) listCommandPermissionIssues(command Command) []error {
	if command.Permission == "" {
		return nil
	}
	if !slices.Contains(PluginPermissions(), command.Permission) {
		return []error{fmt.Errorf("%w: %q: %w: %s", ErrInvalidCommand, command.ID, ErrInvalidPermission, command.Permission)}
	}
	if !m.HasPermission(command.Permission) {
		return []error{fmt.Errorf("%w: %q: %w: %s is missing from permissions", ErrInvalidCommand, command.ID, ErrInvalidPermission, command.Permission)}
	}
	return nil
}
