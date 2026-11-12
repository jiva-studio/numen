package agentrun

import (
	"context"
	"errors"

	"connectrpc.com/connect"
)

// Unimplemented answers every call of the agent service as not implemented. An agent embeds it and
// implements the calls it answers.
type Unimplemented struct{}

func newUnimplementedError(method string) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("numen.v1.AgentService."+method+" is not implemented"))
}

// Ask is not implemented.
func (Unimplemented) Ask(context.Context, *Request, func(*Response) error) error {
	return newUnimplementedError("AskAgent")
}

// GetState is not implemented.
func (Unimplemented) GetState(context.Context) (*State, error) {
	return nil, newUnimplementedError("GetAgentState")
}

// GetPrograms is not implemented.
func (Unimplemented) GetPrograms(context.Context, []ProgramCommand) ([]*Program, error) {
	return nil, newUnimplementedError("GetAgentPrograms")
}

// RegisterTool is not implemented.
func (Unimplemented) RegisterTool(context.Context, *ToolDefinition) error {
	return newUnimplementedError("RegisterTool")
}

// ExtendSystemPrompt is not implemented.
func (Unimplemented) ExtendSystemPrompt(context.Context, string) (string, error) {
	return "", newUnimplementedError("ExtendSystemPrompt")
}

// GetTools is not implemented.
func (Unimplemented) GetTools(context.Context) (*ToolSet, error) {
	return nil, newUnimplementedError("GetAgentTools")
}

// FinishConversation is not implemented.
func (Unimplemented) FinishConversation(context.Context, string) error {
	return newUnimplementedError("FinishConversation")
}
