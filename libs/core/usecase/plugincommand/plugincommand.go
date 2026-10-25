// Package plugincommand holds the scenarios over the commands the plugins declare: listing them and
// running one, checked against its declaration.
package plugincommand

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// The ports' words for a command, which a driving adapter reaches here.
type (
	// Entry is one command an enabled plugin declares, and the plugin that does.
	Entry = pluginport.PluginCommandEntry
	// Command is a command as its manifest declares it.
	Command = pluginport.PluginCommand
	// Argument is one argument a command takes.
	Argument = pluginport.PluginCommandArgument
	// Call is one run of a command.
	Call = pluginport.PluginCommandCall
	// Result is what a command that ran answered.
	Result = pluginport.PluginCommandResult
)

// The types an argument holds.
const (
	ArgumentNumber  = pluginport.CommandArgumentNumber
	ArgumentBoolean = pluginport.CommandArgumentBoolean
)

var (
	// ErrUnknownCommand is a command no enabled plugin declares.
	ErrUnknownCommand = errors.New("no enabled plugin declares that command")
	// ErrBadArguments is arguments or a file that do not meet the declaration.
	ErrBadArguments = errors.New("the arguments do not meet the command")
	// ErrUnavailable is a plugin that cannot be started or reached, or an installation with no plugin
	// host.
	ErrUnavailable = completionport.ErrUnavailable
)

// Commands is both scenarios over the commands the plugins declare.
type Commands struct {
	List List
	Run  Run
}

// NewCommands makes both scenarios over the commands, which may be nothing.
func NewCommands(commands pluginport.PluginCommands) Commands {
	return Commands{List: List{commands: commands}, Run: Run{commands: commands}}
}

// List lists the commands the enabled plugins declare.
type List struct {
	commands pluginport.PluginCommands
}

// Execute lists them. Where there is no plugin host it lists none.
func (u List) Execute() []Entry {
	if u.commands == nil {
		return nil
	}
	return u.commands.ListCommands()
}

// Find returns the command of the id, by the plugin of the lowest id that declares one, and false
// where none does.
func (u List) Find(id string) (Entry, bool) {
	for _, one := range u.Execute() {
		if one.Command.ID == id {
			return one, true
		}
	}
	return Entry{}, false
}

// Run runs one command in the backend of the plugin that declares it.
type Run struct {
	commands pluginport.PluginCommands
}

// Execute checks the call against the declaration and runs it. A call that does not meet the declaration is ErrBadArguments, saying what is wrong,
// and nothing is run.
func (u Run) Execute(ctx context.Context, call Call) (Result, error) {
	if u.commands == nil {
		return Result{}, ErrUnavailable
	}
	listed := u.commands.ListCommands()
	index := slices.IndexFunc(listed, func(one Entry) bool {
		return one.PluginID == call.PluginID && one.Command.ID == call.Command
	})
	if index < 0 {
		return Result{}, fmt.Errorf("%w: %s of %s", ErrUnknownCommand, call.Command, call.PluginID)
	}
	declared := listed[index].Command
	if err := checkCall(declared, call); err != nil {
		return Result{}, err
	}
	normalized := call
	normalized.Arguments = normalizeArguments(declared, call.Arguments)
	return u.commands.RunCommand(ctx, normalized)
}

// checkCall reports the first way the call does not meet the declaration.
func checkCall(declared Command, call Call) error {
	switch {
	case declared.HasFiles() && call.Path == "":
		return fmt.Errorf("%w: %s runs over a file", ErrBadArguments, declared.ID)
	case declared.HasFiles() && !declared.CanRunOn(call.Path):
		return fmt.Errorf("%w: %s runs over files of %v, and %s is none", ErrBadArguments, declared.ID, declared.FileExtensions, call.Path)
	case !declared.HasFiles() && call.Path != "":
		return fmt.Errorf("%w: %s runs over no file", ErrBadArguments, declared.ID)
	}
	for _, name := range slices.Sorted(maps.Keys(call.Arguments)) {
		argument, isDeclared := declared.GetArgument(name)
		if !isDeclared {
			return fmt.Errorf("%w: %s takes no argument %q", ErrBadArguments, declared.ID, name)
		}
		if !isOfType(argument.Type, call.Arguments[name]) {
			return fmt.Errorf("%w: %s is a %s, and %q is none", ErrBadArguments, argument.Name, argument.Type, call.Arguments[name])
		}
	}
	for _, argument := range declared.Arguments {
		if _, isGiven := call.Arguments[argument.Name]; argument.IsRequired && !isGiven {
			return fmt.Errorf("%w: %s needs %s", ErrBadArguments, declared.ID, argument.Name)
		}
	}
	return nil
}

func isOfType(argumentType, value string) bool {
	switch argumentType {
	case ArgumentNumber:
		_, err := strconv.ParseFloat(value, 64)
		return err == nil
	case ArgumentBoolean:
		_, err := strconv.ParseBool(value)
		return err == nil
	}
	return true
}

// normalizeArguments writes a boolean as true or false, however it was spelled.
func normalizeArguments(declared Command, arguments map[string]string) map[string]string {
	normalized := make(map[string]string, len(arguments))
	for name, value := range arguments {
		if argument, _ := declared.GetArgument(name); argument.Type == ArgumentBoolean {
			parsed, _ := strconv.ParseBool(value)
			value = strconv.FormatBool(parsed)
		}
		normalized[name] = value
	}
	return normalized
}
