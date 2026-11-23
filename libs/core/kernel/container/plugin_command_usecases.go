package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// pluginCommandUseCases is everything the application does with the commands the plugins declare:
// listing them, listing the items of one, and running one.
type pluginCommandUseCases struct {
	List       plugincommand.List
	Listing    plugincommand.ListCommands
	Run        plugincommand.Runner
	RunCommand plugincommand.RunCommand
	ListItems  plugincommand.ListItems
}
