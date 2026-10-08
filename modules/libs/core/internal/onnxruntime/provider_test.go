package onnxruntime_test

import (
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
