package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// PersistHandler is called when a plugin modifies its own settings.
type PersistHandler func(pluginID string, key string, value any) error

// Setting declares one configuration field managed by the core host.
type Setting struct {
	Key           string
	Type          string
	Title         string
	Description   string
	DefaultJSON   string
	Palette       bool
	Icon          string
	Options       manifest.SettingOptions
	Min           *float64
	Max           *float64
	Step          *float64
	OptionsBy     *manifest.OptionsBy
	When          *manifest.Condition
	Accept        []string
	Multiple      bool
	Preview       bool
	Operation     string
	OptionsFrom   string
	ActiveOptions manifest.SettingOptions
	IsShown       bool
	ValueJSON     string
}

// NewSetting returns the setting as a window draws it. The default, as JSON, and the options
// are the ones the caller resolved for the declaration, and nothing is resolved under stored values.
func NewSetting(declared manifest.Setting, defaultJSON string, options manifest.SettingOptions) Setting {
	return Setting{
		Key:         declared.Key,
		Type:        declared.Type,
		Title:       declared.Title,
		Description: declared.Description,
		DefaultJSON: defaultJSON,
		Palette:     declared.Palette,
		Icon:        declared.Icon,
		Options:     options,
		Min:         declared.Min,
		Max:         declared.Max,
		Step:        declared.Step,
		OptionsBy:   declared.OptionsBy,
		When:        declared.When,
		Accept:      declared.Accept,
		Multiple:    declared.Multiple,
		Preview:     declared.Preview,
		OptionsFrom: declared.OptionsFrom,
		Operation:   declared.Operation,
	}
}

// ListSettings returns the settings the manifest declares as a window draws them, in the order of
// the declaration. The options in force, whether the setting is shown and the stored value are
// resolved under values, the values the plugin has now.
func ListSettings(declared manifest.Manifest, values map[string]any) []Setting {
	settings := make([]Setting, 0, len(declared.Settings))
	for _, one := range declared.Settings {
		defaultJSON := ""
		if one.Default != nil {
			if encoded, err := json.Marshal(one.Default); err == nil {
				defaultJSON = string(encoded)
			}
		}
		setting := NewSetting(one, defaultJSON, one.Options)
		setting.ActiveOptions = declared.GetSettingOptions(one, values)
		if len(setting.ActiveOptions) == 0 {
			setting.ActiveOptions = setting.Options
		}
		setting.IsShown = declared.IsSettingShown(one, values)
		if value, ok := values[one.Key]; ok {
			if encoded, err := json.Marshal(value); err == nil {
				setting.ValueJSON = string(encoded)
			}
		}
		settings = append(settings, setting)
	}
	return settings
}

// FormatSettingValues returns the values as text: a string stays as it is and any other value is
// its JSON.
func FormatSettingValues(values map[string]any) map[string]string {
	formatted := make(map[string]string, len(values))
	for key, value := range values {
		if text, ok := value.(string); ok {
			formatted[key] = text
			continue
		}
		if encoded, err := json.Marshal(value); err == nil {
			formatted[key] = string(encoded)
		} else {
			formatted[key] = fmt.Sprint(value)
		}
	}
	return formatted
}
