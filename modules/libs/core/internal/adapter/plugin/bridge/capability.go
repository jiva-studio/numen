package bridge

// The capabilities the core asks a plugin for, as a manifest declares them.
const (
	CapabilityDocument      = "document"
	CapabilityBook          = "book"
	CapabilityRecognition   = "recognition"
	CapabilityTranscription = "transcription"
	CapabilityEmbedding     = "embedding"
	CapabilityDownloader    = "downloader"
	CapabilityProofreading  = "proofreading"
	// CapabilityTools marks a plugin that serves PluginToolService: the tools it offers an agent.
	CapabilityTools = "tools"
	// CapabilityAgentProvider is the capability of a plugin that runs agent programs.
	CapabilityAgentProvider = "ai.agent.provider"
)
