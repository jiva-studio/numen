package machine

// The workload presets a person chooses between.
const (
	// ProfileEco runs background work on one thread at the lowest priority.
	ProfileEco = "eco"
	// ProfileLow gives background work a quarter of the performance threads.
	ProfileLow = "low"
	// ProfileBalanced gives background work half of the performance threads.
	ProfileBalanced = "balanced"
	// ProfileHigh gives background work three quarters of the performance threads.
	ProfileHigh = "high"
	// ProfileMaximum gives background work every performance thread.
	ProfileMaximum = "maximum"
)

// Specs represents the detected host machine hardware capabilities. A zero count or size is unknown.
type Specs struct {
	LogicalCPUs     int
	PhysicalCPUs    int
	PerformanceCPUs int
	TotalRAMMB      int64
	AvailableRAMMB  int64
}

// ResolvedCompute holds the concrete, calibrated parameters to apply to worker pools and plugins.
type ResolvedCompute struct {
	Profile       string
	CoreBudget    int
	Workers       int
	MemoryLimitMB int64
	Nice          int
}

// level is what one profile lets background work occupy: one share of the performance threads and
// of installed memory, the niceness of the plugins and the divisor of the budget that sizes the
// general pool.
type level struct {
	profile        string
	sharePercent   int
	isSingleThread bool
	nice           int
	workerDivisor  int
}

const (
	percentBase = 100

	quarterSharePercent      = 25
	halfSharePercent         = 50
	threeQuarterSharePercent = 75
	fullSharePercent         = 100

	ecoNice      = 19
	lowNice      = 15
	balancedNice = 10
	highNice     = 5
	maximumNice  = 0

	// halfBudgetWorkers gives the general pool half the budget, the rest stays with the interface.
	halfBudgetWorkers = 2
	// fullBudgetWorkers gives the general pool the whole budget.
	fullBudgetWorkers = 1
)

func getLevel(profile string) level {
	switch profile {
	case ProfileEco:
		return level{
			profile: ProfileEco, sharePercent: quarterSharePercent, isSingleThread: true,
			nice: ecoNice, workerDivisor: fullBudgetWorkers,
		}
	case ProfileLow:
		return level{profile: ProfileLow, sharePercent: quarterSharePercent, nice: lowNice, workerDivisor: halfBudgetWorkers}
	case ProfileHigh:
		return level{profile: ProfileHigh, sharePercent: threeQuarterSharePercent, nice: highNice, workerDivisor: fullBudgetWorkers}
	case ProfileMaximum:
		return level{profile: ProfileMaximum, sharePercent: fullSharePercent, nice: maximumNice, workerDivisor: fullBudgetWorkers}
	default:
		return level{profile: ProfileBalanced, sharePercent: halfSharePercent, nice: balancedNice, workerDivisor: halfBudgetWorkers}
	}
}

// Normalize returns the specs with every count made consistent: at least one CPU, no more physical
// than logical CPUs, no more performance than physical CPUs, and no more available than total memory.
func (hw Specs) Normalize() Specs {
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

func (hw Specs) getPerformanceThreads() int {
	threadsPerCore := (hw.LogicalCPUs + hw.PhysicalCPUs - 1) / hw.PhysicalCPUs
	return min(hw.PerformanceCPUs*threadsPerCore, hw.LogicalCPUs)
}

func (hw Specs) getMemoryLimitMB(sharePercent int) int64 {
	return hw.TotalRAMMB * int64(sharePercent) / percentBase
}

func (hw Specs) getBudget(l level) int {
	if l.isSingleThread {
		return 1
	}
	return max(1, hw.getPerformanceThreads()*l.sharePercent/percentBase)
}

// ResolveCompute computes the active workload parameters for a given profile and hardware specs. Every pool it sizes occupies at most CoreBudget hardware threads.
// A profile that is none of the five is Balanced.
func ResolveCompute(profile string, hw Specs) ResolvedCompute {
	hw = hw.Normalize()
	l := getLevel(profile)
	budget := hw.getBudget(l)
	return ResolvedCompute{
		Profile:       l.profile,
		CoreBudget:    budget,
		Workers:       max(budget/l.workerDivisor, 1),
		MemoryLimitMB: hw.getMemoryLimitMB(l.sharePercent),
		Nice:          l.nice,
	}
}
