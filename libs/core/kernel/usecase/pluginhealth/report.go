package pluginhealth

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// Report puts a health item a plugin reports about itself in the list.
type Report struct {
	items   *Items
	plugins pluginport.PluginManifests
}

// NewReport returns the use case over the list and the plugins.
func NewReport(items *Items, plugins pluginport.PluginManifests) *Report {
	return &Report{items: items, plugins: plugins}
}

// Execute puts the item in the list as the plugin's, or replaces the one under its key; an item at
// HealthOK removes the key. It returns plugin.ErrInvalidHealth for a key of the core, an item past
// the limits, a ninth key, an action naming a command other than an app: command the app declares
// with its arguments or one of the plugin's own, and values the command does not accept. The value
// plugin of app:settings.open is set to the plugin.
func (u *Report) Execute(pluginID string, item plugin.HealthItem) error {
	item.PluginID = pluginID
	if plugin.IsCoreHealthKey(item.Key) {
		return fmt.Errorf("%w: the key %q is the core's", plugin.ErrInvalidHealth, item.Key)
	}
	if item.Severity == plugin.HealthOK {
		if err := plugin.CheckHealthKey(item.Key); err != nil {
			return err
		}
		u.items.Remove(pluginID, item.Key)
		return nil
	}
	if err := plugin.CheckHealthItem(item); err != nil {
		return err
	}
	action, err := u.checkAction(pluginID, item.Action)
	if err != nil {
		return err
	}
	item.Action = action
	if !u.items.SetWithin(item, plugin.MaxHealthKeys) {
		return fmt.Errorf("%w: the plugin has %d items already", plugin.ErrInvalidHealth, plugin.MaxHealthKeys)
	}
	return nil
}

func (u *Report) checkAction(pluginID string, action *plugin.HealthAction) (*plugin.HealthAction, error) {
	if action == nil {
		return nil, nil
	}
	checked := *action
	checked.Values = maps.Clone(action.Values)
	issues, err := u.listActionIssues(pluginID, &checked)
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		return nil, fmt.Errorf("%w: %w", plugin.ErrInvalidHealth, errors.Join(issues...))
	}
	return &checked, nil
}

func (u *Report) listActionIssues(pluginID string, checked *plugin.HealthAction) ([]error, error) {
	if !strings.HasPrefix(checked.Command, "app:") {
		command, isOwn := u.findOwnCommand(pluginID, checked.Command)
		if !isOwn {
			return nil, fmt.Errorf("%w: the action names %q, which is neither an app: command nor a command of the plugin", plugin.ErrInvalidHealth, checked.Command)
		}
		return manifest.ListDeclaredValueIssues(command, checked.Command, checked.Values), nil
	}
	if _, isDeclared := manifest.GetAppCommandArguments(checked.Command); !isDeclared {
		return nil, fmt.Errorf("%w: the action names %q, which is no app command the app declares", plugin.ErrInvalidHealth, checked.Command)
	}
	if checked.Command == plugin.AppCommandOpenSettings {
		if checked.Values == nil {
			checked.Values = map[string]string{}
		}
		checked.Values[plugin.OpenSettingsPlugin] = pluginID
	}
	return manifest.ListAppValueIssues(checked.Command, checked.Values), nil
}

func (u *Report) findOwnCommand(pluginID, fullID string) (manifest.Command, bool) {
	owner, name, isFull := strings.Cut(fullID, ":")
	if !isFull || owner != pluginID {
		return manifest.Command{}, false
	}
	found, isEnabled := u.plugins.GetPlugin(pluginID)
	if !isEnabled {
		return manifest.Command{}, false
	}
	for _, command := range found.Manifest.Contributes.Commands {
		if command.ID == name {
			return command, true
		}
	}
	return manifest.Command{}, false
}
