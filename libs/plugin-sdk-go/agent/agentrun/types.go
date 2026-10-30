package agentrun

import (
	"net/http"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

type AgentModel = v1.AgentModel

type AgentProgram = v1.AgentProgram

type AgentProgramCommand = v1.AgentProgramCommand

type AgentToolDefinition = v1.AgentToolDefinition

type AskAgentRequest = v1.AskAgentRequest

type AskAgentResponse = v1.AskAgentResponse

type AskAgentResponseEditProgress = v1.AskAgentResponse_EditProgress

type AskAgentResponseModelStarted = v1.AskAgentResponse_ModelStarted

type AskAgentResponseStopped = v1.AskAgentResponse_Stopped

type AskAgentResponseText = v1.AskAgentResponse_Text

type AskAgentResponseToolCall = v1.AskAgentResponse_ToolCall

type AskAgentResponseToolFinished = v1.AskAgentResponse_ToolFinished

type EditProgress = v1.EditProgress

type ExtendSystemPromptRequest = v1.ExtendSystemPromptRequest

type ExtendSystemPromptResponse = v1.ExtendSystemPromptResponse

type FinishConversationRequest = v1.FinishConversationRequest

type FinishConversationResponse = v1.FinishConversationResponse

type GetAgentProgramsRequest = v1.GetAgentProgramsRequest

type GetAgentProgramsResponse = v1.GetAgentProgramsResponse

type GetAgentStateRequest = v1.GetAgentStateRequest

type GetAgentStateResponse = v1.GetAgentStateResponse

type GetAgentToolsRequest = v1.GetAgentToolsRequest

type GetAgentToolsResponse = v1.GetAgentToolsResponse

type ModelStarted = v1.ModelStarted

type Range = v1.Range

type ReasoningLevel = v1.ReasoningLevel

type RegisterToolRequest = v1.RegisterToolRequest

type RegisterToolResponse = v1.RegisterToolResponse

type ToolCall = v1.ToolCall

type ToolDeclaration = v1.ToolDeclaration

type ToolFinished = v1.ToolFinished

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
