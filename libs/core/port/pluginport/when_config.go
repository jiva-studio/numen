package pluginport

// WhenConfig gives the settings as the config variable of when conditions.
type WhenConfig interface {
	// GetWhenConfig returns the config: a setting of the application at config.app.<key>, and one of a
	// plugin at config.<plugin id split at its dots>.<key>, each holding its value or its default.
	GetWhenConfig() map[string]any
}
