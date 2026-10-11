package onnx

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The two models these tests size: an encoder of a small hidden width and a
// short window, and one of a large width and a long window.
var (
	smallModel = port.EmbeddingModel{Dimensions: 384, MaxTokens: 512}
	largeModel = port.EmbeddingModel{Dimensions: 1024, MaxTokens: 8192}
)

const testBatch = 32

func TestALargeModelHoldsMoreMemoryAPassThanASmallOne(t *testing.T) {
	small := getPassBytes(testBatch, smallModel.MaxTokens, smallModel.Dimensions)
	large := getPassBytes(testBatch, largeModel.MaxTokens, largeModel.Dimensions)
	if small <= 0 || large <= small {
		t.Errorf("a pass holds %d on the small model and %d on the large one", small, large)
	}
}

func TestAPassHoldsNoLessForAWiderBatchOrALongerWindow(t *testing.T) {
	base := getPassBytes(8, 256, 384)
	if got := getPassBytes(16, 256, 384); got < base {
		t.Errorf("a wider batch holds %d, below %d", got, base)
	}
	if got := getPassBytes(8, 8192, 384); got < base {
		t.Errorf("a longer window holds %d, below %d", got, base)
	}
}

func TestPassesAreBoundedByTheMemoryTheyMayHold(t *testing.T) {
	for name, model := range map[string]port.EmbeddingModel{"small": smallModel, "large": largeModel} {
		one := getPassBytes(testBatch, model.MaxTokens, model.Dimensions)
		for _, tc := range []struct {
			memory int64
			passes int
			want   int
		}{
			{0, 16, 16},
			{-1, 16, 16},
			{one / 2, 16, 1},
			{one, 16, 1},
			{one*3 + one/2, 16, 3},
			{one * 100, 16, 16},
			{one * 100, 128, 100},
		} {
			got := getPassesWithinMemory(tc.passes, tc.memory, testBatch, model.MaxTokens, model.Dimensions)
			if got != tc.want {
				t.Errorf("%s: %d passes in %d bytes: got %d, want %d", name, tc.passes, tc.memory, got, tc.want)
			}
		}
	}
}

func TestTheSamePassMemoryAllowsFewerPassesOfALargeModel(t *testing.T) {
	// A memory of ten small passes, on a machine whose plan runs 128 workers.
	memory := getPassBytes(testBatch, smallModel.MaxTokens, smallModel.Dimensions) * 10
	small := getPassesWithinMemory(128, memory, testBatch, smallModel.MaxTokens, smallModel.Dimensions)
	large := getPassesWithinMemory(128, memory, testBatch, largeModel.MaxTokens, largeModel.Dimensions)
	if small != 10 || large >= small {
		t.Errorf("the small model runs %d passes and the large one %d", small, large)
	}
}

func TestResolvedPassesTakeTheMemoryOfThePlanUnlessTheSettingsNameThem(t *testing.T) {
	one := getPassBytes(testBatch, smallModel.MaxTokens, smallModel.Dimensions)
	planned := embed.LocalModel{BatchTexts: testBatch, PassBudget: 16, PassMemory: one * 2}
	if got := resolvePasses(planned, onnxruntime.ProviderCPU, smallModel); got != 2 {
		t.Errorf("memory for two passes gave %d", got)
	}

	planned.PassMemory = 0
	if got := resolvePasses(planned, onnxruntime.ProviderCPU, smallModel); got != 16 {
		t.Errorf("unknown memory gave %d, want the plan's 16", got)
	}

	planned.PassMemory = one
	planned.Sessions = 5
	if got := resolvePasses(planned, onnxruntime.ProviderCPU, smallModel); got != 5 {
		t.Errorf("settings naming 5 passes gave %d", got)
	}

	planned.Sessions = 0
	if got := resolvePasses(planned, onnxruntime.ProviderCoreML, smallModel); got != 1 {
		t.Errorf("an accelerator gave %d passes", got)
	}
}
