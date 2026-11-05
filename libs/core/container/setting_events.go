package container

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// SubscribeSettingChanges starts listening to the settings that change, whether a window, a plugin
// or the palette writes them or a person edits the file by hand. See settingsport.ChangeSubscriber.
func (c Config) SubscribeSettingChanges() (<-chan []settingsport.Setting, func()) {
	events := c.getResources().settingEvents
	if events == nil {
		closed := make(chan []settingsport.Setting)
		close(closed)
		return closed, func() {}
	}
	path, err := c.settingsFile()
	if err != nil {
		c.handleError(err)
		closed := make(chan []settingsport.Setting)
		close(closed)
		return closed, func() {}
	}
	return events.Subscribe(settingevents.Source{
		Read:    c.readSettingValues,
		Initial: c.listDefaultSettingValues,
		Report:  c.handleError,
		Hub:     c.getResources().watches,
		Path:    path,
	})
}

// refreshSettingEvents delivers what a write changed.
func (c Config) refreshSettingEvents() {
	if events := c.getResources().settingEvents; events != nil {
		events.Refresh(c.readSettingValues)
	}
}

// readSettingValues reads the values in force from the settings file. A file that is not there is
// an unconfigured installation, and reading does not write it.
func (c Config) readSettingValues() (settingevents.Values, error) {
	path, err := c.settingsFile()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return c.listSettingValues(DefaultSettings())
	}
	target, err := c.getCheckedSettingsAt(path)
	if err != nil {
		return nil, err
	}
	return c.listSettingValues(target)
}

// listDefaultSettingValues is the values of an installation with no settings file.
func (c Config) listDefaultSettingValues() settingevents.Values {
	values, err := c.listSettingValues(DefaultSettings())
	if err != nil {
		c.handleError(err)
	}
	return values
}

// hostFieldEnabled and hostFieldStoredSecrets are the fields of the object of a plugin that the
// host keeps. A plugin that has no stored value for the first is enabled.
const (
	hostFieldEnabled       = "enabled"
	hostFieldStoredSecrets = pluginport.SettingStoredSecrets
)

// valueList collects the values in force.
type valueList struct {
	values   settingevents.Values
	failures []error
}

func (l *valueList) add(path []string, value any) {
	encoded, err := json.Marshal(value)
	if err != nil {
		l.failures = append(l.failures, err)
		return
	}
	l.values[settingevents.JoinPath(path)] = string(encoded)
}

// listSettingValues lists the value in force of every setting the shell and the plugins declare,
// and the fields the host keeps for each plugin.
func (c Config) listSettingValues(target Settings) (settingevents.Values, error) {
	list := &valueList{values: settingevents.Values{}}
	for _, declared := range settings.GetAppDeclaration() {
		path := append([]string{settings.OwnerApp}, strings.Split(declared.Key, ".")...)
		if value, found := target.GetValue(path...); found {
			list.add(path, value)
		}
	}
	for _, plugin := range c.Plugins()() {
		listPluginValues(target, plugin, list)
	}
	return list.values, errors.Join(list.failures...)
}

func listPluginValues(target Settings, plugin pluginport.PluginInfo, list *valueList) {
	for _, declared := range plugin.Settings {
		path := []string{plugin.ID, declared.Key}
		if value, found := target.GetValue(path...); found {
			list.add(path, value)
		} else if declared.DefaultJSON != "" {
			list.values[settingevents.JoinPath(path)] = declared.DefaultJSON
		}
	}
	enabled := true
	if value, found := target.GetValue(plugin.ID, hostFieldEnabled); found {
		if stored, isBool := value.(bool); isBool {
			enabled = stored
		}
	}
	list.add([]string{plugin.ID, hostFieldEnabled}, enabled)
	if value, found := target.GetValue(plugin.ID, hostFieldStoredSecrets); found {
		list.add([]string{plugin.ID, hostFieldStoredSecrets}, value)
	}
}

// subscribeHost makes the plugin host a reader of the events, once: it takes the settings of its
// plugins and the performance profile through them.
func (c Config) subscribeHost() {
	resources := c.getResources()
	if resources.settingEvents == nil || c.PluginHost == nil {
		return
	}
	resources.hostSubscribed.Do(func() {
		changes, _ := c.SubscribeSettingChanges()
		go func() {
			for batch := range changes {
				c.tellHost(batch)
			}
		}()
	})
}

// tellHost gives the plugin host the settings of the plugins among those that changed, and the
// performance profile.
func (c Config) tellHost(batch []settingsport.Setting) {
	notifyPluginSettings(c.PluginHost, batch, c.handleError)
	for _, one := range batch {
		if !slices.Equal(one.Path, profilePath()) {
			continue
		}
		var profile string
		if err := json.Unmarshal([]byte(one.JSON), &profile); err != nil {
			c.handleError(err)
			continue
		}
		c.PluginHost.SetPerformanceProfile(context.Background(), hardware.ResolveCompute(profile, c.Hardware).Profile)
	}
}

// profilePath is the path of the performance profile in the settings file.
func profilePath() []string { return []string{settings.OwnerApp, "performance", "profile"} }
