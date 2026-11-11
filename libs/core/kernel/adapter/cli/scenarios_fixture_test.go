package cli_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

// scenarios are the scenarios over the commands the plugins declare, built the way the container builds them.
type scenarios struct {
	List       plugincommand.List
	Listing    plugincommand.ListCommands
	Run        plugincommand.Run
	RunCommand plugincommand.RunCommand
	ListItems  plugincommand.ListItems
}

func newScenarios(
	commands pluginport.PluginCommands, runs pluginport.PluginCommandRuns, types contenttypeport.ContentTypeTable,
	notes plugincommand.NoteReader, config pluginport.WhenConfig,
) scenarios {
	if types == nil {
		types = noTypes{}
	}
	conditions := when.NewConditions()
	list := plugincommand.NewList(commands, conditions)
	run := plugincommand.NewRun(commands, runs, types, notes, config, conditions)
	return scenarios{
		List:       list,
		Listing:    plugincommand.NewListCommands(list, types),
		Run:        run,
		RunCommand: plugincommand.NewRunCommand(list, run, nil),
		ListItems:  plugincommand.NewListItems(list, nil),
	}
}

// noTypes is a table that gives no file a type.
type noTypes struct{}

func (noTypes) GetContentType(string) domain.ContentType   { return domain.UnknownContentType }
func (noTypes) ListExtensions(domain.ContentType) []string { return nil }

// noCommands is a plugin host that no plugin is enabled in.
type noCommands struct{}

func (noCommands) ListCommands() []plugin.CommandEntry       { return nil }
func (noCommands) ListSlots() []plugin.SlotEntry             { return nil }
func (noCommands) ListKeybindings() []plugin.KeybindingEntry { return nil }
func (noCommands) ListHandlers() []plugin.CommandEntry       { return nil }
func (noCommands) GetParts(string) plugin.Parts              { return plugin.Parts{} }

func newNoPluginCommands() cli.PluginCommands {
	return cli.PluginCommands{List: newScenarios(noCommands{}, nil, nil, nil, nil).List}
}
