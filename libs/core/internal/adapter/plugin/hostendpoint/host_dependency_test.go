package hostendpoint

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func TestDependencyKindsRoundTripTheWire(t *testing.T) {
	for _, kind := range pluginport.GetDependencyKinds() {
		found := false
		for value := range v1.DependencyKind_name {
			got, err := readWireDependencyKind(v1.DependencyKind(value))
			if err == nil && got == kind {
				found = true
			}
		}
		if !found {
			t.Errorf("no wire kind reads as %s", kind)
		}
	}
	if _, err := readWireDependencyKind(v1.DependencyKind_DEPENDENCY_KIND_UNSPECIFIED); err == nil {
		t.Error("an unspecified kind was read")
	}
}

func TestWriteWireDependencyStateMapsEveryState(t *testing.T) {
	want := map[pluginport.DependencyState]v1.DependencyState{
		pluginport.DependencyStateMissing:      v1.DependencyState_DEPENDENCY_STATE_MISSING,
		pluginport.DependencyStateProvisioning: v1.DependencyState_DEPENDENCY_STATE_PROVISIONING,
		pluginport.DependencyStateReady:        v1.DependencyState_DEPENDENCY_STATE_READY,
		pluginport.DependencyStateFailed:       v1.DependencyState_DEPENDENCY_STATE_FAILED,
	}
	for state, wire := range want {
		if got := writeWireDependencyState(state); got != wire {
			t.Errorf("%s written as %s, want %s", state, got, wire)
		}
	}
	if got := writeWireDependencyState(""); got != v1.DependencyState_DEPENDENCY_STATE_UNSPECIFIED {
		t.Errorf("an empty state written as %s", got)
	}
}
