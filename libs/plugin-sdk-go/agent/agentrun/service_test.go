package agentrun_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

func TestServiceIsAnAgent(t *testing.T) {
	var _ agentrun.Agent = &agentrun.Service{}
}
