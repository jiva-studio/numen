package sourcebridge

// ConfigWatcher tells when a plugin has been given its configuration.
type ConfigWatcher interface {
	WatchPluginConfig(pluginID string, fn func()) (stop func())
}
