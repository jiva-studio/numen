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
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
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

// NewCommands makes both scenarios over the commands, which may be nothing, the table that gives a
// file its type, and the vault files a note's type is read from.
func NewCommands(
	commands pluginport.PluginCommands, types contenttypeport.ContentTypeTable, readers vaultport.VaultReaders,
) Commands {
	return Commands{
		List: List{commands: commands, types: types},
		Run:  Run{commands: commands, types: types, readers: readers},
	}
}

// List lists the commands the enabled plugins declare.
type List struct {
	commands pluginport.PluginCommands
	types    contenttypeport.ContentTypeTable
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
	types    contenttypeport.ContentTypeTable
	readers  vaultport.VaultReaders
}

// Execute checks the call against the declaration and runs it over the vault. A call that does not
// meet the declaration is ErrBadArguments, saying what is wrong, and nothing is run.
func (u Run) Execute(ctx context.Context, v domain.Vault, call Call) (Result, error) {
	if u.commands == nil {
		return Result{}, ErrUnavailable
	}
	entries := u.commands.ListCommands()
	index := slices.IndexFunc(entries, func(one Entry) bool {
		return one.PluginID == call.PluginID && one.Command.ID == call.Command
	})
	if index < 0 {
		return Result{}, fmt.Errorf("%w: %s of %s", ErrUnknownCommand, call.Command, call.PluginID)
	}
	declared := entries[index].Command
	if err := u.checkCall(ctx, v, declared, call); err != nil {
		return Result{}, err
	}
	normalized := call
	normalized.Arguments = normalizeArguments(declared, call.Arguments)
	return u.commands.RunCommand(ctx, normalized)
}

// checkCall reports the first way the call does not meet the declaration.
func (u Run) checkCall(ctx context.Context, v domain.Vault, declared Command, call Call) error {
	switch {
	case declared.HasFiles() && (call.Path == "" || strings.HasSuffix(call.Path, "/")):
		return fmt.Errorf("%w: %s runs over a file", ErrBadArguments, declared.ID)
	case declared.HasFiles() && !declared.CanRunOn(u.getContentType(call.Path), u.readNoteType(ctx, v, declared, call.Path)):
		return fmt.Errorf("%w: %s runs over files of %v and notes of types %v, and %s is none",
			ErrBadArguments, declared.ID, declared.ContentTypes, declared.NoteTypes, call.Path)
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
			parsed, err := strconv.ParseBool(value)
			value = strconv.FormatBool(err == nil && parsed)
		}
		normalized[name] = value
	}
	return normalized
}

// getContentType is the type of the file at the path, unknown where there is no table.
func (u Run) getContentType(path string) domain.ContentType {
	if u.types == nil {
		return domain.UnknownContentType
	}
	return u.types.GetContentType(path)
}

// readNoteType is the type of the note at the path where the command runs over notes of some type,
// and no type where it runs over none, where the vault has no readers, or where the note cannot be
// read.
func (u Run) readNoteType(ctx context.Context, v domain.Vault, declared Command, path string) domain.NoteType {
	if len(declared.NoteTypes) == 0 || u.readers == nil {
		return ""
	}
	noteType, err := vaultport.ReadNoteType(ctx, u.readers, v, path)
	if err != nil {
		return ""
	}
	return noteType
}

// ListExtensions lists, lowercase and without the dot, the extensions that give a file a type the
// command runs over, and `*` alone where it runs over a file of any type.
func (u List) ListExtensions(command Command) []string {
	if slices.Contains(command.ContentTypes, domain.AnyContentType) {
		return []string{"*"}
	}
	var out []string
	for _, one := range command.ContentTypes {
		if u.types != nil {
			out = append(out, u.types.ListExtensions(one)...)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
