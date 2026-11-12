package textport

// TextPlugins is the enabled plugins, which may record text versions.
type TextPlugins interface {
	// IsEnabled reports whether the plugin of the id is enabled.
	IsEnabled(pluginID string) bool
}
