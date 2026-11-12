package engine

import (
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/engine/onnx"
)

func TestLocalModelGetPasses(t *testing.T) {
	prev := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(prev)

	// Set GOMAXPROCS to 5 so CPU fallback produces 3, distinguishing from returns of 1
	runtime.GOMAXPROCS(5)

	if passes := getPasses(model.LocalModel{Sessions: 3}, onnx.ProviderCPU); passes != 3 {
		t.Errorf("expected 3 passes for Sessions=3, got %d", passes)
	}
	if passes := getPasses(model.LocalModel{Sessions: 1}, onnx.ProviderCPU); passes != 1 {
		t.Errorf("expected 1 pass for Sessions=1, got %d", passes)
	}

	if passes := getPasses(model.LocalModel{}, onnx.ProviderCoreML); passes != 1 {
		t.Errorf("expected 1 pass for CoreML, got %d", passes)
	}
	if passes := getPasses(model.LocalModel{}, onnx.ProviderCUDA); passes != 1 {
		t.Errorf("expected 1 pass for CUDA, got %d", passes)
	}

	if passes := getPasses(model.LocalModel{PassBudget: 5}, onnx.ProviderCPU); passes != 5 {
		t.Errorf("expected 5 passes for PassBudget=5, got %d", passes)
	}
	if passes := getPasses(model.LocalModel{PassBudget: 1}, onnx.ProviderCPU); passes != 1 {
		t.Errorf("expected 1 pass for PassBudget=1, got %d", passes)
	}

	// Verify querying GOMAXPROCS does not mutate it (calling twice gives consistent result)
	runtime.GOMAXPROCS(4)
	if p1 := getPasses(model.LocalModel{}, onnx.ProviderCPU); p1 != 2 {
		t.Errorf("expected 2 passes for 4 cpus, got %d", p1)
	}
	if p2 := getPasses(model.LocalModel{}, onnx.ProviderCPU); p2 != 2 {
		t.Errorf("expected 2 passes on repeated call, got %d", p2)
	}

	for _, tc := range []struct {
		cpus int
		want int
	}{
		{cpus: 1, want: 1},
		{cpus: 2, want: 1},
		{cpus: 3, want: 2},
		{cpus: 4, want: 2},
		{cpus: 5, want: 3},
	} {
		runtime.GOMAXPROCS(tc.cpus)
		if got := getPasses(model.LocalModel{}, onnx.ProviderCPU); got != tc.want {
			t.Errorf("GetPasses with GOMAXPROCS=%d: got %d, want %d", tc.cpus, got, tc.want)
		}
	}
}

func TestLocalModelGetProvider(t *testing.T) {
	if p := getProvider(model.LocalModel{}); p != onnx.ProviderAuto {
		t.Errorf("expected default ProviderAuto, got %v", p)
	}
	if p := getProvider(model.LocalModel{Provider: "invalid"}); p != onnx.ProviderAuto {
		t.Errorf("expected ProviderAuto for invalid provider, got %v", p)
	}
	if p := getProvider(model.LocalModel{Provider: "cpu"}); p != onnx.ProviderCPU {
		t.Errorf("expected ProviderCPU, got %v", p)
	}
}

func TestLocalModelGetThreads(t *testing.T) {
	prev := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(prev)
	runtime.GOMAXPROCS(2)

	if th := getThreads(model.LocalModel{Threads: 4}); th != 4 {
		t.Errorf("expected 4 threads, got %d", th)
	}
	if th := getThreads(model.LocalModel{Threads: 1}); th != 1 {
		t.Errorf("expected 1 thread, got %d", th)
	}
	if th := getThreads(model.LocalModel{}); th != onnx.GetDefaultThreads() {
		t.Errorf("expected default threads, got %d", th)
	}
}

func TestLocalModel_GetProviderFallsBackToHostEnv(t *testing.T) {
	t.Setenv("NUMEN_ONNX_PROVIDER", "cpu")
	if p := getProvider(model.LocalModel{}); p != onnx.ProviderCPU {
		t.Errorf("expected the host's ProviderCPU, got %v", p)
	}
	t.Setenv("NUMEN_ONNX_PROVIDER", "cuda")
	if p := getProvider(model.LocalModel{Provider: "cpu"}); p != onnx.ProviderCPU {
		t.Errorf("expected the model's ProviderCPU, got %v", p)
	}
	if p := getProvider(model.LocalModel{}); p != onnx.ProviderCUDA {
		t.Errorf("expected the host's ProviderCUDA, got %v", p)
	}
}

func TestNewConfigRunsOnePassWhateverTheCPUCount(t *testing.T) {
	prev := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(prev)
	runtime.GOMAXPROCS(16)

	info, local := newTestConfig(t, model.DefaultModel)

	if got := resolvePasses(local, onnx.ProviderCPU, info); got != 1 {
		t.Errorf("expected one pass at a time, got %d", got)
	}
}

func TestGetPassBytesAndMemoryBounds(t *testing.T) {
	if got := model.GetDefaultBatchTexts(); got != 8 {
		t.Errorf("model.GetDefaultBatchTexts() = %d, want 8", got)
	}
	if got := getPassBytes(0, 0, 0); got != 20 {
		t.Errorf("getPassBytes(0, 0, 0) = %d, want 20", got)
	}
	if got := getPassBytes(1, 1, 1); got != 20 {
		t.Errorf("getPassBytes(1, 1, 1) = %d, want 20", got)
	}
	// batchTexts=2, window=10, hiddenDim=64 -> heads=1, tokens=20, cells=200 -> 4 * (200 + 20*64*4) = 4 * 5320 = 21280
	if got := getPassBytes(2, 10, 64); got != 21280 {
		t.Errorf("getPassBytes(2, 10, 64) = %d, want 21280", got)
	}
	// batchTexts=1, window=1024, hiddenDim=64 -> heads=1, tokens=1024, cells=1048576 -> 4 * (1048576 + 1024*64*4) = 5242880
	if got := getPassBytes(1, 1024, 64); got != 5242880 {
		t.Errorf("getPassBytes(1, 1024, 64) = %d, want 5242880", got)
	}
	// batchTexts=1, window=1, hiddenDim=128 -> heads=2, tokens=1, cells=1 -> 4 * (1*2 + 1*128*4) = 4 * 514 = 2056
	if got := getPassBytes(1, 1, 128); got != 2056 {
		t.Errorf("getPassBytes(1, 1, 128) = %d, want 2056", got)
	}
	// hiddenDim=126 -> heads=1 (126/64=1, whereas 126/63=2) -> 4 * (1*1 + 1*126*4) = 4 * 505 = 2020
	if got := getPassBytes(1, 1, 126); got != 2020 {
		t.Errorf("getPassBytes(1, 1, 126) = %d, want 2020", got)
	}

	passBytes := getPassBytes(1, 64, 64)
	if got := getPassesWithinMemory(3, 0, 1, 64, 64); got != 3 {
		t.Errorf("getPassesWithinMemory with memory=0: got %d, want 3", got)
	}
	if got := getPassesWithinMemory(3, -1, 1, 64, 64); got != 3 {
		t.Errorf("getPassesWithinMemory with memory=-1: got %d, want 3", got)
	}
	if got := getPassesWithinMemory(3, 1, 1, 64, 64); got != 1 {
		t.Errorf("getPassesWithinMemory with memory=1: got %d, want 1", got)
	}
	if got := getPassesWithinMemory(3, passBytes*2, 1, 64, 64); got != 2 {
		t.Errorf("getPassesWithinMemory with memory=2*passBytes: got %d, want 2", got)
	}

	info := model.ModelInfo{Dimensions: 64, MaxTokens: 64}
	if got := resolvePasses(model.LocalModel{Sessions: 3, PassMemory: 1}, onnx.ProviderCPU, info); got != 3 {
		t.Errorf("resolvePasses with explicit Sessions=3: got %d, want 3", got)
	}
	if got := resolvePasses(model.LocalModel{PassBudget: 3, PassMemory: 1}, onnx.ProviderCPU, info); got != 1 {
		t.Errorf("resolvePasses with PassBudget=3 and tight memory: got %d, want 1", got)
	}
	if got := resolvePasses(model.LocalModel{BatchTexts: 1, PassBudget: 3, PassMemory: passBytes * 2}, onnx.ProviderCPU, info); got != 2 {
		t.Errorf("resolvePasses with BatchTexts=1 and memory for 2 passes: got %d, want 2", got)
	}
}

func newTestConfig(t *testing.T, id string) (model.ModelInfo, model.LocalModel) {
	t.Helper()
	profiles, err := model.GetShippedProfiles()
	if err != nil {
		t.Fatalf("GetShippedProfiles: %v", err)
	}
	profile, err := profiles.GetLocal(id)
	if err != nil {
		t.Fatalf("GetLocal(%s): %v", id, err)
	}
	return model.NewConfig(profile)
}

func getShippedProfiles(t *testing.T) *model.Profiles {
	t.Helper()
	table, err := model.GetShippedProfiles()
	if err != nil {
		t.Fatalf("GetShippedProfiles: %v", err)
	}
	return table
}
