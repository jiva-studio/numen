package command_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
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

func (s scenarios) withVersions(versions plugincommand.VersionItems) scenarios {
	s.RunCommand = plugincommand.NewRunCommand(s.List, s.Run, versions)
	s.ListItems = plugincommand.NewListItems(s.List, versions)
	return s
}

func newHandler(s scenarios) *command.Handler {
	return command.NewHandler(shownVault{}, s.Listing, s.RunCommand, s.ListItems)
}

// noTypes is a table that gives no file a type.
type noTypes struct{}

func (noTypes) GetContentType(string) domain.ContentType   { return domain.UnknownContentType }
func (noTypes) ListExtensions(domain.ContentType) []string { return nil }
