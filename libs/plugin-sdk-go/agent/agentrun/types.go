package agentrun

import (
	"net/http"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

type AgentModel = numenv1.AgentModel

type AgentProgram = numenv1.AgentProgram

type AgentProgramCommand = numenv1.AgentProgramCommand

type AgentToolDefinition = numenv1.AgentToolDefinition

type AskAgentRequest = numenv1.AskAgentRequest

type AskAgentResponse = numenv1.AskAgentResponse

type AskAgentResponse_EditProgress = numenv1.AskAgentResponse_EditProgress

type AskAgentResponse_ModelStarted = numenv1.AskAgentResponse_ModelStarted

type AskAgentResponse_Stopped = numenv1.AskAgentResponse_Stopped

type AskAgentResponse_Text = numenv1.AskAgentResponse_Text

type AskAgentResponse_ToolCall = numenv1.AskAgentResponse_ToolCall

type AskAgentResponse_ToolFinished = numenv1.AskAgentResponse_ToolFinished

type EditProgress = numenv1.EditProgress

type ExtendSystemPromptRequest = numenv1.ExtendSystemPromptRequest

type ExtendSystemPromptResponse = numenv1.ExtendSystemPromptResponse

type FinishConversationRequest = numenv1.FinishConversationRequest

type FinishConversationResponse = numenv1.FinishConversationResponse

type GetAgentProgramsRequest = numenv1.GetAgentProgramsRequest

type GetAgentProgramsResponse = numenv1.GetAgentProgramsResponse

type GetAgentStateRequest = numenv1.GetAgentStateRequest

type GetAgentStateResponse = numenv1.GetAgentStateResponse

type GetAgentToolsRequest = numenv1.GetAgentToolsRequest

type GetAgentToolsResponse = numenv1.GetAgentToolsResponse

type ModelStarted = numenv1.ModelStarted

type Range = numenv1.Range

type ReasoningLevel = numenv1.ReasoningLevel

type RegisterToolRequest = numenv1.RegisterToolRequest

type RegisterToolResponse = numenv1.RegisterToolResponse

type ToolCall = numenv1.ToolCall

type ToolDeclaration = numenv1.ToolDeclaration

type ToolFinished = numenv1.ToolFinished

type AgentServiceHandler = numenv1connect.AgentServiceHandler

type UnimplementedAgentServiceHandler = numenv1connect.UnimplementedAgentServiceHandler

type AgentServiceClient = numenv1connect.AgentServiceClient

// NewAgentServiceHandler builds an HTTP handler from the service implementation.
func NewAgentServiceHandler(svc AgentServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewAgentServiceHandler(svc, opts...)
}

// NewAgentServiceClient constructs a client for the numen.v1.AgentService service.
func NewAgentServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) AgentServiceClient {
	return numenv1connect.NewAgentServiceClient(httpClient, baseURL, opts...)
}
