package bridge

// The capabilities the core requests from a plugin, as declared in its
// manifest.
const (
	CapabilityEmbedding = "embedding"
	// CapabilityTools marks a plugin that serves PluginToolService: the tools it offers an agent.
	CapabilityTools = "tools"
	// CapabilityAgentProvider is the capability of a plugin that runs agent programs.
	CapabilityAgentProvider = "ai.agent.provider"
)
