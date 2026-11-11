package plugincommand

import "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

// ListedCommand is a command of an enabled plugin and whether it offers one item per text version of a file.
type ListedCommand struct {
	plugin.CommandEntry
	HasItems bool
}

// Listing is everything a window shows of the commands the enabled plugins declare.
type Listing struct {
	Commands               []ListedCommand
	Slots                  []plugin.SlotEntry
	Keybindings            []plugin.KeybindingEntry
	Handlers               []ListedCommand
	ContentTypeByExtension map[string]string
}

// ListCommands lists the commands, slots, keybindings and handlers of shared commands of the enabled
// plugins, with the content type of each file extension.
type ListCommands struct {
	list  List
	types ContentTypes
}

// Execute lists every command whichever part of the plugin runs it.
func (u ListCommands) Execute() Listing {
	return Listing{
		Commands:               u.mark(u.list.ExecuteAll()),
		Slots:                  u.list.ExecuteSlots(),
		Keybindings:            u.list.ExecuteKeybindings(),
		Handlers:               u.mark(u.list.ExecuteHandlers()),
		ContentTypeByExtension: u.types.Execute(),
	}
}

func (u ListCommands) mark(entries []plugin.CommandEntry) []ListedCommand {
	out := make([]ListedCommand, 0, len(entries))
	for _, one := range entries {
		out = append(out, ListedCommand{CommandEntry: one, HasItems: u.list.HasItems(one.Command)})
	}
	return out
}
