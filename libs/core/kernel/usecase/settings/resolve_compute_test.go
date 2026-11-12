package settings_test

import (
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

type machineOf settingsdomain.MachineSpecs

func (m machineOf) ReadMachine() settingsdomain.MachineSpecs { return settingsdomain.MachineSpecs(m) }

func TestTheProfileIsResolvedOnTheMachineRead(t *testing.T) {
	specs := settingsdomain.MachineSpecs{LogicalCPUs: 16, PhysicalCPUs: 8, PerformanceCPUs: 8, TotalRAMMB: 32768}
	got := settingsusecase.NewResolveCompute(machineOf(specs)).Execute(settingsdomain.ProfileMaximum)

	if got.Profile != settingsdomain.ProfileMaximum || got.Workers != 16 || got.MemoryLimitMB != 24576 {
		t.Errorf("resolved %+v", got)
	}
}

func TestAProfileThatIsNoPresetResolvesAsBalanced(t *testing.T) {
	got := settingsusecase.NewResolveCompute(machineOf{LogicalCPUs: 4, PhysicalCPUs: 4, PerformanceCPUs: 4}).Execute("turbo")
	if got.Profile != settingsdomain.ProfileBalanced || got.CoreBudget != 2 {
		t.Errorf("resolved %+v", got)
	}
}
