package platform

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/jiva-studio/numen/libs/plugin/catalog/indexsource"
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
	dirs := getPluginsDirs()
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
	dirsWithCustom := getPluginsDirs()
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

	dirs := getPluginsDirs()
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

	dirs := getPluginsDirs()
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

func TestNewPluginServices(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	services := NewPluginServices(TargetEditor)
	if services.Execution == nil || services.Preferences == nil || services.Endpoints == nil {
		t.Fatal("expected configured plugin host")
	}
	if services.Resources.PluginCatalog.InstallDir == "" || services.Resources.PluginCatalog.CacheDir == "" {
		t.Errorf("expected InstallDir and CacheDir to be set, got %+v", services.Resources.PluginCatalog)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	noHome := NewPluginServices(TargetEditor)
	if noHome.Resources.PluginCatalog.InstallDir != "" || noHome.Resources.PluginCatalog.CacheDir != "" {
		t.Errorf("expected empty InstallDir and CacheDir without HOME, got %+v", noHome.Resources.PluginCatalog)
	}
}

func TestPluginServicesUseTheSystemKeychain(t *testing.T) {
	services := NewPluginServices(TargetFlashcards)
	if services.Resources.KeyStore == nil {
		t.Error("the desktop configuration names no key store")
	}

	keyring.MockInitWithError(keyring.ErrUnsupportedPlatform)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	withDir := getKeyStore()
	if withDir == nil {
		t.Fatal("expected non-nil key store when settings path is available")
	}
	if err := withDir.SetKey(t.Context(), "test-key", "test-val"); err != nil {
		t.Fatalf("SetKey with configured fallback dir failed: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	withoutDir := getKeyStore()
	if withoutDir == nil {
		t.Fatal("expected non-nil zero key store when settings path fails")
	}
	if err := withoutDir.SetKey(t.Context(), "test-key", "test-val"); err == nil {
		t.Fatal("expected SetKey on zero key store without keychain to fail")
	}
}

func TestReportPluginFailure(t *testing.T) {
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = origStderr
	})

	reportPluginFailure("discover plugins", nil)
	reportPluginHost(errors.New("bad manifest"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), "numen: plugin host: bad manifest\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGetConfigListsThePluginsOfTheIndex(t *testing.T) {
	t.Setenv("NUMEN_DEV", "")
	dir := t.TempDir()
	t.Setenv("NUMEN_PLUGINS_PATH", dir)
	index := `{"plugins":[{"id":"org.example.offered","name":"Offered","version":"1.0.0"}]}`
	if err := os.WriteFile(filepath.Join(dir, indexsource.FileName), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}

	services := NewPluginServices(TargetEditor)
	sources := services.Resources.PluginCatalog.Sources

	if len(sources) != 2 {
		t.Fatalf("%d catalog sources, want the plugin folders and the plugin index", len(sources))
	}
	entries, err := sources[1].List(t.Context())
	if err != nil || len(entries) != 1 || entries[0].ID != "org.example.offered" {
		t.Errorf("List = %+v, %v; want the plugin of the index", entries, err)
	}

	work := t.TempDir()
	devPlugin := filepath.Join(work, "plugins", "sample")
	if err := os.MkdirAll(devPlugin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devPlugin, "manifest.json"), []byte(`{"id":"md.numen.dev","name":"Dev","version":"1.0.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)

	t.Setenv("NUMEN_DEV", "")
	if list, err := getIndexSource(nil).List(t.Context()); err != nil || len(list) != 0 {
		t.Errorf("expected no dev manifest fallback in release mode, got %+v (%v)", list, err)
	}
	t.Setenv("NUMEN_DEV", "1")
	if list, err := getIndexSource(nil).List(t.Context()); err != nil || len(list) != 1 {
		t.Errorf("expected dev manifest fallback in dev mode, got %+v (%v)", list, err)
	}
}
