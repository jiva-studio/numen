package pluginport

// PluginSetting declares one configuration field managed by the core host.
type PluginSetting struct {
	Key         string
	Type        string
	Title       string
	Description string
	DefaultJSON string
	Palette     bool
	Options     []string
	Min         *float64
	Max         *float64
	Step        *float64
	// ModelsOf is the agent program whose models the setting is offered. Empty is a setting offered none.
	ModelsOf string
	// Replaces are the places in the settings file the setting was kept before the plugin took it
	// over, as the keys that lead to each joined by dots.
	Replaces []string
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
	Secrets       []string
	StoredSecrets []string
	// Consent is whether the person lets the plugin ask for model completions.
	Consent CompletionConsent
	// Failure is set while the host suspends the plugin after it failed to start.
	Failure *PluginFailure
	// HasFrontend is whether the plugin declares a bundle the windows load.
	HasFrontend bool
}

// CompletionConsent is whether the person lets a plugin ask for model
// completions.
type CompletionConsent int

const (
	// ConsentNone is a plugin that does not hold the ai:complete permission.
	ConsentNone CompletionConsent = iota
	// ConsentUnanswered is a plugin the person has not answered for, and that
	// has not asked since the host started.
	ConsentUnanswered
	// ConsentAsked is a plugin that has asked since the host started, was
	ConsentAsked
	// ConsentAllowed is a plugin the person lets ask.
	ConsentAllowed
	ConsentRefused
)
