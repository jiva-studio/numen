package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
)

// The handler a host that links the plugin in serves answers the agent service with the token it
// was given.
func TestTheLinkedHandlerAnswersTheAgentService(t *testing.T) {
	agenttest.CheckLinkedHandler(t, NewHandler)
}
