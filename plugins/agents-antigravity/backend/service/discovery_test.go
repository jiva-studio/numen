package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
)

func TestDiscovery(t *testing.T) {
	agenttest.CheckDiscovery(t, "agy", FindAntigravity, IsAntigravityInstalled, withoutAgentsAnywhere)
}
