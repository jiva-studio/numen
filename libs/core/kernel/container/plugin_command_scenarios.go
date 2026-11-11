package container

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"

// PluginCommandScenarios is everything the application does with the commands the plugins declare:
// listing them, listing the items of one, and running one.
type PluginCommandScenarios struct {
	List       plugincommand.List
	Listing    plugincommand.ListCommands
	Run        plugincommand.Run
	RunCommand plugincommand.RunCommand
	ListItems  plugincommand.ListItems
}
