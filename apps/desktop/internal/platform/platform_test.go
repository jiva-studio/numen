package platform

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/jiva-studio/numen/apps/desktop/internal/keychain"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/indexsource"
)

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
	if services.Resources.Stores.Keys == nil {
		t.Error("the desktop configuration names no key store")
	}

	keyring.MockInitWithError(keyring.ErrUnsupportedPlatform)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	withDir := getKeyStore()
	if withDir == nil {
		t.Fatal("expected non-nil key store when settings path is available")
	}
	withDir.UsePlaintextFile()
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

func TestCredentialsFileChoiceIsAppliedToTheKeyStore(t *testing.T) {
	keyring.MockInitWithError(keyring.ErrUnsupportedPlatform)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	path, err := settings.GetPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := settings.WriteCredentialsFile(path, settings.CredentialsFilePlaintext); err != nil {
		t.Fatal(err)
	}
	store := getKeyStore()

	applyCredentialsFile(store, &keychain.Passphrase{})

	if err := store.SetKey(t.Context(), "test-key", "test-val"); err != nil {
		t.Fatalf("SetKey after the plaintext choice: %v", err)
	}
}

func TestACredentialsFileWithoutAChoiceRefusesKeys(t *testing.T) {
	keyring.MockInitWithError(keyring.ErrUnsupportedPlatform)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	store := getKeyStore()

	applyCredentialsFile(store, &keychain.Passphrase{})

	if err := store.SetKey(t.Context(), "test-key", "test-val"); !errors.Is(err, keychain.ErrFileBackendNotConfirmed) {
		t.Fatalf("SetKey = %v, want ErrFileBackendNotConfirmed", err)
	}
}
