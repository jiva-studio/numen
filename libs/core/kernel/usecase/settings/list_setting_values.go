package settings

import (
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
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
	late     settingsport.LatePrograms
}

// NewListSettingValues makes the use case. plugins and late are nil where there are no plugin host
// and no program list.
func NewListSettingValues(
	codec settingsport.ValueCodec, file settingsport.SettingsFile, defaults settingsport.Defaults, app settingsport.AppDeclarations,
	plugins settingsport.PluginInfos, late settingsport.LatePrograms,
) *ListSettingValues {
	return &ListSettingValues{codec: codec, file: file, defaults: defaults, app: app, plugins: plugins, late: late}
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
	if u.late != nil {
		if late := u.late.GetLate(); len(late) > 0 {
			add([]string{settingsdomain.AppOwner, "agent", "use"}, getAgentUse(stored.GetAgentUse(), late))
		}
	}
	return values, errors.Join(failures...)
}

func getAgentUse(configured string, programs []agent.Program) string {
	if configured != "" {
		return configured
	}
	if def, ok := agent.GetDefaultProgram(programs); ok {
		return def.Name
	}
	return programs[0].Name
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
}
