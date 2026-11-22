package protocol_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/protocol"
)

func TestThePluginProtocolIsInItsFirstVersion(t *testing.T) {
	if protocol.PluginProtocolVersion != 1 {
		t.Errorf("PluginProtocolVersion = %d, want 1", protocol.PluginProtocolVersion)
	}
}
