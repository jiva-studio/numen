package engine

import "testing"

func TestGetModelProfileListsEveryModelOfTheSettings(t *testing.T) {
	cases := []struct {
		model string
		want  ModelProfile
	}{
		{"Xenova/bge-m3", ModelProfile{1024, 8192, PoolHead, 0.30}},
		{"bge-m3", ModelProfile{1024, 8192, PoolHead, 0.30}},
		{"baai/bge-m3", ModelProfile{1024, 8192, PoolHead, 0.30}},
		{"Xenova/multilingual-e5-small", ModelProfile{384, 512, PoolMean, 0.81}},
		{"Xenova/bge-small-en-v1.5", ModelProfile{384, 512, PoolHead, 0}},
		{"openai/text-embedding-3-small", ModelProfile{1536, 8191, PoolHead, 0.30}},
		{"text-embedding-3-small", ModelProfile{1536, 8191, PoolHead, 0.30}},
		{"openai/text-embedding-3-large", ModelProfile{3072, 8191, PoolHead, 0.30}},
		{"text-embedding-3-large", ModelProfile{3072, 8191, PoolHead, 0.30}},
		{"openai/text-embedding-ada-002", ModelProfile{1536, 8191, PoolHead, 0.75}},
		{"text-embedding-ada-002", ModelProfile{1536, 8191, PoolHead, 0.75}},
		{"qwen/qwen3-embedding-8b", ModelProfile{4096, 32768, PoolHead, 0}},
	}
	for _, tc := range cases {
		if got := GetModelProfile(tc.model); got != tc.want {
			t.Errorf("%s: profile = %+v, want %+v", tc.model, got, tc.want)
		}
		if !HasModelProfile(tc.model) {
			t.Errorf("%s: HasModelProfile = false", tc.model)
		}
	}
}

func TestGetModelProfileOfAnUnknownModelKeepsTheBGEM3Numbers(t *testing.T) {
	got := GetModelProfile("owner/unknown")
	want := ModelProfile{Dimensions: 1024, MaxTokens: 8192, Pooling: PoolHead, FloorHint: 0}
	if got != want {
		t.Errorf("profile = %+v, want %+v", got, want)
	}
	if HasModelProfile("owner/unknown") {
		t.Error("HasModelProfile = true for an unlisted model")
	}
}

func TestEveryProfileIsAValidIdentity(t *testing.T) {
	for model, profile := range modelProfiles {
		identity := ModelInfo{Name: model, Dimensions: profile.Dimensions, MaxTokens: profile.MaxTokens, Pooling: profile.Pooling}
		if err := CheckIdentity(identity, model); err != nil {
			t.Errorf("%s: %v", model, err)
		}
		if profile.FloorHint < 0 || profile.FloorHint >= 1 {
			t.Errorf("%s: floor hint %v outside [0, 1)", model, profile.FloorHint)
		}
	}
}
