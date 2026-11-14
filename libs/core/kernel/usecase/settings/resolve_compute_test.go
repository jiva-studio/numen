package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

type machineOf machine.Specs

func (m machineOf) ReadSpecs() machine.Specs { return machine.Specs(m) }

func TestTheProfileIsResolvedOnTheMachineRead(t *testing.T) {
	specs := machine.Specs{LogicalCPUs: 16, PhysicalCPUs: 8, PerformanceCPUs: 8, TotalRAMMB: 32768}
	got := settingsusecase.NewResolveCompute(machineOf(specs)).Execute(machine.ProfileMaximum)

	if got.Profile != machine.ProfileMaximum || got.Workers != 16 || got.MemoryLimitMB != 24576 {
		t.Errorf("resolved %+v", got)
	}
}

func TestAProfileThatIsNoPresetResolvesAsBalanced(t *testing.T) {
	got := settingsusecase.NewResolveCompute(machineOf{LogicalCPUs: 4, PhysicalCPUs: 4, PerformanceCPUs: 4}).Execute("turbo")
	if got.Profile != machine.ProfileBalanced || got.CoreBudget != 2 {
		t.Errorf("resolved %+v", got)
	}
}
