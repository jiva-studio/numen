package command_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
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

func (s useCases) withVersions(versions plugincommand.VersionItems) useCases {
	s.RunCommand = plugincommand.NewRunCommand(s.List, s.Run, versions)
	s.ListItems = plugincommand.NewListItems(s.List, versions)
	return s
}

func newHandler(s useCases) *command.Handler {
	return command.NewHandler(shownVault{}, s.Listing, s.RunCommand, s.ListItems)
}

// noTypes is a table that gives no file a type.
type noTypes struct{}

func (noTypes) GetContentType(string) domain.ContentType   { return domain.UnknownContentType }
func (noTypes) ListExtensions(domain.ContentType) []string { return nil }
