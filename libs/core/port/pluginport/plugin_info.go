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
	// Replaces are the places in the settings file the setting was kept before the plugin took it
	// over, as the keys that lead to each joined by dots.
	Replaces []string
	// OptionsBy, When, Accept and Multiple are as the manifest declares them.
	OptionsBy *PluginSettingOptionsBy
	When      *PluginSettingCondition
	Accept    []string
	Multiple  bool
	// Preview and OptionsFrom are as the manifest declares them.
	Preview     bool
	OptionsFrom string
	// ActiveOptions, IsShown and ValueJSON are the options, the visibility and the value in force
	// under the stored values, resolved by the manifest. A window draws them and applies no rule of
	// OptionsBy or When itself.
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
		Replaces:    declared.Replaces,
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
	// Secrets are the names of the secrets the manifest declares, and
	// StoredSecrets those of them that have a key stored. A key never appears here.
	Secrets       []PluginSecret
	StoredSecrets []string
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
	// ID is the id the frontend registers the reader under.
	ID string
	// Extensions are the extensions, lowercase and without the dot, of the files the reader opens:
	// those of its types where it is the default reader of every enabled plugin.
	Extensions []string
}
