package agentrun_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

func TestServiceIsAnAgentServiceHandler(t *testing.T) {
	var _ numenv1connect.AgentServiceHandler = &agentrun.Service{}
}
