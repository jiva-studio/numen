package cli_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
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
