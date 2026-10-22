package plugin

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestDependencyKindsRoundTripTheWire(t *testing.T) {
	for _, kind := range port.GetDependencyKinds() {
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

func TestDependencyStatesAreWrittenToTheWire(t *testing.T) {
	want := map[port.DependencyState]v1.DependencyState{
		port.DependencyStateMissing:      v1.DependencyState_DEPENDENCY_STATE_MISSING,
		port.DependencyStateProvisioning: v1.DependencyState_DEPENDENCY_STATE_PROVISIONING,
		port.DependencyStateReady:        v1.DependencyState_DEPENDENCY_STATE_READY,
		port.DependencyStateFailed:       v1.DependencyState_DEPENDENCY_STATE_FAILED,
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
