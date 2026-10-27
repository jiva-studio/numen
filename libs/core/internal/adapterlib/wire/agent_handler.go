package wire

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// ErrNoAgent is an installation with none. Whether one can be reached is asked of GetAgentState.
var ErrNoAgent = errors.New("no agent is set up for this vault")

// AgentHandler answers AgentService for a window: the agent its tasks go to, the folder a task
// naming none is started in, and why no agent can be reached.
type AgentHandler struct {
	numenv1connect.UnimplementedAgentServiceHandler

	// GetAgent is the agent asked at each call. Nil, or one returning nil, is a window with none.
	GetAgent func() agentport.Agent
	// GetRoot is the folder a task naming none is started in. Nil leaves it empty.
	GetRoot func() string
	// GetUnreachable is why no agent can be reached, empty while one can. Nil is always empty.
	GetUnreachable func() string
}

// getAgent is the agent the handler is given, nil where it has none.
func (h *AgentHandler) getAgent() agentport.Agent {
	if h.GetAgent == nil {
		return nil
	}
	return h.GetAgent()
}

// GetAgentState is whether an agent can be reached at all. Which agent answers is the
// installation's, so every window asks it here.
func (h *AgentHandler) GetAgentState(
	_ context.Context, _ *connect.Request[v1.GetAgentStateRequest],
) (*connect.Response[v1.GetAgentStateResponse], error) {
	var why string
	if h.GetUnreachable != nil {
		why = h.GetUnreachable()
	}
	return connect.NewResponse(&v1.GetAgentStateResponse{Unreachable: why}), nil
}

// AskAgent runs the task on the agent.
func (h *AgentHandler) AskAgent(
	ctx context.Context,
	r *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	taking := h.getAgent()
	if taking == nil {
		return connect.NewError(connect.CodeFailedPrecondition, ErrNoAgent)
	}

	root := r.Msg.GetRoot()
	if root == "" && h.GetRoot != nil {
		root = h.GetRoot()
	}

	task := agentport.Task{
		Question:     r.Msg.GetPrompt(),
		Focus:        r.Msg.GetFocus(),
		Conversation: r.Msg.GetConversation(),
		Agent:        r.Msg.GetAgent(),
		Model:        r.Msg.GetModel(),
		Reasoning:    r.Msg.GetReasoning(),
		Context:      r.Msg.GetContext(),
		Root:         root,
	}
	return StreamAgentTask(ctx, taking, task, stream)
}

// FinishConversation ends the conversation on the agent.
func (h *AgentHandler) FinishConversation(
	ctx context.Context,
	r *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	if taking := h.getAgent(); taking != nil && r.Msg.GetConversation() != "" {
		_ = taking.EndConversation(ctx, r.Msg.GetConversation())
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}
