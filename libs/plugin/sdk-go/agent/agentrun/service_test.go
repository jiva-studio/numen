package agentrun_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

func TestServiceIsAnAgent(t *testing.T) {
	var _ agentrun.Runner = &agentrun.Service{}
	var _ agentrun.ToolStore = &agentrun.Service{}
}
