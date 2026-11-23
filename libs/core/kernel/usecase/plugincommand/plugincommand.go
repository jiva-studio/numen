// Package plugincommand holds the scenarios over the commands the plugins declare: listing them and
// running one, checked against its declaration.
package plugincommand

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/commandrun"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Runner is the helper that runs a command of a plugin, checked against its declaration, as the container
// builds and hands it on: the container may import nothing under internal/.
type Runner struct{ commandrun.Runner }

func NewRunner(commands commandrun.Commands, runs commandrun.Runs, types commandrun.ContentTypes, readers vaultport.VaultReaders, config pluginport.WhenConfig, conditions *when.Conditions) Runner {
	var open commandrun.OpenReader
	if readers != nil {
		open = func(v domain.Vault) (edit.Reader, error) { return readers.Open(v) }
	}
	return Runner{commandrun.New(commands, runs, types, open, config, conditions)}
}

// RunnerOpener opens the command runner over a vault.
type RunnerOpener = func(ctx context.Context, v domain.Vault) (Runner, func() error, error)

// NewList makes the scenario that lists the commands the plugins declare and, over points, the slots they
// draw; points may be nil, which lists none. conditions parses the condition of a command; it is shared
// with the scenarios that run commands, so one condition is parsed once.
func NewList(commands PluginCommands, points SlotPoints, conditions *when.Conditions) List {
	return List{commands: commands, points: points, conditions: conditions}
}

// List lists the commands the enabled plugins declare.
type List struct {
	commands   PluginCommands
	conditions *when.Conditions
	points     SlotPoints
}

// Execute lists the commands that a plugin backend runs.
func (u List) Execute() []plugin.CommandEntry {
	return slices.DeleteFunc(u.ExecuteAll(), func(one plugin.CommandEntry) bool { return !u.commands.GetParts(one.PluginID).HasBackend })
}

// ExecuteAll lists every command the enabled plugins declare, whichever part of the plugin runs it.
// A plugin with neither a frontend nor a backend has nothing that runs its commands, and a plugin the
// host holds suspended runs none; both list none.
func (u List) ExecuteAll() []plugin.CommandEntry {
	return slices.DeleteFunc(u.commands.ListCommands(), func(one plugin.CommandEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// ExecuteHandlers lists the handlers of shared commands of the enabled plugins, each as a command under
// the full id of its shared command, leaving out those of a plugin the host holds suspended and of a
// plugin with neither a frontend nor a backend.
func (u List) ExecuteHandlers() []plugin.CommandEntry {
	return slices.DeleteFunc(u.commands.ListHandlers(), func(one plugin.CommandEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// ExecuteSlots lists the slot entries of the enabled plugins, leaving out those of a plugin the host
// holds suspended and of a plugin with neither a frontend nor a backend.
func (u List) ExecuteSlots() []plugin.SlotEntry {
	return slices.DeleteFunc(u.commands.ListSlots(), func(one plugin.SlotEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// ExecuteSlotPoints lists the slots the enabled plugins draw, leaving out those of a plugin the host
// holds suspended and of a plugin with neither a frontend nor a backend.
func (u List) ExecuteSlotPoints() []plugin.SlotPointEntry {
	if u.points == nil {
		return nil
	}
	return slices.DeleteFunc(u.points.ListSlotPoints(), func(one plugin.SlotPointEntry) bool {
		parts := u.commands.GetParts(one.PluginID)
		return parts.IsSuspended || (!parts.HasBackend && !parts.HasFrontend)
	})
}

// ExecuteKeybindings lists the keybindings of the enabled plugins, leaving out those of a plugin the host
// holds suspended and of a plugin with neither a frontend nor a backend.
func (u List) ExecuteKeybindings() []plugin.KeybindingEntry {
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
	for _, extension := range u.types.ListExtensions(domain.AnyContentType) {
		out[extension] = string(u.types.GetContentType("file." + extension))
	}
	return out
}
