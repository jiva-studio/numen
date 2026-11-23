// Package commandrun runs a command of a plugin checked against its declaration: its backend, its
// condition and its arguments. It is the guarded runner the scenarios over commands, triggers and text
// versions share, not a scenario of its own.
package commandrun

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Commands is what the manifests of the enabled plugins declare.
type Commands interface {
	// ListCommands lists the commands of every enabled plugin.
	ListCommands() []plugin.CommandEntry
	// ListHandlers lists the handlers of shared commands of every enabled plugin, each as a command under
	// the full id of its shared command.
	ListHandlers() []plugin.CommandEntry
	// GetParts says what the plugin of the id has and whether it is suspended.
	GetParts(pluginID string) plugin.Parts
}

// Runs runs a command in the backend of its plugin.
type Runs interface {
	// RunCommand runs the command in the backend of its plugin. A plugin that cannot be started or
	// reached is an *UnavailableError, and a command that ran and failed a *CommandFailedError.
	RunCommand(ctx context.Context, call plugin.CommandCall) (plugin.CommandResult, error)
}

// ContentTypes gives a file its type by its name.
type ContentTypes interface {
	GetContentType(name string) domain.ContentType
}

// Config gives the config a condition reads.
type Config interface {
	GetWhenConfig() map[string]any
}

// OpenReader opens a reader of the files of a vault.
type OpenReader = func(v domain.Vault) (edit.Reader, error)

// New makes the runner over the manifests commands, the backends runs, which may run nothing, the table
// types that gives a file its type, the reader that open gives for the type and properties of a note, and
// the settings a condition reads as config. conditions parses the condition of a command. Without open, or
// without config, a condition over the note or over config is undecided and holds.
func New(commands Commands, runs Runs, types ContentTypes, open OpenReader, config Config, conditions *when.Conditions) Runner {
	run := Runner{commands: commands, runs: runs, types: types, notes: noNotes{}, config: noConfig{}, conditions: conditions}
	if open != nil {
		run.notes = readingNotes{open}
	}
	if config != nil {
		run.config = givenConfig{config}
	}
	return run
}

// HasItems reports whether the command offers items, or its condition cannot be parsed.
func HasItems(conditions *when.Conditions, command manifest.Command) bool {
	condition, err := conditions.Parse(command.When)
	return err != nil || condition.HasItems()
}

// Runner runs one command in the backend of the plugin that declares it.
type Runner struct {
	commands   Commands
	runs       Runs
	types      ContentTypes
	notes      targetNotes
	config     targetConfig
	conditions *when.Conditions
}

// Run checks the call against the declaration and runs it over the vault; a call of a shared
// command runs the handler the plugin declares. A call that does not meet the declaration is
// plugin.ErrBadArguments, saying what is wrong, and nothing is run. A call over Paths meets it where each of
// them does; a path of Paths that holds no file is a removed note, with no note type and no properties.
func (u Runner) Run(ctx context.Context, v domain.Vault, call plugin.CommandCall) (plugin.CommandResult, error) {
	var entries []plugin.CommandEntry
	if manifest.IsSharedCommand(call.Command) {
		entries = u.commands.ListHandlers()
	} else {
		entries = u.commands.ListCommands()
	}
	if !u.commands.GetParts(call.PluginID).HasBackend {
		return plugin.CommandResult{}, fmt.Errorf("%w: %s of %s", plugin.ErrUnknownCommand, call.Command, call.PluginID)
	}
	index := slices.IndexFunc(entries, func(one plugin.CommandEntry) bool {
		return one.PluginID == call.PluginID && one.Command.ID == call.Command
	})
	if index < 0 {
		return plugin.CommandResult{}, fmt.Errorf("%w: %s of %s", plugin.ErrUnknownCommand, call.Command, call.PluginID)
	}
	declared := entries[index].Command
	if err := u.checkCall(ctx, v, declared, call); err != nil {
		return plugin.CommandResult{}, err
	}
	normalized := call
	normalized.Arguments = plugin.NormalizeCommandArguments(declared, call.Arguments)
	return u.runs.RunCommand(ctx, normalized)
}

// IsAvailable reports whether the condition of the command of the entry holds over the path, as a
// run of it checks; a condition the core cannot decide holds.
func (u Runner) IsAvailable(ctx context.Context, v domain.Vault, entry plugin.CommandEntry, path string) bool {
	return u.checkPath(ctx, v, entry.Command, plugin.CommandCall{PluginID: entry.PluginID, Command: entry.Command.ID, Path: path}, false) == nil
}

func (u Runner) checkCall(ctx context.Context, v domain.Vault, declared manifest.Command, call plugin.CommandCall) error {
	if len(call.Paths) == 0 {
		if err := u.checkPath(ctx, v, declared, call, false); err != nil {
			return err
		}
	}
	for _, path := range call.Paths {
		one := call
		one.Path, one.Paths = path, nil
		if err := u.checkPath(ctx, v, declared, one, true); err != nil {
			return err
		}
	}
	return plugin.CheckCommandArguments(declared, call, HasItems(u.conditions, declared))
}

func (u Runner) checkPath(ctx context.Context, v domain.Vault, declared manifest.Command, call plugin.CommandCall, isGoneAllowed bool) error {
	condition, err := u.conditions.Parse(declared.When)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", plugin.ErrBadArguments, declared.ID, err)
	}
	if condition.HasItems() && call.Path == "" {
		return fmt.Errorf("%w: %s needs a file", plugin.ErrBadArguments, declared.ID)
	}
	target := plugin.NewTargetValues(call.Path, false, u.types.GetContentType(call.Path))
	values, err := u.readNote(ctx, v, condition, target, call.Path)
	if isGoneAllowed && errors.Is(err, edit.ErrNotExist) {
		values, err = target.WithNote("", nil), nil
	}
	if err != nil {
		return fmt.Errorf("read the note %s: %w", call.Path, err)
	}
	if slices.Contains(condition.ListNames(), "config") {
		values = u.config.addConfig(values)
	}
	if condition.Evaluate(values.Get()) == when.False {
		return fmt.Errorf("%w: %s is not available over %q, as its condition %q does not hold there",
			plugin.ErrBadArguments, declared.ID, call.Path, declared.When)
	}
	return nil
}

func (u Runner) readNote(
	ctx context.Context, v domain.Vault, condition when.Condition, target plugin.TargetValues, path string,
) (plugin.TargetValues, error) {
	names := condition.ListNames()
	if (!slices.Contains(names, "target.noteType") && !slices.Contains(names, "target.properties")) || !target.IsNote() {
		return target, nil
	}
	return u.notes.addNote(ctx, v, target, path)
}

// targetNotes adds the type and the properties of a note to the values a condition reads.
type targetNotes interface {
	addNote(ctx context.Context, v domain.Vault, target plugin.TargetValues, path string) (plugin.TargetValues, error)
}

type readingNotes struct{ open OpenReader }

func (n readingNotes) addNote(ctx context.Context, v domain.Vault, target plugin.TargetValues, path string) (plugin.TargetValues, error) {
	reader, err := n.open(v)
	if err != nil {
		return target, err
	}
	noteType, properties, err := edit.ReadHead(ctx, reader, path)
	if err != nil {
		return target, err
	}
	return target.WithNote(noteType, properties), nil
}

// noNotes leaves a condition over the type or the properties of a note undecided.
type noNotes struct{}

func (noNotes) addNote(_ context.Context, _ domain.Vault, target plugin.TargetValues, _ string) (plugin.TargetValues, error) {
	return target, nil
}

// targetConfig adds the config variable to the values a condition reads.
type targetConfig interface {
	addConfig(values plugin.TargetValues) plugin.TargetValues
}

type givenConfig struct{ config Config }

func (c givenConfig) addConfig(values plugin.TargetValues) plugin.TargetValues {
	return values.WithConfig(c.config.GetWhenConfig())
}

// noConfig leaves a condition over config undecided.
type noConfig struct{}

func (noConfig) addConfig(values plugin.TargetValues) plugin.TargetValues { return values }
