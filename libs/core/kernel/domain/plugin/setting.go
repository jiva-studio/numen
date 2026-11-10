package plugin

import "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

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
	}
}
