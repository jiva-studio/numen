package hardware

import (
	"runtime"
)

// PerformanceProfile is the high-level workload preset.
type PerformanceProfile = string

const (
	// ProfileEco minimizes power consumption and CPU heat (1 thread, small batch).
	ProfileEco PerformanceProfile = "eco"
	// ProfileBalanced provides responsive balance between speed and UI (50% CPU).
	ProfileBalanced PerformanceProfile = "balanced"
	// ProfileMaximum saturates 100% of detected hardware for maximum throughput.
	ProfileMaximum PerformanceProfile = "maximum"
	// ProfileCustom allows manual override of individual compute parameters.
	ProfileCustom PerformanceProfile = "custom"
)

// DefaultPerformanceProfile is the default workload preset for untouched installations.
const DefaultPerformanceProfile = ProfileBalanced

// Performance is the settings section for workload and resource calibration.
type Performance struct {
	// Profile is the active preset.
	Profile string `json:"profile"`
	// Custom holds user-defined overrides when Profile is ProfileCustom.
	Custom PerformanceCustom `json:"custom"`
}

// PerformanceCustom holds manual tuning parameters for compute workloads.
type PerformanceCustom struct {
	EmbeddingBatchSize int `json:"embedding_batch_size"`
	OcrThreads         int `json:"ocr_threads"`
	OcrSessions        int `json:"ocr_sessions"`
	LlmConcurrency     int `json:"llm_concurrency"`
}

// DefaultPerformance returns the default performance settings.
func DefaultPerformance() Performance {
	return Performance{
		Profile: string(DefaultPerformanceProfile),
		Custom: PerformanceCustom{
			EmbeddingBatchSize: 32,
			OcrThreads:         4,
			OcrSessions:        1,
			LlmConcurrency:     2,
		},
	}
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
	CoreBudget         int `json:"core_budget"`
	EmbeddingBatchSize int `json:"embedding_batch_size"`
	// OcrThreads is the intra-op thread count of one OCR session; OcrThreads times OcrSessions fits CoreBudget.
	OcrThreads  int `json:"ocr_threads"`
	OcrSessions int `json:"ocr_sessions"`
	// LlmConcurrency is the number of model requests in flight at once.
	LlmConcurrency int `json:"llm_concurrency"`
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

	// ocrSessionTarget is the number of concurrent OCR sessions a budget is split into before thread counts are considered.
	ocrSessionTarget = 4
	// ocrMaxThreadsPerSession is the intra-op thread count beyond which a session on small tensors gains nothing.
	ocrMaxThreadsPerSession = 8
	// ocrMaxSessions is the most OCR sessions opened, each holding its own copy of the models.
	ocrMaxSessions = 16

	// ecoEmbeddingBatch is the fixed embedding batch of Eco.
	ecoEmbeddingBatch = 16
	// balancedBatchMBPerItem and maximumBatchMBPerItem are the free megabytes that earn one more passage in an embedding batch.
	balancedBatchMBPerItem = 256
	maximumBatchMBPerItem  = 128
	balancedBatchMin       = 16
	balancedBatchMax       = 64
	maximumBatchMin        = 32
	maximumBatchMax        = 256
	// assumedFreeRAMPercent is the share of installed memory taken as free when only the installed size is known.
	assumedFreeRAMPercent = 50
	// assumedFreeRAMMB is the free memory taken when nothing about memory is known.
	assumedFreeRAMMB = 4096

	// LLM requests wait on the network, so their concurrency does not follow the core budget.
	ecoLlmConcurrency      = 1
	balancedLlmConcurrency = 2
	maximumLlmConcurrency  = 4

	customBatchDefault = 32
	customBatchMax     = 512
	customThreadsMax   = 64
	customSessionsMax  = 16
	customLlmMax       = 16

	// ecoCoreBudget is the single hardware thread Eco occupies.
	ecoCoreBudget = 1
	// The share of installed memory each profile lets the Go runtime use before the collector works harder. Eco leaves the most to other programs; Maximum leaves the least.
	ecoMemoryLimitPercent      = 25
	balancedMemoryLimitPercent = 50
	maximumMemoryLimitPercent  = 75

	// balancedWorkerShare divides the Balanced budget into the size of the general pool.
	balancedWorkerShare = 2
)

// getFreeRAMMB is the free memory the plan sizes batches from, falling back to the installed size and then to a constant.
func (hw Specs) getFreeRAMMB() int64 {
	switch {
	case hw.AvailableRAMMB > 0:
		return hw.AvailableRAMMB
	case hw.TotalRAMMB > 0:
		return hw.TotalRAMMB * assumedFreeRAMPercent / 100
	default:
		return assumedFreeRAMMB
	}
}

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

// getOcrShape splits a core budget into sessions and threads per session, so that their product never exceeds the budget.
func getOcrShape(budget int) (threads, sessions int) {
	sessions = max(ocrSessionTarget, (budget+ocrMaxThreadsPerSession-1)/ocrMaxThreadsPerSession)
	sessions = min(sessions, ocrMaxSessions, budget)
	threads = min(budget/sessions, ocrMaxThreadsPerSession)
	return max(threads, 1), max(sessions, 1)
}

func getEmbeddingBatch(freeMB int64, mbPerItem int64, least, most int) int {
	return int(min(max(freeMB/mbPerItem, int64(least)), int64(most)))
}

// ResolveCompute computes the active workload parameters for a given profile and hardware specs. Every pool it sizes occupies at most CoreBudget hardware threads, except a custom profile that sets both OCR counts.
func ResolveCompute(profile PerformanceProfile, hw Specs, custom PerformanceCustom) ResolvedCompute {
	hw = hw.normalize()
	freeMB := hw.getFreeRAMMB()

	switch profile {
	case ProfileEco:
		return ResolvedCompute{
			Profile:            ProfileEco,
			CoreBudget:         ecoCoreBudget,
			EmbeddingBatchSize: ecoEmbeddingBatch,
			OcrThreads:         1,
			OcrSessions:        1,
			LlmConcurrency:     ecoLlmConcurrency,
			Workers:            1,
			MemoryLimitMB:      hw.getMemoryLimitMB(ecoMemoryLimitPercent),
		}

	case ProfileMaximum:
		budget := getBudget(hw, maximumSharePercent)
		threads, sessions := getOcrShape(budget)
		return ResolvedCompute{
			Profile:            ProfileMaximum,
			CoreBudget:         budget,
			EmbeddingBatchSize: getEmbeddingBatch(freeMB, maximumBatchMBPerItem, maximumBatchMin, maximumBatchMax),
			OcrThreads:         threads,
			OcrSessions:        sessions,
			LlmConcurrency:     maximumLlmConcurrency,
			Workers:            budget,
			MemoryLimitMB:      hw.getMemoryLimitMB(maximumMemoryLimitPercent),
		}

	case ProfileCustom:
		return resolveCustom(hw, custom)

	case ProfileBalanced:
		fallthrough
	default:
		budget := getBudget(hw, balancedSharePercent)
		threads, sessions := getOcrShape(budget)
		return ResolvedCompute{
			Profile:            ProfileBalanced,
			CoreBudget:         budget,
			EmbeddingBatchSize: getEmbeddingBatch(freeMB, balancedBatchMBPerItem, balancedBatchMin, balancedBatchMax),
			OcrThreads:         threads,
			OcrSessions:        sessions,
			LlmConcurrency:     balancedLlmConcurrency,
			Workers:            max(budget/balancedWorkerShare, 1),
			MemoryLimitMB:      hw.getMemoryLimitMB(balancedMemoryLimitPercent),
		}
	}
}

// resolveCustom clamps the user's values to their bounds. A count left unset is derived from the Maximum budget; when both OCR counts are set they are kept as written.
func resolveCustom(hw Specs, custom PerformanceCustom) ResolvedCompute {
	budget := getBudget(hw, maximumSharePercent)

	batch := custom.EmbeddingBatchSize
	if batch < 1 {
		batch = customBatchDefault
	}
	batch = min(batch, customBatchMax)

	threads := min(custom.OcrThreads, customThreadsMax)
	sessions := min(custom.OcrSessions, customSessionsMax)
	switch {
	case threads < 1 && sessions < 1:
		threads, sessions = getOcrShape(budget)
	case threads < 1:
		threads = max(min(budget/sessions, ocrMaxThreadsPerSession), 1)
	case sessions < 1:
		sessions = max(min(budget/threads, ocrMaxSessions), 1)
	}

	balancedWorkers := max(getBudget(hw, balancedSharePercent)/balancedWorkerShare, 1)
	return ResolvedCompute{
		Profile:            ProfileCustom,
		CoreBudget:         budget,
		EmbeddingBatchSize: batch,
		OcrThreads:         threads,
		OcrSessions:        sessions,
		LlmConcurrency:     min(max(custom.LlmConcurrency, 1), customLlmMax),
		Workers:            balancedWorkers,
		MemoryLimitMB:      hw.getMemoryLimitMB(balancedMemoryLimitPercent),
	}
}
