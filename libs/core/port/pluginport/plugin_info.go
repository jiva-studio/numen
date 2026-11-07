package pluginport

// PluginSetting declares one configuration field managed by the core host.
type PluginSetting struct {
	Key           string
	Type          string
	Title         string
	Description   string
	DefaultJSON   string
	Palette       bool
	Icon          string
	Options       PluginSettingOptions
	Min           *float64
	Max           *float64
	Step          *float64
	OptionsBy     *PluginSettingOptionsBy
	When          *PluginSettingCondition
	Accept        []string
	Multiple      bool
	Preview       bool
	OptionsFrom   string
	ActiveOptions PluginSettingOptions
	IsShown       bool
	ValueJSON     string
}

// NewPluginSetting returns the setting as a window draws it. The default, as JSON, and the options
// are the ones the caller resolved for the declaration, and nothing is resolved under stored values.
func NewPluginSetting(declared PluginManifestSetting, defaultJSON string, options PluginSettingOptions) PluginSetting {
	return PluginSetting{
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
	// Secrets are the secrets the manifest declares. A value never appears here.
	Secrets []PluginSecret
	// Failure is set while the host suspends the plugin after it failed to start.
	Failure *PluginFailure
	// HasFrontend is whether the plugin declares a bundle the windows load.
	HasFrontend bool
	// HasBackend is whether the plugin declares a process the host starts.
	HasBackend bool
	// Readers are the readers the manifest declares, with the files each opens.
	Readers []PluginReaderInfo
	// SettingsViews are the components the settings page draws among the settings of the plugin.
	SettingsViews []PluginSettingsView
}

// PluginReaderInfo is a reader of a plugin and the files it opens.
type PluginReaderInfo struct {
	ID         string
	Extensions []string
}
