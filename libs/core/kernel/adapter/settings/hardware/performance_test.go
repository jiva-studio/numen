package hardware_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
)

func TestDetectHardware_ReportsConsistentCounts(t *testing.T) {
	hw := hardware.Reader{}.ReadMachine()

	if hw.LogicalCPUs < 1 || hw.PhysicalCPUs < 1 || hw.PerformanceCPUs < 1 {
		t.Fatalf("detected %+v, want every count at least 1", hw)
	}
	if hw.PhysicalCPUs > hw.LogicalCPUs || hw.PerformanceCPUs > hw.PhysicalCPUs {
		t.Errorf("detected %+v, want performance <= physical <= logical", hw)
	}
	if hw.TotalRAMMB > 0 && hw.AvailableRAMMB > hw.TotalRAMMB {
		t.Errorf("detected %+v, want available memory within installed memory", hw)
	}
}
