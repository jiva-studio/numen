package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/indexsource"
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

func TestGetConfigNamesTheSystemKeychain(t *testing.T) {
	if GetConfigFor(TargetFlashcards).KeyStore == nil {
		t.Error("the desktop configuration names no key store")
	}
}

func TestGetConfigOffersThePluginsOfTheIndex(t *testing.T) {
	SetDev(false)
	dir := t.TempDir()
	t.Setenv("NUMEN_PLUGINS_PATH", dir)
	index := `{"plugins":[{"id":"org.example.offered","name":"Offered","version":"1.0.0"}]}`
	if err := os.WriteFile(filepath.Join(dir, indexsource.FileName), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}

	sources := GetConfigFor(TargetEditor).PluginCatalog.Sources

	if len(sources) != 1 {
		t.Fatalf("%d catalog sources, want the plugin index", len(sources))
	}
	entries, err := sources[0].List(t.Context())
	if err != nil || len(entries) != 1 || entries[0].ID != "org.example.offered" {
		t.Errorf("List = %+v, %v; want the plugin of the index", entries, err)
	}
}
