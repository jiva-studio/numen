package settings

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

// HardwareSpecs represents the detected host machine hardware capabilities.
type HardwareSpecs struct {
	LogicalCPUs int   `json:"logical_cpus"`
	TotalRAMMB  int64 `json:"total_ram_mb"`
}

// ResolvedCompute holds the concrete, calibrated parameters to apply to worker pools.
type ResolvedCompute struct {
	Profile            PerformanceProfile `json:"profile"`
	EmbeddingBatchSize int                `json:"embedding_batch_size"`
	OcrThreads         int                `json:"ocr_threads"`
	OcrSessions        int                `json:"ocr_sessions"`
	LlmConcurrency     int                `json:"llm_concurrency"`
}

// DetectHardware inspects the current host machine capabilities.
func DetectHardware() HardwareSpecs {
	cpus := runtime.NumCPU()
	if cpus < 1 {
		cpus = 1
	}
	// Fallback baseline for total RAM when system-level RAM cannot be queried directly:
	// Estimate 8 GB baseline on standard desktop platforms.
	return HardwareSpecs{
		LogicalCPUs: cpus,
		TotalRAMMB:  8192,
	}
}

// ResolveCompute computes the active workload parameters for a given profile and hardware specs.
func ResolveCompute(profile PerformanceProfile, hw HardwareSpecs, custom PerformanceCustom) ResolvedCompute {
	if hw.LogicalCPUs < 1 {
		hw.LogicalCPUs = 1
	}
	ramGB := hw.TotalRAMMB / 1024
	if ramGB < 1 {
		ramGB = 1
	}

	switch profile {
	case ProfileEco:
		return ResolvedCompute{
			Profile:            ProfileEco,
			EmbeddingBatchSize: 16,
			OcrThreads:         1,
			OcrSessions:        1,
			LlmConcurrency:     1,
		}

	case ProfileMaximum:
		// Maximum saturates all logical CPUs and scales batch size with available RAM.
		batch := int(ramGB * 4)
		if batch < 32 {
			batch = 32
		} else if batch > 256 {
			batch = 256
		}

		ocrSessions := hw.LogicalCPUs / 2
		if ocrSessions < 1 {
			ocrSessions = 1
		}

		return ResolvedCompute{
			Profile:            ProfileMaximum,
			EmbeddingBatchSize: batch,
			OcrThreads:         hw.LogicalCPUs,
			OcrSessions:        ocrSessions,
			LlmConcurrency:     4,
		}

	case ProfileCustom:
		batch := custom.EmbeddingBatchSize
		if batch < 1 {
			batch = 32
		} else if batch > 512 {
			batch = 512
		}

		threads := custom.OcrThreads
		if threads < 1 {
			threads = 1
		} else if threads > 64 {
			threads = 64
		}

		sessions := custom.OcrSessions
		if sessions < 1 {
			sessions = 1
		} else if sessions > 16 {
			sessions = 16
		}

		concurrency := custom.LlmConcurrency
		if concurrency < 1 {
			concurrency = 1
		} else if concurrency > 16 {
			concurrency = 16
		}

		return ResolvedCompute{
			Profile:            ProfileCustom,
			EmbeddingBatchSize: batch,
			OcrThreads:         threads,
			OcrSessions:        sessions,
			LlmConcurrency:     concurrency,
		}

	case ProfileBalanced:
		fallthrough
	default:
		threads := hw.LogicalCPUs / 2
		if threads < 1 {
			threads = 1
		}

		batch := int(ramGB * 2)
		if batch < 16 {
			batch = 16
		} else if batch > 64 {
			batch = 64
		}

		sessions := hw.LogicalCPUs / 4
		if sessions < 1 {
			sessions = 1
		}

		return ResolvedCompute{
			Profile:            ProfileBalanced,
			EmbeddingBatchSize: batch,
			OcrThreads:         threads,
			OcrSessions:        sessions,
			LlmConcurrency:     2,
		}
	}
}
