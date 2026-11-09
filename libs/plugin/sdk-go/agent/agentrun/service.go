package agentrun

// unimplemented answers every call of the agent service as not implemented.
type unimplemented struct {
	Unimplemented
}

// Service answers the calls of the agent service that every agent plugin answers the same way:
// RegisterTool, ExtendSystemPrompt, GetTools and FinishConversation. A plugin embeds it and
// answers the rest. The zero value is ready.
type Service struct {
	unimplemented
	Toolbox
	Registry
}
