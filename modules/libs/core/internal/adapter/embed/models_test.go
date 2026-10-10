package embed_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
)

func TestProfilesIncludesINT8Profiles(t *testing.T) {
	profiles := embed.Profiles()

	if len(profiles) < 4 {
		t.Fatalf("expected at least 4 profiles, got %d", len(profiles))
	}

	foundE5 := false
	foundBGE := false
	foundMiniLM := false

	for _, p := range profiles {
		switch p.Model.Name {
		case "e5-small-int8":
			foundE5 = true
			if p.Model.Dimensions != 384 {
				t.Errorf("expected 384 dims, got %d", p.Model.Dimensions)
			}
		case "bge-m3-int8":
			foundBGE = true
			if p.Model.Dimensions != 1024 {
				t.Errorf("expected 1024 dims, got %d", p.Model.Dimensions)
			}
		case "all-minilm-l6-v2-int8":
			foundMiniLM = true
			if p.Model.Dimensions != 384 {
				t.Errorf("expected 384 dims, got %d", p.Model.Dimensions)
			}
		}
	}

	if !foundE5 {
		t.Error("e5-small-int8 profile not found")
	}
	if !foundBGE {
		t.Error("bge-m3-int8 profile not found")
	}
	if !foundMiniLM {
		t.Error("all-minilm-l6-v2-int8 profile not found")
	}
}
func TestGetModelsReturnsProfilesAndAtomicWrites(t *testing.T) {
	held := embed.Config{
		Model: embed.Model{Name: "custom-model", Dimensions: 128, MaxTokens: 512, Pooling: embed.PoolHead},
		Indexing: embed.Provider{
			Use: embed.UseLocal,
		},
	}

	isFetched := func(embed.LocalModel) bool { return true }
	models := embed.GetModels(held, isFetched)

	if len(models) < len(embed.Profiles())+1 {
		t.Fatalf("expected at least profiles + 1 (configured), got %d", len(models))
	}

	var foundE5 bool
	for _, m := range models {
		if m.Name == "e5-small-int8" {
			foundE5 = true
			// Check atomic writes
			hasName := false
			for _, w := range m.Writes {
				if len(w.Path) == 2 && w.Path[0] == "model" && w.Path[1] == "name" {
					hasName = true
					if w.JSON != "\"e5-small-int8\"" {
						t.Errorf("expected e5-small-int8 name, got %s", w.JSON)
					}
				}
			}
			if !hasName {
				t.Errorf("missing model.name write")
			}
		}
	}

	if !foundE5 {
		t.Error("e5-small-int8 profile not found in GetModels")
	}
}
