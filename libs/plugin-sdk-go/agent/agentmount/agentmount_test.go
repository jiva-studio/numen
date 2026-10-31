package agentmount_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentmount"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
)

type handler struct {
	agentrun.Service
	agentrun.ExecutablePath
}

func (*handler) GetAgentState(
	context.Context, *connect.Request[agentrun.GetAgentStateRequest],
) (*connect.Response[agentrun.GetAgentStateResponse], error) {
	return connect.NewResponse(&agentrun.GetAgentStateResponse{Unreachable: ""}), nil
}

func newAgent() agentmount.Agent { return new(handler) }

func TestNewEntryHandlerServesAgentService(t *testing.T) {
	_, newHandler := agentmount.NewEntry(newAgent)

	agenttest.CheckLinkedHandler(t, newHandler)
}
