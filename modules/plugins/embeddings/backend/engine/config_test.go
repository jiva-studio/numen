package engine

import (
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

func TestDefaultBGEM3Config(t *testing.T) {
	info, local := DefaultBGEM3Config()
	if info.Name != "bge-m3" {
		t.Errorf("expected name bge-m3, got %s", info.Name)
	}
	if info.Dimensions != 1024 {
		t.Errorf("expected 1024 dimensions, got %d", info.Dimensions)
	}
	if info.MaxTokens != 8192 {
		t.Errorf("expected 8192 max tokens, got %d", info.MaxTokens)
	}
	if info.Pooling != PoolHead {
		t.Errorf("expected head pooling, got %s", info.Pooling)
	}
	if info.Concurrency != 4 {
		t.Errorf("expected 4 concurrency, got %d", info.Concurrency)
	}
	if local.Name != "Xenova/bge-m3" {
		t.Errorf("expected Xenova/bge-m3, got %s", local.Name)
	}
	if local.File != "model_int8.onnx" {
		t.Errorf("expected model_int8.onnx, got %s", local.File)
	}
	if local.BatchTexts != 32 {
		t.Errorf("expected 32 batch texts, got %d", local.BatchTexts)
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

func TestDefaultBGEM3ConfigIsPinned(t *testing.T) {
	_, local := DefaultBGEM3Config()
	if len(local.Revision) != 40 {
		t.Errorf("revision %q is not a commit", local.Revision)
	}
	for _, name := range []string{ModelFolder + "/" + local.File, TokenizerFile} {
		if len(local.Sums[name]) != 64 {
			t.Errorf("%s has no sha256 pinned: %q", name, local.Sums[name])
		}
	}
}

func TestWithRepositoryDropsThePins(t *testing.T) {
	_, local := DefaultBGEM3Config()
	other := local.WithRepository("owner/repo")
	if other.Name != "owner/repo" || other.Revision != "" || other.Sums != nil {
		t.Errorf("WithRepository() = %+v", other)
	}
	if local.Name != "Xenova/bge-m3" || local.Revision == "" || local.Sums == nil {
		t.Errorf("WithRepository changed the receiver: %+v", local)
	}
}

func TestLocalModelGetProviderFollowsTheHostUnlessTheModelNamesOne(t *testing.T) {
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
