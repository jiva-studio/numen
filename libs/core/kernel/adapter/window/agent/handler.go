// Package agent answers a window's AgentService: the tasks the user gives the agent and the end of
// their conversations.
package agent

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	agentuse "github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
)

// ErrNoAgent means the installation has no agent. Reachability of an agent is reported by GetAgentState.
var ErrNoAgent = errors.New("no agent is set up for this vault")

// Handler answers AgentService for a window: the agent its tasks go to, the folder a task naming
// none is started in, and why no agent can be reached. A handler with no use case set has no agent.
type Handler struct {
	numenv1connect.UnimplementedAgentServiceHandler

	Start *agentuse.StartTask
	End   *agentuse.EndConversation

	GetRoot        func() string
	GetVault       func() domain.VaultID
	GetUnreachable func() string
}

// GetAgentState reports whether an agent can be reached at all. The installation
// chooses which agent responds, so every window queries it here.
func (h *Handler) GetAgentState(
	_ context.Context, _ *connect.Request[v1.GetAgentStateRequest],
) (*connect.Response[v1.GetAgentStateResponse], error) {
	var why string
	if h.GetUnreachable != nil {
		why = h.GetUnreachable()
	}
	return connect.NewResponse(&v1.GetAgentStateResponse{Unreachable: why}), nil
}

// AskAgent runs the task on the agent and writes its steps to the stream while the client is connected.
// The work stops on return, so none outlives the panel that requested it. A model that thinks for a
// minute writes nothing and a silent stream never detects a disconnect, so an empty step is sent each
// second; a failed write ends the work.
func (h *Handler) AskAgent(
	ctx context.Context,
	r *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	if h.Start == nil {
		return connect.NewError(connect.CodeFailedPrecondition, ErrNoAgent)
	}

	root := r.Msg.GetRoot()
	if root == "" && h.GetRoot != nil {
		root = h.GetRoot()
	}
	vault := domain.VaultID(r.Msg.GetVaultId())
	if vault == "" && h.GetVault != nil {
		vault = h.GetVault()
	}

	task := agentdomain.Task{
		Question:     r.Msg.GetPrompt(),
		Focus:        r.Msg.GetFocus(),
		Conversation: r.Msg.GetConversation(),
		Agent:        r.Msg.GetAgent(),
		Model:        r.Msg.GetModel(),
		Reasoning:    r.Msg.GetReasoning(),
		Context:      r.Msg.GetContext(),
		Root:         root,
		Vault:        vault,
	}
	work, err := h.Start.Execute(ctx, task)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	return streamRun(ctx, work, stream)
}

func (h *Handler) FinishConversation(
	ctx context.Context,
	r *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	if h.End != nil {
		if err := h.End.Execute(ctx, r.Msg.GetConversation()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}
