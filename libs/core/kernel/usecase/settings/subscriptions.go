package settings

// Subscriptions holds the two feeds of setting changes that one installation shares: the signals of the
// plugin host changes and the delivery to each plugin host.
type Subscriptions struct {
	Changes PluginChanges
	Hosts   HostSubscriptions
}
