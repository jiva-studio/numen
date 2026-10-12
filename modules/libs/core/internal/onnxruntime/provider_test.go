package onnxruntime_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

func TestParseProvider(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  onnxruntime.Provider
		err   bool
	}{
		{"", onnxruntime.ProviderAuto, false},
		{"auto", onnxruntime.ProviderAuto, false},
		{"cpu", onnxruntime.ProviderCPU, false},
		{"coreml", onnxruntime.ProviderCoreML, false},
		{"core-ml", onnxruntime.ProviderCoreML, false},
		{"cuda", onnxruntime.ProviderCUDA, false},
		{"directml", onnxruntime.ProviderDirectML, false},
		{"direct-ml", onnxruntime.ProviderDirectML, false},
		{"dml", onnxruntime.ProviderDirectML, false},
		{"DML", onnxruntime.ProviderDirectML, false},
		{"rocm", onnxruntime.ProviderROCm, false},
		{"unknown-ep", onnxruntime.ProviderCPU, true},
	} {
		got, err := onnxruntime.ParseProvider(tc.input)
		if tc.err && err == nil {
			t.Errorf("ParseProvider(%q) expected error, got %v", tc.input, got)
		}
		if !tc.err && err != nil {
			t.Errorf("ParseProvider(%q) unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("ParseProvider(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestNewSessionOptionsCPU(t *testing.T) {
	engine, _, err := onnxruntime.Open(t.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		t.Skipf("onnx runtime not available: %v", err)
	}

	opts, used, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
		Provider: onnxruntime.ProviderCPU,
		Threads:  2,
	})
	if err != nil {
		t.Fatalf("NewSessionOptions(CPU) failed: %v", err)
	}
	defer opts.Destroy()

	if used != onnxruntime.ProviderCPU {
		t.Errorf("expected ProviderCPU, got %v", used)
	}
}

func TestNewSessionOptionsAuto(t *testing.T) {
	engine, _, err := onnxruntime.Open(t.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		t.Skipf("onnx runtime not available: %v", err)
	}

	opts, used, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
		Provider: onnxruntime.ProviderAuto,
		Threads:  0,
	})
	if err != nil {
		t.Fatalf("NewSessionOptions(Auto) failed: %v", err)
	}
	defer opts.Destroy()

	switch used {
	case onnxruntime.ProviderCPU, onnxruntime.ProviderCUDA, onnxruntime.ProviderCoreML, onnxruntime.ProviderDirectML:
	default:
		t.Errorf("unexpected provider for Auto: %v", used)
	}
}

func TestNewSessionOptionsExplicitUnsupportedProvider(t *testing.T) {
	engine, _, err := onnxruntime.Open(t.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		t.Skipf("onnx runtime not available: %v", err)
	}

	if runtime.GOOS != "darwin" {
		_, _, err = onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
			Provider: onnxruntime.ProviderCoreML,
		})
		if err == nil {
			t.Error("expected error for CoreML provider on non-darwin, got nil")
		}
	}

	if runtime.GOOS != "windows" {
		_, _, err = onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
			Provider: onnxruntime.ProviderDirectML,
		})
		if err == nil {
			t.Error("expected error for DirectML provider on non-windows, got nil")
		}
	}
}

func BenchmarkParseProvider(b *testing.B) {
	inputs := []string{"", "auto", "cpu", "coreml", "cuda", "directml", "dml", "rocm"}
	b.ResetTimer()
	for b.Loop() {
		for _, input := range inputs {
			_, _ = onnxruntime.ParseProvider(input)
		}
	}
}

func TestNewSessionOptionsCoreML(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("CoreML provider is supported on darwin/arm64")
	}

	engine, _, err := onnxruntime.Open(t.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		t.Skipf("onnx runtime not available: %v", err)
	}

	opts, used, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
		Provider: onnxruntime.ProviderCoreML,
		Threads:  0,
	})
	if err != nil {
		t.Fatalf("NewSessionOptions(CoreML) failed: %v", err)
	}
	defer opts.Destroy()

	if used != onnxruntime.ProviderCoreML {
		t.Errorf("expected ProviderCoreML, got %v", used)
	}
}

func BenchmarkNewSession(b *testing.B) {
	engine, _, err := onnxruntime.Open(b.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		b.Skipf("onnx runtime not available: %v", err)
	}

	opts, _, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
		Provider: onnxruntime.ProviderAuto,
	})
	if err != nil {
		b.Fatalf("NewSessionOptions failed: %v", err)
	}
	defer opts.Destroy()

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		b.Skipf("user cache dir not found: %v", err)
	}
	modelDir := filepath.Join(cacheDir, "numen", "models")
	entries, err := os.ReadDir(modelDir)
	if err != nil {
		b.Skip("no models found in cache")
	}
	var modelPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".onnx") {
			modelPath = filepath.Join(modelDir, e.Name())
			break
		}
	}
	if modelPath == "" {
		b.Skip("no onnx model found")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		session, err := onnxruntime.NewSession(engine, modelPath, opts)
		if err != nil {
			b.Fatal(err)
		}
		session.Destroy()
	}
}

func TestDefaultThreads(t *testing.T) {
	got := onnxruntime.DefaultThreads()
	want := min(max(1, runtime.GOMAXPROCS(0)), 8)
	if got != want {
		t.Errorf("DefaultThreads() = %d, want %d", got, want)
	}
	if got < 1 || got > 8 {
		t.Errorf("DefaultThreads() = %d, expected within [1, 8]", got)
	}
}

func BenchmarkNewSessionOptions(b *testing.B) {
	engine, _, err := onnxruntime.Open(b.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		b.Skipf("onnx runtime not available: %v", err)
	}

	for _, tc := range []struct {
		name    string
		threads int
	}{
		{"default", 0},
		{"threads=1", 1},
		{"threads=4", 4},
		{"threads=8", 8},
		{"threads=NumCPU", runtime.NumCPU()},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				opts, _, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
					Provider: onnxruntime.ProviderCPU,
					Threads:  tc.threads,
				})
				if err != nil {
					b.Fatal(err)
				}
				opts.Destroy()
			}
		})
	}
}

// ParseProvider reads an empty name as auto; a session whose settings leave the
// provider empty is not parsed from a name, and runs on the CPU.
func TestASessionWithNoProviderRunsOnTheCPU(t *testing.T) {
	engine, _, err := onnxruntime.Open(t.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		t.Skipf("onnx runtime not available: %v", err)
	}

	opts, used, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{Threads: 1})
	if err != nil {
		t.Fatalf("NewSessionOptions with no provider failed: %v", err)
	}
	defer opts.Destroy()

	if used != onnxruntime.ProviderCPU {
		t.Errorf("no provider ran as %v, want %v", used, onnxruntime.ProviderCPU)
	}
}

func TestSessionOptionsRefuseUnknownComputeUnits(t *testing.T) {
	engine, _, err := onnxruntime.Open(t.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		t.Skipf("onnx runtime not available: %v", err)
	}

	_, _, err = onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
		Provider:     onnxruntime.ProviderCPU,
		ComputeUnits: "NeuralEngine",
	})
	if err == nil {
		t.Error("expected an error for compute units the runtime does not know, got nil")
	}
}

func TestComputeUnitsLeaveTheCPUProviderAlone(t *testing.T) {
	engine, _, err := onnxruntime.Open(t.Context(), onnxruntime.Settings{Section: "indexing.recognition"})
	if err != nil {
		t.Skipf("onnx runtime not available: %v", err)
	}

	opts, used, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
		Provider:     onnxruntime.ProviderCPU,
		ComputeUnits: onnxruntime.ComputeUnitsCPUAndNeuralEngine,
	})
	if err != nil {
		t.Fatalf("NewSessionOptions(CPU, compute units) failed: %v", err)
	}
	defer opts.Destroy()

	if used != onnxruntime.ProviderCPU {
		t.Errorf("expected ProviderCPU, got %v", used)
	}
}

// The thread count follows the processors the process may use, up to eight, and
// asking for it leaves that number as it was.
func TestDefaultThreadsFollowsTheProcessorsUpToEight(t *testing.T) {
	previous := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	for _, tc := range []struct{ procs, want int }{
		{1, 1}, {3, 3}, {8, 8}, {9, 8}, {32, 8},
	} {
		runtime.GOMAXPROCS(tc.procs)
		if got := onnxruntime.DefaultThreads(); got != tc.want {
			t.Errorf("DefaultThreads() with %d processors = %d, want %d", tc.procs, got, tc.want)
		}
		if left := runtime.GOMAXPROCS(0); left != tc.procs {
			t.Errorf("DefaultThreads() left %d processors of %d", left, tc.procs)
		}
	}
}
