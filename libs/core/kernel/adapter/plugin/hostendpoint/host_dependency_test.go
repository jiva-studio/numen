package hostendpoint

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestDependencyKindsRoundTripTheWire(t *testing.T) {
	for _, kind := range manifest.GetDependencyKinds() {
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
	want := map[plugin.DependencyState]v1.DependencyState{
		plugin.DependencyStateMissing:      v1.DependencyState_DEPENDENCY_STATE_MISSING,
		plugin.DependencyStateProvisioning: v1.DependencyState_DEPENDENCY_STATE_PROVISIONING,
		plugin.DependencyStateReady:        v1.DependencyState_DEPENDENCY_STATE_READY,
		plugin.DependencyStateFailed:       v1.DependencyState_DEPENDENCY_STATE_FAILED,
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
