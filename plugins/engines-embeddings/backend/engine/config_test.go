package engine

import (
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/onnx"
)

func TestNewConfigOfBGEM3(t *testing.T) {
	info, local := newTestConfig(t, "Xenova/bge-m3")
	if info.Name != "Xenova/bge-m3" {
		t.Errorf("expected name Xenova/bge-m3, got %s", info.Name)
	}
	if info.Dimensions != 1024 {
		t.Errorf("expected 1024 dimensions, got %d", info.Dimensions)
	}
	if info.MaxTokens != 512 {
		t.Errorf("expected 512 max tokens, got %d", info.MaxTokens)
	}
	if info.Pooling != PoolCLS {
		t.Errorf("expected cls pooling, got %s", info.Pooling)
	}
	if info.Concurrency != 1 {
		t.Errorf("expected 1 concurrency, got %d", info.Concurrency)
	}
	if info.MinChunkLength != 0 {
		t.Errorf("expected 0 min chunk length, got %d", info.MinChunkLength)
	}
	if local.Name != "Xenova/bge-m3" {
		t.Errorf("expected Xenova/bge-m3, got %s", local.Name)
	}
	if local.File != "model_int8.onnx" {
		t.Errorf("expected model_int8.onnx, got %s", local.File)
	}
	if local.BatchTexts != 8 {
		t.Errorf("expected 8 batch texts, got %d", local.BatchTexts)
	}
	if local.Threads != 0 {
		t.Errorf("expected 0 threads, got %d", local.Threads)
	}
	if local.Sessions != 0 {
		t.Errorf("expected 0 sessions, got %d", local.Sessions)
	}
	if local.PassMemory != 0 {
		t.Errorf("expected 0 pass memory, got %d", local.PassMemory)
	}
	if !local.ShouldDownload {
		t.Errorf("expected download enabled by default")
	}
	if err := CheckIdentity(info, local.Name); err != nil {
		t.Fatalf("check identity failed: %v", err)
	}
}

func TestCheckIdentity(t *testing.T) {
	for _, tc := range []ModelInfo{
		{Dimensions: 1, MaxTokens: 1, Pooling: ""},
		{Dimensions: 1024, MaxTokens: 512, Pooling: PoolMean},
		{Dimensions: 1024, MaxTokens: 512, Pooling: PoolHead},
	} {
		if err := CheckIdentity(tc, "repo"); err != nil {
			t.Errorf("expected valid identity for %+v, got %v", tc, err)
		}
	}

	for _, tc := range []ModelInfo{
		{Dimensions: 0, MaxTokens: 100, Pooling: PoolMean},
		{Dimensions: -1, MaxTokens: 100, Pooling: PoolMean},
	} {
		if err := CheckIdentity(tc, "repo"); err == nil {
			t.Errorf("expected error for non-positive dimensions: %+v", tc)
		}
	}

	for _, tc := range []ModelInfo{
		{Dimensions: 1024, MaxTokens: 0, Pooling: PoolMean},
		{Dimensions: 1024, MaxTokens: -1, Pooling: PoolMean},
	} {
		if err := CheckIdentity(tc, "repo"); err == nil {
			t.Errorf("expected error for non-positive max tokens: %+v", tc)
		}
	}

	invalidPooling := ModelInfo{Dimensions: 1024, MaxTokens: 512, Pooling: "invalid"}
	if err := CheckIdentity(invalidPooling, "repo"); err == nil {
		t.Error("expected error for invalid pooling")
	}
}

func TestLocalModelGetPasses(t *testing.T) {
	prev := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(prev)

	// Set GOMAXPROCS to 5 so CPU fallback produces 3, distinguishing from returns of 1
	runtime.GOMAXPROCS(5)

	if passes := (LocalModel{Sessions: 3}).GetPasses(onnx.ProviderCPU); passes != 3 {
		t.Errorf("expected 3 passes for Sessions=3, got %d", passes)
	}
	if passes := (LocalModel{Sessions: 1}).GetPasses(onnx.ProviderCPU); passes != 1 {
		t.Errorf("expected 1 pass for Sessions=1, got %d", passes)
	}

	if passes := (LocalModel{}).GetPasses(onnx.ProviderCoreML); passes != 1 {
		t.Errorf("expected 1 pass for CoreML, got %d", passes)
	}
	if passes := (LocalModel{}).GetPasses(onnx.ProviderCUDA); passes != 1 {
		t.Errorf("expected 1 pass for CUDA, got %d", passes)
	}

	if passes := (LocalModel{PassBudget: 5}).GetPasses(onnx.ProviderCPU); passes != 5 {
		t.Errorf("expected 5 passes for PassBudget=5, got %d", passes)
	}
	if passes := (LocalModel{PassBudget: 1}).GetPasses(onnx.ProviderCPU); passes != 1 {
		t.Errorf("expected 1 pass for PassBudget=1, got %d", passes)
	}

	// Verify querying GOMAXPROCS does not mutate it (calling twice gives consistent result)
	runtime.GOMAXPROCS(4)
	if p1 := (LocalModel{}).GetPasses(onnx.ProviderCPU); p1 != 2 {
		t.Errorf("expected 2 passes for 4 cpus, got %d", p1)
	}
	if p2 := (LocalModel{}).GetPasses(onnx.ProviderCPU); p2 != 2 {
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
		if got := (LocalModel{}).GetPasses(onnx.ProviderCPU); got != tc.want {
			t.Errorf("GetPasses with GOMAXPROCS=%d: got %d, want %d", tc.cpus, got, tc.want)
		}
	}
}

func TestLocalModelGetProvider(t *testing.T) {
	if p := (LocalModel{}).GetProvider(); p != onnx.ProviderAuto {
		t.Errorf("expected default ProviderAuto, got %v", p)
	}
	if p := (LocalModel{Provider: "invalid"}).GetProvider(); p != onnx.ProviderAuto {
		t.Errorf("expected ProviderAuto for invalid provider, got %v", p)
	}
	if p := (LocalModel{Provider: "cpu"}).GetProvider(); p != onnx.ProviderCPU {
		t.Errorf("expected ProviderCPU, got %v", p)
	}
}

func TestLocalModelGetThreads(t *testing.T) {
	prev := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(prev)
	runtime.GOMAXPROCS(2)

	if th := (LocalModel{Threads: 4}).GetThreads(); th != 4 {
		t.Errorf("expected 4 threads, got %d", th)
	}
	if th := (LocalModel{Threads: 1}).GetThreads(); th != 1 {
		t.Errorf("expected 1 thread, got %d", th)
	}
	if th := (LocalModel{}).GetThreads(); th != onnx.GetDefaultThreads() {
		t.Errorf("expected default threads, got %d", th)
	}
}

func TestLocalModelGetAddress(t *testing.T) {
	m1 := LocalModel{Name: "Xenova/bge-m3"}
	if addr := m1.GetAddress(); addr != "local:Xenova/bge-m3/model.onnx" {
		t.Errorf("expected local:Xenova/bge-m3/model.onnx, got %s", addr)
	}

	m2 := LocalModel{Name: "model", Dir: "custom/path", File: "weights.onnx"}
	if addr := m2.GetAddress(); addr != "local:custom/path/weights.onnx" {
		t.Errorf("expected local:custom/path/weights.onnx, got %s", addr)
	}
}

func TestNewConfigIsPinnedForTheCuratedModels(t *testing.T) {
	for _, id := range []string{"Xenova/bge-m3", DefaultModel} {
		_, local := newTestConfig(t, id)
		requirePins(t, id, local)
	}
}

func requirePins(t *testing.T, id string, local LocalModel) {
	t.Helper()
	if len(local.Revision) != 40 {
		t.Errorf("%s: revision %q is not a commit", id, local.Revision)
	}
	for _, name := range []string{ModelFolder + "/" + local.File, TokenizerFile} {
		if len(local.Sums[name]) != 64 {
			t.Errorf("%s: %s has no sha256 pinned: %q", id, name, local.Sums[name])
		}
	}
}

func TestWithRepositoryDropsThePins(t *testing.T) {
	_, local := newTestConfig(t, "Xenova/bge-m3")
	other := local.WithRepository("owner/repo")
	if other.Name != "owner/repo" || other.Revision != "" || other.Sums != nil {
		t.Errorf("WithRepository() = %+v", other)
	}
	if local.Name != "Xenova/bge-m3" || local.Revision == "" || local.Sums == nil {
		t.Errorf("WithRepository changed the receiver: %+v", local)
	}
}

func TestLocalModel_GetProviderFallsBackToHostEnv(t *testing.T) {
	t.Setenv("NUMEN_ONNX_PROVIDER", "cpu")
	if p := (LocalModel{}).GetProvider(); p != onnx.ProviderCPU {
		t.Errorf("expected the host's ProviderCPU, got %v", p)
	}
	t.Setenv("NUMEN_ONNX_PROVIDER", "cuda")
	if p := (LocalModel{Provider: "cpu"}).GetProvider(); p != onnx.ProviderCPU {
		t.Errorf("expected the model's ProviderCPU, got %v", p)
	}
	if p := (LocalModel{}).GetProvider(); p != onnx.ProviderCUDA {
		t.Errorf("expected the host's ProviderCUDA, got %v", p)
	}
}

func TestNewConfigRunsOnePassWhateverTheCPUCount(t *testing.T) {
	prev := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(prev)
	runtime.GOMAXPROCS(16)

	info, local := newTestConfig(t, DefaultModel)

	if got := resolvePasses(local, onnx.ProviderCPU, info); got != 1 {
		t.Errorf("expected one pass at a time, got %d", got)
	}
}

func TestGetPassBytesAndMemoryBounds(t *testing.T) {
	if got := GetDefaultBatchTexts(); got != 8 {
		t.Errorf("GetDefaultBatchTexts() = %d, want 8", got)
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

	info := ModelInfo{Dimensions: 64, MaxTokens: 64}
	if got := resolvePasses(LocalModel{Sessions: 3, PassMemory: 1}, onnx.ProviderCPU, info); got != 3 {
		t.Errorf("resolvePasses with explicit Sessions=3: got %d, want 3", got)
	}
	if got := resolvePasses(LocalModel{PassBudget: 3, PassMemory: 1}, onnx.ProviderCPU, info); got != 1 {
		t.Errorf("resolvePasses with PassBudget=3 and tight memory: got %d, want 1", got)
	}
	if got := resolvePasses(LocalModel{BatchTexts: 1, PassBudget: 3, PassMemory: passBytes * 2}, onnx.ProviderCPU, info); got != 2 {
		t.Errorf("resolvePasses with BatchTexts=1 and memory for 2 passes: got %d, want 2", got)
	}
}

// newTestConfig returns the configuration of a model the plugin ships a profile for.
func newTestConfig(t *testing.T, id string) (ModelInfo, LocalModel) {
	t.Helper()
	profiles, err := GetShippedProfiles()
	if err != nil {
		t.Fatalf("GetShippedProfiles: %v", err)
	}
	profile, err := profiles.GetLocal(id)
	if err != nil {
		t.Fatalf("GetLocal(%s): %v", id, err)
	}
	return NewConfig(profile)
}
