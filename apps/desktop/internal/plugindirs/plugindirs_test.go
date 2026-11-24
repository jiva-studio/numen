package plugindirs

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGetPluginsDirs(t *testing.T) {
	t.Setenv("NUMEN_DEV", "true")
	if !IsDev() {
		t.Fatal("expected IsDev() to be true when NUMEN_DEV=true")
	}
	t.Setenv("NUMEN_DEV", "1")
	t.Setenv("NUMEN_PLUGINS_PATH", "plugins")
	configHome := filepath.Join(t.TempDir(), "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "localappdata"))
	dirs := GetDirs()
	if len(dirs) == 0 {
		t.Fatal("expected non-empty plugins directories")
	}
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if wantConfigPlugins := filepath.Join(userConfigDir, "numen", "plugins"); !slices.Contains(dirs, wantConfigPlugins) {
		t.Errorf("dirs %v missing %s", dirs, wantConfigPlugins)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cwdPlugins := filepath.Join(cwd, "plugins")
	count := 0
	for _, d := range dirs {
		if d == cwdPlugins {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected %s once after deduplication, got %d in %v", cwdPlugins, count, dirs)
	}

	execPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	execDir := filepath.Dir(execPath)
	for _, want := range []string{
		filepath.Clean(filepath.Join(execDir, "plugins")),
		filepath.Clean(filepath.Join(execDir, "..", "PlugIns")),
		filepath.Clean(filepath.Join(execDir, "..", "Resources", "plugins")),
		filepath.Clean(filepath.Join(execDir, "..", "lib", "numen", "plugins")),
		filepath.Clean(filepath.Join(execDir, "..", "share", "numen", "plugins")),
		filepath.Clean(filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Numen", "plugins")),
	} {
		if !slices.Contains(dirs, want) {
			t.Errorf("dirs %v missing %s", dirs, want)
		}
	}

	t.Setenv("LOCALAPPDATA", "")
	customPath := filepath.Join(t.TempDir(), "custom-plugins")
	t.Setenv("NUMEN_PLUGINS_PATH", customPath)
	dirsWithCustom := GetDirs()
	if len(dirsWithCustom) == 0 || dirsWithCustom[0] != customPath {
		t.Fatalf("expected custom path %s at head, got %v", customPath, dirsWithCustom)
	}
	for _, d := range dirsWithCustom {
		if filepath.Base(filepath.Dir(d)) == "Numen" {
			t.Errorf("unexpected LOCALAPPDATA path %s when LOCALAPPDATA is empty", d)
		}
	}
}

func TestPluginDiscoveryReleaseMode(t *testing.T) {
	t.Setenv("NUMEN_DEV", "")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cwdPlugins := filepath.Join(cwd, "plugins")

	dirs := GetDirs()
	for _, dir := range dirs {
		if dir == cwdPlugins {
			t.Fatalf("release discovery includes working directory path: %s", dir)
		}
	}
}

func TestPluginDiscoveryDevMode(t *testing.T) {
	t.Setenv("NUMEN_DEV", "1")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cwdPlugins := filepath.Join(cwd, "plugins")

	dirs := GetDirs()
	hasPlugins := false
	for _, dir := range dirs {
		if dir == cwdPlugins {
			hasPlugins = true
		}
	}
	if !hasPlugins {
		t.Fatalf("expected dev discovery to include %s, got %v", cwdPlugins, dirs)
	}
}
