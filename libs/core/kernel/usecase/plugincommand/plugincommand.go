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
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// NoteReader reads the type and the frontmatter of a note from its current file.
type NoteReader interface {
	Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteType, map[string]any, error)
}

// VaultFinding resolves what the user typed, a name, a path or an identity, into a vault on the list.
type VaultFinding = func(nameOrPath string) (domain.Vault, error)

// RunnerOpener opens the command runner over a vault.
type RunnerOpener = func(ctx context.Context, v domain.Vault) (Run, func() error, error)

var (
	// ErrUnknownCommand is a command no enabled plugin declares.
	ErrUnknownCommand = errors.New("no enabled plugin declares that command")
	// ErrBadArguments is arguments or a file that do not meet the declaration.
	ErrBadArguments = errors.New("the arguments do not meet the command")
)

// NewList makes the scenario that lists the commands the plugins declare. conditions parses the condition of
// a command; it is shared with the scenarios that run commands, so one condition is parsed once.
func NewList(commands pluginport.PluginCommands, conditions *when.Conditions) List {
	return List{commands: commands, conditions: conditions}
}

// NewRun makes the scenario that runs a command in the backend that declares it, over the backends that run
// them, which may be nothing, the table that gives a file its type, the reader of the type and properties of
// a note, and the settings a condition reads as config; without settings a condition over config is
// undecided and holds.
func NewRun(
	commands pluginport.PluginCommands, runs pluginport.PluginCommandRuns, types contenttypeport.ContentTypeTable,
	notes NoteReader, config pluginport.WhenConfig, conditions *when.Conditions,
) Run {
	return Run{commands: commands, runs: runs, types: types, notes: notes, config: config, conditions: conditions}
}

// List lists the commands the enabled plugins declare.
type List struct {
	commands   pluginport.PluginCommands
	conditions *when.Conditions
}

// Execute lists the commands that a plugin backend runs. Where there is no plugin host it lists none.
func (u List) Execute() []plugin.CommandEntry {
	return slices.DeleteFunc(u.ExecuteAll(), func(one plugin.CommandEntry) bool { return !u.commands.GetParts(one.PluginID).HasBackend })
}

// ExecuteAll lists every command the enabled plugins declare, whichever part of the plugin runs it.
// A plugin with neither a frontend nor a backend has nothing that runs its commands, and a plugin the
// host holds suspended runs none; both list none.
func (u List) ExecuteAll() []plugin.CommandEntry {
	if u.commands == nil {
		return nil
	}
	return slices.DeleteFunc(u.commands.ListCommands(), func(one plugin.CommandEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// ExecuteHandlers lists the handlers of shared commands of the enabled plugins, each as a command under
// the full id of its shared command, leaving out those of a plugin the host holds suspended and of a
// plugin with neither a frontend nor a backend.
func (u List) ExecuteHandlers() []plugin.CommandEntry {
	if u.commands == nil {
		return nil
	}
	return slices.DeleteFunc(u.commands.ListHandlers(), func(one plugin.CommandEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

func getParts(commands pluginport.PluginCommands, pluginID string) plugin.Parts {
	if commands == nil {
		return plugin.Parts{}
	}
	return commands.GetParts(pluginID)
}

func hasBackend(commands pluginport.PluginCommands, pluginID string) bool {
	return commands != nil && commands.GetParts(pluginID).HasBackend
}

// ExecuteSlots lists the slot entries of the enabled plugins, leaving out those of a plugin the host
// holds suspended and of a plugin with neither a frontend nor a backend.
func (u List) ExecuteSlots() []plugin.SlotEntry {
	if u.commands == nil {
		return nil
	}
	return slices.DeleteFunc(u.commands.ListSlots(), func(one plugin.SlotEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// ExecuteKeybindings lists the keybindings of the enabled plugins, leaving out those of a plugin the host
// holds suspended and of a plugin with neither a frontend nor a backend.
func (u List) ExecuteKeybindings() []plugin.KeybindingEntry {
	if u.commands == nil {
		return nil
	}
	return slices.DeleteFunc(u.commands.ListKeybindings(), func(one plugin.KeybindingEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// Find returns the command of the id, by the plugin of the lowest id that declares one, and false
// where none does.
func (u List) Find(id string) (plugin.CommandEntry, bool) {
	for _, one := range u.Execute() {
		if one.Command.ID == id {
			return one, true
		}
	}
	return plugin.CommandEntry{}, false
}

// Run runs one command in the backend of the plugin that declares it.
type Run struct {
	commands   pluginport.PluginCommands
	runs       pluginport.PluginCommandRuns
	types      contenttypeport.ContentTypeTable
	notes      NoteReader
	config     pluginport.WhenConfig
	conditions *when.Conditions
}

// Execute checks the call against the declaration and runs it over the vault; a call of a shared
// command runs the handler the plugin declares. A call that does not meet the declaration is
// ErrBadArguments, saying what is wrong, and nothing is run.
func (u Run) Execute(ctx context.Context, v domain.Vault, call plugin.CommandCall) (plugin.CommandResult, error) {
	if u.commands == nil || u.runs == nil {
		return plugin.CommandResult{}, domain.ErrUnavailable
	}
	var entries []plugin.CommandEntry
	if manifest.IsSharedCommand(call.Command) {
		entries = u.commands.ListHandlers()
	} else {
		entries = u.commands.ListCommands()
	}
	if !u.commands.GetParts(call.PluginID).HasBackend {
		return plugin.CommandResult{}, fmt.Errorf("%w: %s of %s", ErrUnknownCommand, call.Command, call.PluginID)
	}
	index := slices.IndexFunc(entries, func(one plugin.CommandEntry) bool {
		return one.PluginID == call.PluginID && one.Command.ID == call.Command
	})
	if index < 0 {
		return plugin.CommandResult{}, fmt.Errorf("%w: %s of %s", ErrUnknownCommand, call.Command, call.PluginID)
	}
	declared := entries[index].Command
	if err := u.checkCall(ctx, v, declared, call); err != nil {
		return plugin.CommandResult{}, err
	}
	normalized := call
	normalized.Arguments = normalizeArguments(declared, call.Arguments)
	return u.runs.RunCommand(ctx, normalized)
}

// IsAvailable reports whether the condition of the command of the entry holds over the path, as a
// run of it checks; a condition the core cannot decide holds.
func (u Run) IsAvailable(ctx context.Context, v domain.Vault, entry plugin.CommandEntry, path string) bool {
	return u.checkPath(ctx, v, entry.Command, plugin.CommandCall{PluginID: entry.PluginID, Command: entry.Command.ID, Path: path}) == nil
}

func (u Run) checkCall(ctx context.Context, v domain.Vault, declared manifest.Command, call plugin.CommandCall) error {
	if err := u.checkPath(ctx, v, declared, call); err != nil {
		return err
	}
	return checkArguments(declared, call, hasItems(u.conditions, declared))
}

func (u Run) checkPath(ctx context.Context, v domain.Vault, declared manifest.Command, call plugin.CommandCall) error {
	condition, err := u.conditions.Parse(declared.When)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrBadArguments, declared.ID, err)
	}
	if condition.HasItems() && call.Path == "" {
		return fmt.Errorf("%w: %s needs a file", ErrBadArguments, declared.ID)
	}
	values, err := u.readNote(ctx, v, condition, plugin.NewTargetValues(call.Path, false, u.getContentType(call.Path)), call.Path)
	if err != nil {
		return fmt.Errorf("read the note %s: %w", call.Path, err)
	}
	if u.config != nil && slices.Contains(condition.ListNames(), "config") {
		values = values.WithConfig(u.config.GetWhenConfig())
	}
	if condition.Evaluate(values.Get()) == when.False {
		return fmt.Errorf("%w: %s is not available over %q, as its condition %q does not hold there",
			ErrBadArguments, declared.ID, call.Path, declared.When)
	}
	return nil
}

func checkArguments(declared manifest.Command, call plugin.CommandCall, hasItems bool) error {
	for _, name := range slices.Sorted(maps.Keys(call.Arguments)) {
		if name == manifest.CommandArgumentVersion && hasItems {
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
	case manifest.CommandArgumentNumber:
		_, err := strconv.ParseFloat(value, 64)
		return err == nil
	case manifest.CommandArgumentBoolean:
		_, err := strconv.ParseBool(value)
		return err == nil
	}
	return true
}

func normalizeArguments(declared manifest.Command, arguments map[string]string) map[string]string {
	normalized := make(map[string]string, len(arguments))
	for name, value := range arguments {
		if argument, _ := declared.GetArgument(name); argument.Type == manifest.CommandArgumentBoolean {
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
	ctx context.Context, v domain.Vault, condition when.Condition, target plugin.TargetValues, path string,
) (plugin.TargetValues, error) {
	names := condition.ListNames()
	if (!slices.Contains(names, "target.noteType") && !slices.Contains(names, "target.properties")) || u.notes == nil || !target.IsNote() {
		return target, nil
	}
	noteType, properties, err := u.notes.Execute(ctx, v, path)
	if err != nil {
		return target, err
	}
	return target.WithNote(noteType, properties), nil
}

func hasItems(conditions *when.Conditions, command manifest.Command) bool {
	condition, err := conditions.Parse(command.When)
	return err != nil || condition.HasItems()
}

// HasTargetVariable reports whether the condition of the command names the target it runs over, so a
// call names a path.
func (u List) HasTargetVariable(command manifest.Command) bool {
	condition, err := u.conditions.Parse(command.When)
	return err == nil && (condition.HasTargetVariable() || condition.HasItems())
}

// ContentTypes lists the content type of each file extension the table registers.
type ContentTypes struct {
	types contenttypeport.ContentTypeTable
}

// Execute returns the content type of each registered extension, lowercase and without the dot.
func (u ContentTypes) Execute() map[string]string {
	out := map[string]string{}
	if u.types == nil {
		return out
	}
	for _, extension := range u.types.ListExtensions(domain.AnyContentType) {
		out[extension] = string(u.types.GetContentType("file." + extension))
	}
	return out
}
