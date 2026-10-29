package pluginport

// PluginSetting declares one configuration field managed by the core host.
type PluginSetting struct {
	Key         string
	Type        string
	Title       string
	Description string
	DefaultJSON string
	Palette     bool
	Icon        string
	Options     PluginSettingOptions
	Min         *float64
	Max         *float64
	Step        *float64
	// ModelsOf is the agent program whose models the setting is offered. Empty is a setting offered none.
	ModelsOf string
	// Replaces are the places in the settings file the setting was kept before the plugin took it
	// over, as the keys that lead to each joined by dots.
	Replaces []string
	// OptionsBy, When and Accept are as the manifest declares them.
	OptionsBy *PluginSettingOptionsBy
	When      *PluginSettingCondition
	Accept    []string
}

// PluginInfo describes a discovered plugin and its declarative settings.
type PluginInfo struct {
	ID           string
	Name         string
	Version      string
	Description  string
	Capabilities []string
	Settings     []PluginSetting
	IsActive     bool
	// Permissions are the permissions the manifest declares.
	Permissions []string
	// Secrets are the names of the secrets the manifest declares, and
	// StoredSecrets those of them that have a key stored. A key never appears here.
	Secrets       []PluginSecret
	StoredSecrets []string
	// Consent is whether the user allows the plugin to request model completions.
	Consent CompletionConsent
	// Failure is set while the host suspends the plugin after it failed to start.
	Failure *PluginFailure
	// HasFrontend is whether the plugin declares a bundle the windows load.
	HasFrontend bool
	// Readers are the readers the manifest declares, with the files each opens.
	Readers []PluginReaderInfo
	// SettingsViews are the components the settings page draws among the settings of the plugin.
	SettingsViews []PluginSettingsView
}

// PluginReaderInfo is a reader of a plugin and the files it opens.
type PluginReaderInfo struct {
	// ID is the id the frontend registers the reader under.
	ID string
	// Extensions are the extensions, lowercase and without the dot, of the files the reader opens:
	// those of its types where it is the default reader of every enabled plugin.
	Extensions []string
}

// CompletionConsent is whether the user allows a plugin to request model
// completions.
type CompletionConsent int

const (
	// ConsentNone is a plugin that does not hold the ai:complete permission.
	ConsentNone CompletionConsent = iota
	// ConsentPending is a plugin without a user decision that has not requested a
	// completion since the host started.
	ConsentPending
	// ConsentRequested is a plugin that has requested a completion since the host
	// started and awaits the user's decision.
	ConsentRequested
	// ConsentAllowed is a plugin the user allows to request completions.
	ConsentAllowed
	ConsentDenied
)
