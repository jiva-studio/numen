package cli_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/commandrun"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// useCases are the use cases over the commands the plugins declare, built the way the container builds them.
type useCases struct {
	List       plugincommand.List
	Listing    plugincommand.ListCommands
	Run        plugincommand.Runner
	RunCommand plugincommand.RunCommand
	ListItems  plugincommand.ListItems
}

func newUseCases(
	commands plugincommand.PluginCommands, runs commandrun.Runs, types contenttypeport.ContentTypeTable,
	config pluginport.WhenConfig,
) useCases {
	if types == nil {
		types = noTypes{}
	}
	conditions := when.NewConditions()
	list := plugincommand.NewList(commands, nil, conditions)
	run := plugincommand.NewRunner(commands, runs, types, nil, config, conditions)
	return useCases{
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

// noCommands is a plugin host with no plugin.
type noCommands struct{}

func (noCommands) ListCommands() []plugin.CommandEntry       { return nil }
func (noCommands) ListSlots() []plugin.SlotEntry             { return nil }
func (noCommands) ListKeybindings() []plugin.KeybindingEntry { return nil }
func (noCommands) ListHandlers() []plugin.CommandEntry       { return nil }
func (noCommands) GetParts(string) plugin.Parts              { return plugin.Parts{} }
func (noCommands) RunCommand(context.Context, plugin.CommandCall) (plugin.CommandResult, error) {
	return plugin.CommandResult{}, domain.ErrUnavailable
}
