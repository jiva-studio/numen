package subscription_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
)

func TestState_CanWriteInTheActiveAndTheGracePhaseOnly(t *testing.T) {
	for name, one := range map[string]struct {
		phase subscription.Phase
		want  bool
	}{
		"none":      {phase: subscription.PhaseNone, want: false},
		"active":    {phase: subscription.PhaseActive, want: true},
		"grace":     {phase: subscription.PhaseGrace, want: true},
		"read only": {phase: subscription.PhaseReadOnly, want: false},
		"deleted":   {phase: subscription.PhaseDeleted, want: false},
	} {
		if got := (subscription.State{Phase: one.phase}).CanWrite(); got != one.want {
			t.Errorf("CanWrite in the phase %s = %v, want %v", name, got, one.want)
		}
	}
}
