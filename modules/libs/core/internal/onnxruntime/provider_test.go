package onnxruntime_test

import (
	"runtime"
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
	case onnxruntime.ProviderCPU, onnxruntime.ProviderCUDA, onnxruntime.ProviderCoreML:
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
}
