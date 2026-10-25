package platform

import (
	"os"
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

func TestPluginDiscoveryReleaseMode(t *testing.T) {
	SetDev(false)
	defer SetDev(false)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cwdPlugins := filepath.Join(cwd, "plugins")
	cwdModulesPlugins := filepath.Join(cwd, "modules", "plugins")

	dirs := getPluginsDirs()
	for _, dir := range dirs {
		if dir == cwdPlugins || dir == cwdModulesPlugins {
			t.Fatalf("release discovery includes working directory path: %s", dir)
		}
	}
}

func TestPluginDiscoveryDevMode(t *testing.T) {
	SetDev(true)
	defer SetDev(false)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cwdPlugins := filepath.Join(cwd, "plugins")
	cwdModulesPlugins := filepath.Join(cwd, "modules", "plugins")

	dirs := getPluginsDirs()
	hasPlugins := false
	hasModulesPlugins := false
	for _, dir := range dirs {
		if dir == cwdPlugins {
			hasPlugins = true
		}
		if dir == cwdModulesPlugins {
			hasModulesPlugins = true
		}
	}
	if !hasPlugins || !hasModulesPlugins {
		t.Fatalf("expected dev discovery to include %s and %s, got %v", cwdPlugins, cwdModulesPlugins, dirs)
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
