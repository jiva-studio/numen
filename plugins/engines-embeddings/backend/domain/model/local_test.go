package model

import (
	"testing"
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
		{Dimensions: 1024, MaxTokens: 512, Pooling: PoolCLS},
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
