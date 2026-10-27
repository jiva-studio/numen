package hardware

import (
	"runtime"
)

// PerformanceProfile is the high-level workload preset.
type PerformanceProfile = string

const (
	// ProfileEco minimizes power consumption and CPU heat (1 thread).
	ProfileEco PerformanceProfile = "eco"
	// ProfileBalanced provides responsive balance between speed and UI (50% CPU).
	ProfileBalanced PerformanceProfile = "balanced"
	// ProfileMaximum saturates 100% of detected hardware for maximum throughput.
	ProfileMaximum PerformanceProfile = "maximum"
)

// DefaultPerformanceProfile is the default workload preset for untouched installations.
const DefaultPerformanceProfile = ProfileBalanced

// Performance is the settings section for workload and resource calibration.
type Performance struct {
	// Profile is the active preset.
	Profile string `json:"profile"`
}

// DefaultPerformance returns the default performance settings.
func DefaultPerformance() Performance {
	return Performance{Profile: string(DefaultPerformanceProfile)}
}

// AcceleratorCoreML names the Apple Core ML runtime, present on Apple Silicon.
const AcceleratorCoreML = "coreml"

// Specs represents the detected host machine hardware capabilities. A zero count or size is unknown.
type Specs struct {
	// LogicalCPUs is the number of hardware threads the process may run on.
	LogicalCPUs int `json:"logical_cpus"`
	// PhysicalCPUs is the number of physical cores, including efficiency cores.
	PhysicalCPUs int `json:"physical_cpus"`
	// PerformanceCPUs is the number of physical cores of the highest performance class; it equals PhysicalCPUs when the cores do not differ.
	PerformanceCPUs int `json:"performance_cpus"`
	// TotalRAMMB is the installed physical memory in megabytes.
	TotalRAMMB int64 `json:"total_ram_mb"`
	// AvailableRAMMB is the memory that can be claimed without swapping, in megabytes.
	AvailableRAMMB int64 `json:"available_ram_mb"`
	// Accelerators lists the compute accelerators the operating system and architecture provide.
	Accelerators []string `json:"accelerators"`
}

// ResolvedCompute holds the concrete, calibrated parameters to apply to worker pools.
type ResolvedCompute struct {
	Profile PerformanceProfile `json:"profile"`
	// CoreBudget is the number of hardware threads the profile lets CPU-bound work occupy at once.
	CoreBudget int `json:"core_budget"`
	// Workers is the size of a general pool for CPU-bound jobs such as text extraction and file parsing.
	Workers int `json:"workers"`
	// MemoryLimitMB is the soft limit on the memory managed by the Go runtime, in megabytes. Zero means no limit, which is the case when the installed memory is unknown.
	MemoryLimitMB int64 `json:"memory_limit_mb"`
}

// Detect inspects the current host machine capabilities.
func Detect() Specs {
	hw := readPlatformHardware()
	logical := runtime.NumCPU()
	if hw.LogicalCPUs > 0 && hw.LogicalCPUs < logical {
		logical = hw.LogicalCPUs
	}
	hw.LogicalCPUs = logical
	return hw.normalize()
}

// normalize makes the counts consistent: at least one logical CPU, no more cores than threads, no more performance cores than cores, no more free memory than installed.
func (hw Specs) normalize() Specs {
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

// getPerformanceThreads is the number of hardware threads of the performance cores. Each core is taken to have the ratio of logical to physical CPUs, rounded up; when the counts are not known it is the logical CPU count.
func (hw Specs) getPerformanceThreads() int {
	threadsPerCore := (hw.LogicalCPUs + hw.PhysicalCPUs - 1) / hw.PhysicalCPUs
	threads := hw.PerformanceCPUs * threadsPerCore
	if threads > hw.LogicalCPUs {
		threads = hw.LogicalCPUs
	}
	return threads
}

// getMemoryLimitMB is a share of the installed memory in megabytes, and zero when that size is unknown.
func (hw Specs) getMemoryLimitMB(sharePercent int64) int64 {
	return hw.TotalRAMMB * sharePercent / 100
}

func getBudget(hw Specs, sharePercent int) int {
	return max(1, hw.getPerformanceThreads()*sharePercent/100)
}

// ResolveCompute computes the active workload parameters for a given profile and hardware specs. Every pool it sizes occupies at most CoreBudget hardware threads.
func ResolveCompute(profile PerformanceProfile, hw Specs) ResolvedCompute {
	hw = hw.normalize()

	switch profile {
	case ProfileEco:
		return ResolvedCompute{
			Profile:       ProfileEco,
			CoreBudget:    ecoCoreBudget,
			Workers:       1,
			MemoryLimitMB: hw.getMemoryLimitMB(ecoMemoryLimitPercent),
		}

	case ProfileMaximum:
		budget := getBudget(hw, maximumSharePercent)
		return ResolvedCompute{
			Profile:       ProfileMaximum,
			CoreBudget:    budget,
			Workers:       budget,
			MemoryLimitMB: hw.getMemoryLimitMB(maximumMemoryLimitPercent),
		}

	case ProfileBalanced:
		fallthrough
	default:
		budget := getBudget(hw, balancedSharePercent)
		return ResolvedCompute{
			Profile:       ProfileBalanced,
			CoreBudget:    budget,
			Workers:       max(budget/balancedWorkerShare, 1),
			MemoryLimitMB: hw.getMemoryLimitMB(balancedMemoryLimitPercent),
		}
	}
}
