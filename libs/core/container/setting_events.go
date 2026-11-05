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

// listSettingValues lists the value in force of every setting the shell and the plugins declare.
func (c Config) listSettingValues(target Settings) (settingevents.Values, error) {
	values := settingevents.Values{}
	var failures []error
	put := func(path []string, encoded string, err error) {
		if err != nil {
			failures = append(failures, err)
			return
		}
		values[settingevents.JoinPath(path)] = encoded
	}
	for _, declared := range settings.GetAppDeclaration() {
		path := append([]string{settings.OwnerApp}, strings.Split(declared.Key, ".")...)
		if value, found := target.GetValue(path...); found {
			encoded, err := json.Marshal(value)
			put(path, string(encoded), err)
		}
	}
	for _, plugin := range c.Plugins()() {
		for _, declared := range plugin.Settings {
			path := []string{plugin.ID, declared.Key}
			if value, found := target.GetValue(path...); found {
				encoded, err := json.Marshal(value)
				put(path, string(encoded), err)
			} else if declared.DefaultJSON != "" {
				put(path, declared.DefaultJSON, nil)
			}
		}
	}
	return values, errors.Join(failures...)
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
