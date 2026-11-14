package settings

import (
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// ListSettingValues lists the value in force of every declared setting: the stored value, else the
// default its owner declares.
type ListSettingValues struct {
	codec    settingsport.ValueCodec
	file     settingsport.SettingsFile
	defaults settingsport.Defaults
	app      settingsport.AppDeclarations
	plugins  settingsport.PluginInfos
}

// NewListSettingValues makes the use case. plugins is nil where there is no plugin host.
func NewListSettingValues(
	codec settingsport.ValueCodec, file settingsport.SettingsFile, defaults settingsport.Defaults, app settingsport.AppDeclarations,
	plugins settingsport.PluginInfos,
) *ListSettingValues {
	return &ListSettingValues{codec: codec, file: file, defaults: defaults, app: app, plugins: plugins}
}

// Execute lists the values the file holds, with the defaults for the rest.
func (u *ListSettingValues) Execute() (settingsdomain.Values, error) {
	stored, err := u.file.ReadChecked()
	if err != nil {
		return nil, err
	}
	return u.list(stored)
}

// ListDefaults lists the values of an unconfigured installation.
func (u *ListSettingValues) ListDefaults() (settingsdomain.Values, error) {
	return u.list(u.defaults.GetDefaults())
}

func (u *ListSettingValues) list(stored settingsdomain.Stored) (settingsdomain.Values, error) {
	values := settingsdomain.Values{}
	var failures []error
	add := func(path []string, value any) {
		encoded, err := u.codec.Encode(value)
		if err != nil {
			failures = append(failures, err)
			return
		}
		values[settingsdomain.JoinPath(path)] = encoded
	}
	for _, declared := range u.app.GetAppDeclaration() {
		path := append([]string{settingsdomain.AppOwner}, strings.Split(declared.Key, ".")...)
		if value, found := stored.GetValue(path...); found {
			add(path, value)
		}
	}
	if u.plugins != nil {
		for _, one := range u.plugins.GetPluginInfos() {
			listPluginValues(stored, one, add, values)
		}
	}
	connections, err := stored.GetConnectionsJSON()
	if err != nil {
		failures = append(failures, err)
	} else if connections != "" {
		values[settingsdomain.JoinPath([]string{settingsdomain.ConnectionsKey})] = connections
	}
	return values, errors.Join(failures...)
}

func listPluginValues(stored settingsdomain.Stored, one plugin.Info, add func([]string, any), values settingsdomain.Values) {
	for _, declared := range one.Settings {
		path := []string{one.ID, declared.Key}
		if value, found := stored.GetValue(path...); found {
			add(path, value)
		} else if declared.DefaultJSON != "" {
			values[settingsdomain.JoinPath(path)] = declared.DefaultJSON
		}
	}
	enabled := !one.IsOffByDefault
	if value, found := stored.GetValue(one.ID, settingsdomain.HostFieldEnabled); found {
		if configured, isBool := value.(bool); isBool {
			enabled = configured
		}
	}
	add([]string{one.ID, settingsdomain.HostFieldEnabled}, enabled)
	if value, found := stored.GetValue(one.ID, settingsdomain.HostFieldDiscover); found {
		add([]string{one.ID, settingsdomain.HostFieldDiscover}, value)
	}
}
