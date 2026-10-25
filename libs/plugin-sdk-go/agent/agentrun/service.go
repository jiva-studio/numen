package agentrun

import "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

// unimplemented answers every call of AgentService as not implemented.
type unimplemented struct {
	numenv1connect.UnimplementedAgentServiceHandler
}

// Service answers the calls of AgentService that every agent plugin answers the same way:
// RegisterTool, ExtendSystemPrompt, GetAgentTools and FinishConversation. A plugin embeds it and
// answers the rest. The zero value is ready.
type Service struct {
	unimplemented
	Toolbox
	Registry
}
