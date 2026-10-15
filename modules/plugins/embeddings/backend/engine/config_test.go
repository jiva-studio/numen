package engine

import (
	"testing"
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
	if local.Name != "Xenova/bge-m3" {
		t.Errorf("expected Xenova/bge-m3, got %s", local.Name)
	}
	if local.File != "model_int8.onnx" {
		t.Errorf("expected model_int8.onnx, got %s", local.File)
	}
	if !local.ShouldDownload {
		t.Errorf("expected download enabled by default")
	}
	if err := CheckIdentity(info, local.Name); err != nil {
		t.Fatalf("check identity failed: %v", err)
	}
}

func TestCheckIdentityInvalid(t *testing.T) {
	invalidDim := ModelInfo{Dimensions: 0, MaxTokens: 100, Pooling: PoolMean}
	if err := CheckIdentity(invalidDim, "repo"); err == nil {
		t.Error("expected error for 0 dimensions")
	}

	invalidTokens := ModelInfo{Dimensions: 1024, MaxTokens: -1, Pooling: PoolMean}
	if err := CheckIdentity(invalidTokens, "repo"); err == nil {
		t.Error("expected error for negative tokens")
	}

	invalidPooling := ModelInfo{Dimensions: 1024, MaxTokens: 512, Pooling: "invalid"}
	if err := CheckIdentity(invalidPooling, "repo"); err == nil {
		t.Error("expected error for invalid pooling")
	}
}
