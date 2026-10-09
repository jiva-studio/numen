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
