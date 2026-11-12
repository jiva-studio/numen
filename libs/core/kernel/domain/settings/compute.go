package settings

// The workload presets a person chooses between.
const (
	// ProfileEco minimizes power consumption and CPU heat (1 thread).
	ProfileEco = "eco"
	// ProfileBalanced provides responsive balance between speed and UI (50% CPU).
	ProfileBalanced = "balanced"
	// ProfileMaximum saturates 100% of detected hardware for maximum throughput.
	ProfileMaximum = "maximum"
)

// MachineSpecs represents the detected host machine hardware capabilities. A zero count or size is unknown.
type MachineSpecs struct {
	LogicalCPUs     int
	PhysicalCPUs    int
	PerformanceCPUs int
	TotalRAMMB      int64
	AvailableRAMMB  int64
}

// ResolvedCompute holds the concrete, calibrated parameters to apply to worker pools.
type ResolvedCompute struct {
	Profile       string
	CoreBudget    int
	Workers       int
	MemoryLimitMB int64
}

const (
	// balancedSharePercent is the share of the performance threads Balanced leaves to CPU-bound work, so the rest stays with the interface.
	balancedSharePercent = 50
	// maximumSharePercent is the share Maximum gives to CPU-bound work.
	maximumSharePercent = 100

	// ecoCoreBudget is the single hardware thread Eco occupies.
	ecoCoreBudget = 1
	// The share of installed memory each profile lets the Go runtime use before the collector works harder. Eco leaves the most to other programs; Maximum leaves the least.
	ecoMemoryLimitPercent      = 25
	balancedMemoryLimitPercent = 50
	maximumMemoryLimitPercent  = 75

	// balancedWorkerShare divides the Balanced budget into the size of the general pool.
	balancedWorkerShare = 2
)

// Normalize returns the specs with every count made consistent: at least one CPU, no more physical
// than logical CPUs, no more performance than physical CPUs, and no more available than total memory.
func (hw MachineSpecs) Normalize() MachineSpecs {
	if hw.LogicalCPUs < 1 {
		hw.LogicalCPUs = 1
	}
	if hw.PhysicalCPUs < 1 || hw.PhysicalCPUs > hw.LogicalCPUs {
		hw.PhysicalCPUs = hw.LogicalCPUs
	}
	if hw.PerformanceCPUs < 1 || hw.PerformanceCPUs > hw.PhysicalCPUs {
		hw.PerformanceCPUs = hw.PhysicalCPUs
	}
	if hw.TotalRAMMB > 0 && hw.AvailableRAMMB > hw.TotalRAMMB {
		hw.AvailableRAMMB = hw.TotalRAMMB
	}
	return hw
}

func (hw MachineSpecs) getPerformanceThreads() int {
	threadsPerCore := (hw.LogicalCPUs + hw.PhysicalCPUs - 1) / hw.PhysicalCPUs
	return min(hw.PerformanceCPUs*threadsPerCore, hw.LogicalCPUs)
}

func (hw MachineSpecs) getMemoryLimitMB(sharePercent int64) int64 {
	return hw.TotalRAMMB * sharePercent / 100
}

func (hw MachineSpecs) getBudget(sharePercent int) int {
	return max(1, hw.getPerformanceThreads()*sharePercent/100)
}

// ResolveCompute computes the active workload parameters for a given profile and hardware specs. Every pool it sizes occupies at most CoreBudget hardware threads.
// A profile that is none of the three is Balanced.
func ResolveCompute(profile string, hw MachineSpecs) ResolvedCompute {
	hw = hw.Normalize()

	switch profile {
	case ProfileEco:
		return ResolvedCompute{
			Profile:       ProfileEco,
			CoreBudget:    ecoCoreBudget,
			Workers:       1,
			MemoryLimitMB: hw.getMemoryLimitMB(ecoMemoryLimitPercent),
		}

	case ProfileMaximum:
		budget := hw.getBudget(maximumSharePercent)
		return ResolvedCompute{
			Profile:       ProfileMaximum,
			CoreBudget:    budget,
			Workers:       budget,
			MemoryLimitMB: hw.getMemoryLimitMB(maximumMemoryLimitPercent),
		}

	default:
		budget := hw.getBudget(balancedSharePercent)
		return ResolvedCompute{
			Profile:       ProfileBalanced,
			CoreBudget:    budget,
			Workers:       max(budget/balancedWorkerShare, 1),
			MemoryLimitMB: hw.getMemoryLimitMB(balancedMemoryLimitPercent),
		}
	}
}
