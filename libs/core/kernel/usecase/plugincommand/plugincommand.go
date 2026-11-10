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

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// The ports' terms for a command, used by a driving adapter.
type (
	// Entry is one command an enabled plugin declares, and the plugin that does.
	Entry = pluginport.PluginCommandEntry
	// SlotEntry is one slot entry an enabled plugin declares, and the plugin that does.
	SlotEntry = pluginport.PluginSlotEntry
	// KeybindingEntry is one keybinding an enabled plugin declares, and the plugin that does.
	KeybindingEntry = pluginport.PluginKeybindingEntry
	// Parts is what one plugin has and whether the host holds it suspended.
	Parts = pluginport.PluginParts
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
	ErrUnavailable = domain.ErrUnavailable
)

// Commands is both scenarios over the commands the plugins declare.
type Commands struct {
	List List
	Run  Run
}

// NewCommands makes both scenarios over the commands, which may be nothing, the table that gives a
// file its type, the vault files a note's type is read from, and the settings a condition reads as
// config; without settings a condition over config is undecided and holds.
func NewCommands(
	commands pluginport.PluginCommands, types contenttypeport.ContentTypeTable, readers vaultport.VaultReaders,
	config pluginport.WhenConfig,
) Commands {
	conditions := pluginport.NewConditions()
	return Commands{
		List: List{commands: commands, types: types, conditions: conditions},
		Run:  Run{commands: commands, types: types, readers: readers, config: config, conditions: conditions},
	}
}

// List lists the commands the enabled plugins declare.
type List struct {
	commands   pluginport.PluginCommands
	types      contenttypeport.ContentTypeTable
	conditions *pluginport.PluginConditions
}

// Execute lists the commands that a plugin backend runs. Where there is no plugin host it lists none.
func (u List) Execute() []Entry {
	return slices.DeleteFunc(u.ExecuteAll(), func(one Entry) bool { return !u.commands.GetParts(one.PluginID).HasBackend })
}

// ExecuteAll lists every command the enabled plugins declare, whichever part of the plugin runs it.
// A plugin with neither a frontend nor a backend has nothing that runs its commands, and a plugin the
// host holds suspended runs none; both list none.
func (u List) ExecuteAll() []Entry {
	if u.commands == nil {
		return nil
	}
	return slices.DeleteFunc(u.commands.ListCommands(), func(one Entry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// ExecuteHandlers lists the handlers of shared commands of the enabled plugins, each as a command under
// the full id of its shared command, leaving out those of a plugin the host holds suspended and of a
// plugin with neither a frontend nor a backend.
func (u List) ExecuteHandlers() []Entry {
	if u.commands == nil {
		return nil
	}
	return slices.DeleteFunc(u.commands.ListHandlers(), func(one Entry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// GetParts returns the parts of the enabled plugin of the id, its name and the titles of its text kinds.
func (u List) GetParts(pluginID string) Parts {
	if u.commands == nil {
		return Parts{}
	}
	return u.commands.GetParts(pluginID)
}

// HasBackend reports whether the plugin has a backend that runs its commands and handlers.
func (u List) HasBackend(pluginID string) bool {
	return u.commands != nil && u.commands.GetParts(pluginID).HasBackend
}

// ExecuteSlots lists the slot entries of the enabled plugins, leaving out those of a plugin the host
// holds suspended and of a plugin with neither a frontend nor a backend.
func (u List) ExecuteSlots() []SlotEntry {
	if u.commands == nil {
		return nil
	}
	return slices.DeleteFunc(u.commands.ListSlots(), func(one SlotEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// ExecuteKeybindings lists the keybindings of the enabled plugins, leaving out those of a plugin the host
// holds suspended and of a plugin with neither a frontend nor a backend.
func (u List) ExecuteKeybindings() []KeybindingEntry {
	if u.commands == nil {
		return nil
	}
	return slices.DeleteFunc(u.commands.ListKeybindings(), func(one KeybindingEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
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
	commands   pluginport.PluginCommands
	types      contenttypeport.ContentTypeTable
	readers    vaultport.VaultReaders
	config     pluginport.WhenConfig
	conditions *pluginport.PluginConditions
}

// Execute checks the call against the declaration and runs it over the vault; a call of a shared
// command runs the handler the plugin declares. A call that does not meet the declaration is
// ErrBadArguments, saying what is wrong, and nothing is run.
func (u Run) Execute(ctx context.Context, v domain.Vault, call Call) (Result, error) {
	if u.commands == nil {
		return Result{}, ErrUnavailable
	}
	var entries []Entry
	if pluginport.IsSharedCommand(call.Command) {
		entries = u.commands.ListHandlers()
	} else {
		entries = u.commands.ListCommands()
	}
	if !u.commands.GetParts(call.PluginID).HasBackend {
		return Result{}, fmt.Errorf("%w: %s of %s", ErrUnknownCommand, call.Command, call.PluginID)
	}
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

// IsAvailable reports whether the condition of the command of the entry holds over the path, as a
// run of it checks; a condition the core cannot decide holds.
func (u Run) IsAvailable(ctx context.Context, v domain.Vault, entry Entry, path string) bool {
	return u.checkPath(ctx, v, entry.Command, Call{PluginID: entry.PluginID, Command: entry.Command.ID, Path: path}) == nil
}

func (u Run) checkCall(ctx context.Context, v domain.Vault, declared Command, call Call) error {
	if err := u.checkPath(ctx, v, declared, call); err != nil {
		return err
	}
	return checkArguments(declared, call, hasItems(u.conditions, declared))
}

func (u Run) checkPath(ctx context.Context, v domain.Vault, declared Command, call Call) error {
	condition, err := u.conditions.Parse(declared.When)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrBadArguments, declared.ID, err)
	}
	if condition.HasItems() && call.Path == "" {
		return fmt.Errorf("%w: %s needs a file", ErrBadArguments, declared.ID)
	}
	values, err := u.readNote(ctx, v, condition, pluginport.NewTargetValues(call.Path, false, u.getContentType(call.Path)), call.Path)
	if err != nil {
		return fmt.Errorf("read the note %s: %w", call.Path, err)
	}
	if u.config != nil && slices.Contains(condition.ListNames(), "config") {
		values = values.WithConfig(u.config.GetWhenConfig())
	}
	if condition.Evaluate(values.Get()) == pluginport.ConditionFalse {
		return fmt.Errorf("%w: %s is not available over %q, as its condition %q does not hold there",
			ErrBadArguments, declared.ID, call.Path, declared.When)
	}
	return nil
}

func checkArguments(declared Command, call Call, hasItems bool) error {
	for _, name := range slices.Sorted(maps.Keys(call.Arguments)) {
		if name == ArgumentVersion && hasItems {
			continue
		}
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

func (u Run) getContentType(path string) domain.ContentType {
	if u.types == nil {
		return domain.UnknownContentType
	}
	return u.types.GetContentType(path)
}

func (u Run) readNote(
	ctx context.Context, v domain.Vault, condition pluginport.PluginCondition, target pluginport.TargetValues, path string,
) (pluginport.TargetValues, error) {
	names := condition.ListNames()
	if (!slices.Contains(names, "target.noteType") && !slices.Contains(names, "target.properties")) || u.readers == nil || !target.IsNote() {
		return target, nil
	}
	noteType, properties, err := vaultport.ReadNote(ctx, u.readers, v, path)
	if err != nil {
		return target, err
	}
	return target.WithNote(noteType, properties), nil
}

// ListContentTypes returns the content type of each registered extension, lowercase and without the dot.
func (u List) ListContentTypes() map[string]string {
	out := map[string]string{}
	if u.types == nil {
		return out
	}
	for _, extension := range u.types.ListExtensions(domain.AnyContentType) {
		out[extension] = string(u.types.GetContentType("file." + extension))
	}
	return out
}

// HasItems reports whether the command offers one item per text version of a file: its condition
// names version.*. A condition that cannot be read counts as having items, so it is listed nowhere.
func (u List) HasItems(command Command) bool {
	return hasItems(u.conditions, command)
}

func hasItems(conditions *pluginport.PluginConditions, command Command) bool {
	condition, err := conditions.Parse(command.When)
	return err != nil || condition.HasItems()
}

// HasTargetVariable reports whether the condition of the command names the target it runs over, so a
// call names a path.
func (u List) HasTargetVariable(command Command) bool {
	condition, err := u.conditions.Parse(command.When)
	return err == nil && (condition.HasTargetVariable() || condition.HasItems())
}

// IsSharedCommand reports whether the full id is a shared command, which the handlers of plugins run.
func IsSharedCommand(id string) bool { return pluginport.IsSharedCommand(id) }
