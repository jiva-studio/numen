package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

func TestResolveCompute_Eco(t *testing.T) {
	hw := settings.HardwareSpecs{LogicalCPUs: 16, TotalRAMMB: 32768}
	res := settings.ResolveCompute(settings.ProfileEco, hw, settings.PerformanceCustom{})

	if res.Profile != settings.ProfileEco {
		t.Errorf("expected ProfileEco, got %s", res.Profile)
	}
	if res.OcrThreads != 1 {
		t.Errorf("expected 1 OCR thread in Eco, got %d", res.OcrThreads)
	}
	if res.EmbeddingBatchSize > 16 {
		t.Errorf("expected batch size <= 16 in Eco, got %d", res.EmbeddingBatchSize)
	}
	if res.LlmConcurrency != 1 {
		t.Errorf("expected 1 LLM concurrency in Eco, got %d", res.LlmConcurrency)
	}
}

func TestResolveCompute_Balanced(t *testing.T) {
	hw := settings.HardwareSpecs{LogicalCPUs: 8, TotalRAMMB: 16384}
	res := settings.ResolveCompute(settings.ProfileBalanced, hw, settings.PerformanceCustom{})

	if res.Profile != settings.ProfileBalanced {
		t.Errorf("expected ProfileBalanced, got %s", res.Profile)
	}
	if res.OcrThreads != 4 {
		t.Errorf("expected 4 OCR threads (50%% of 8), got %d", res.OcrThreads)
	}
	if res.EmbeddingBatchSize < 16 || res.EmbeddingBatchSize > 64 {
		t.Errorf("expected batch size in [16, 64], got %d", res.EmbeddingBatchSize)
	}
	if res.LlmConcurrency != 2 {
		t.Errorf("expected 2 LLM concurrency, got %d", res.LlmConcurrency)
	}
}

func TestResolveCompute_Maximum(t *testing.T) {
	hw := settings.HardwareSpecs{LogicalCPUs: 12, TotalRAMMB: 32768}
	res := settings.ResolveCompute(settings.ProfileMaximum, hw, settings.PerformanceCustom{})

	if res.Profile != settings.ProfileMaximum {
		t.Errorf("expected ProfileMaximum, got %s", res.Profile)
	}
	if res.OcrThreads != 12 {
		t.Errorf("expected 12 OCR threads (100%% CPU), got %d", res.OcrThreads)
	}
	if res.EmbeddingBatchSize < 32 {
		t.Errorf("expected batch size >= 32, got %d", res.EmbeddingBatchSize)
	}
	if res.LlmConcurrency != 4 {
		t.Errorf("expected 4 LLM concurrency, got %d", res.LlmConcurrency)
	}
}

func TestResolveCompute_Custom(t *testing.T) {
	hw := settings.HardwareSpecs{LogicalCPUs: 8, TotalRAMMB: 16384}
	custom := settings.PerformanceCustom{
		EmbeddingBatchSize: 96,
		OcrThreads:         6,
		OcrSessions:        3,
		LlmConcurrency:     5,
	}
	res := settings.ResolveCompute(settings.ProfileCustom, hw, custom)

	if res.Profile != settings.ProfileCustom {
		t.Errorf("expected ProfileCustom, got %s", res.Profile)
	}
	if res.EmbeddingBatchSize != 96 {
		t.Errorf("expected 96 batch size, got %d", res.EmbeddingBatchSize)
	}
	if res.OcrThreads != 6 {
		t.Errorf("expected 6 OCR threads, got %d", res.OcrThreads)
	}
	if res.OcrSessions != 3 {
		t.Errorf("expected 3 OCR sessions, got %d", res.OcrSessions)
	}
	if res.LlmConcurrency != 5 {
		t.Errorf("expected 5 LLM concurrency, got %d", res.LlmConcurrency)
	}
}

func TestResolveCompute_CustomClamping(t *testing.T) {
	hw := settings.HardwareSpecs{LogicalCPUs: 4, TotalRAMMB: 8192}
	custom := settings.PerformanceCustom{
		EmbeddingBatchSize: 9999, // Should be clamped
		OcrThreads:         -5,   // Should be clamped
		LlmConcurrency:     0,    // Should be clamped
	}
	res := settings.ResolveCompute(settings.ProfileCustom, hw, custom)

	if res.EmbeddingBatchSize > 512 {
		t.Errorf("expected batch size to be clamped <= 512, got %d", res.EmbeddingBatchSize)
	}
	if res.OcrThreads < 1 {
		t.Errorf("expected OCR threads to be clamped >= 1, got %d", res.OcrThreads)
	}
	if res.LlmConcurrency < 1 {
		t.Errorf("expected LLM concurrency to be clamped >= 1, got %d", res.LlmConcurrency)
	}
}
