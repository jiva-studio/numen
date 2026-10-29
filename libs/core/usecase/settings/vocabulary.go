package settings

import (
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// Setting is one setting of the file, and what to put there. Model is a model a
// setting can name, and Presence is what its files are on this machine.
type (
	Setting                = settingsport.Setting
	Model                  = settingsport.Model
	Presence               = settingsport.Presence
	PluginSetting          = pluginport.PluginSetting
	PluginInfo             = pluginport.PluginInfo
	PluginFailure          = pluginport.PluginFailure
	PluginReaderInfo       = pluginport.PluginReaderInfo
	PluginSecret           = pluginport.PluginSecret
	PluginSettingsView     = pluginport.PluginSettingsView
	PluginSettingOption    = pluginport.PluginSettingOption
	PluginSettingOptions   = pluginport.PluginSettingOptions
	PluginSettingOptionsBy = pluginport.PluginSettingOptionsBy
	PluginSettingCondition = pluginport.PluginSettingCondition
	UnavailableError       = pluginport.UnavailableError
	AgentModel             = agentport.AgentModel
	ReasoningLevel         = agentport.ReasoningLevel
	Consent                = pluginport.CompletionConsent
)

// Whether the user allows a plugin to request model completions.
const (
	ConsentNone      = pluginport.ConsentNone
	ConsentPending   = pluginport.ConsentPending
	ConsentRequested = pluginport.ConsentRequested
	ConsentAllowed   = pluginport.ConsentAllowed
	ConsentDenied    = pluginport.ConsentDenied
)

// What a model's files are on this machine.
const (
	NothingToFetch = settingsport.NothingToFetch
	Present        = settingsport.Present
	NotFetched     = settingsport.NotFetched
)

// A plugin the host cannot reach is not installed, disabled or failed.
const PluginNotInstalled = pluginport.PluginNotInstalled

// ErrNotASetting is a value the settings cannot be read out of again. Nothing is
// written, and the value is the caller's to correct.
var ErrNotASetting = settingsport.ErrNotASetting
