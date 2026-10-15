package platform

import (
	"path/filepath"
	"testing"
)

func TestGetPluginsDirs(t *testing.T) {
	dirs := getPluginsDirs()
	if len(dirs) == 0 {
		t.Fatal("expected non-empty plugins directories")
	}

	customPath := filepath.Join(t.TempDir(), "custom-plugins")
	t.Setenv("NUMEN_PLUGINS_PATH", customPath)
	dirsWithCustom := getPluginsDirs()
	if len(dirsWithCustom) == 0 || dirsWithCustom[0] != customPath {
		t.Fatalf("expected custom path %s at head, got %v", customPath, dirsWithCustom)
	}
}

func TestGetConfig(t *testing.T) {
	cfg := GetConfig()
	if cfg.PluginHost == nil {
		t.Fatal("expected configured plugin host")
	}
}

func TestBuildMode(t *testing.T) {
	mode := BuildMode
	if mode != "monolith" && mode != "modular" {
		t.Fatalf("unexpected build mode: %s", mode)
	}
	if IsMonolith() != (mode == "monolith") {
		t.Fatalf("IsMonolith mismatch with mode %s", mode)
	}
}
