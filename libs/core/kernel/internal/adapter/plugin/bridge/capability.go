package bridge

// The capabilities the core requests from a plugin, as declared in its
// manifest.
const (
	// CapabilitySearchSource marks a plugin that serves SearchSourceService: a participant of the search.
	CapabilitySearchSource = "search.source"
	// CapabilityTools marks a plugin that serves PluginToolService: the tools it offers an agent.
	CapabilityTools = "tools"
)
