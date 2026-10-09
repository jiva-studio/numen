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
