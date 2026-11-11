package editor_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowsettings "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Delta runs in the flashcards window only and asks for the group mistral.
const deltaManifest = `{
	"id": "md.numen.test.delta", "name": "Delta", "version": "1.0.0", "targets": ["desktop.flashcards"],
	"backend": {"entrypoint": "bin/delta"},
	"secrets": [{"name": "apiKey", "title": "API key", "group": "mistral"}]
}`

// betaWithoutSecrets is the manifest of beta after an update that dropped its declaration.
const betaWithoutSecrets = `{
	"id": "md.numen.test.beta", "name": "Beta", "version": "1.1.0", "targets": ["desktop.editor"],
	"backend": {"entrypoint": "bin/beta"}
}`

func openOverFile(t *testing.T, pluginsDir, file string, manifests map[string]string, keys pluginport.KeyStore, active []string) *window {
	t.Helper()
	for dir, body := range manifests {
		writeFile(t, filepath.Join(pluginsDir, dir, "manifest.json"), body)
	}

	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor", pluginsDir)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})

	probes := map[string]*probe{alphaID: {}, betaID: {}, gammaID: {}}
	ports := map[string]int{}
	processes := map[string]*os.Process{}
	for id, one := range probes {
		processes[id] = wiretest.StartSleepProcess(t).Process
		mux := http.NewServeMux()
		mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(one))
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		ports[id] = wiretest.GetPort(t, server)
	}
	host.Processes.SetLauncher(func(_ context.Context, p manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return processes[p.Manifest.ID], wiretest.GetReadyOutput(ports[p.Manifest.ID]), nil
	})

	configDir := t.TempDir()
	cfg := container.NewConfig(container.Config{
		PluginHost: container.AdaptPluginHost(host),
		KeyStore:   keys,
		Paths:      container.Paths{SettingsPath: file, IndexPath: filepath.Join(t.TempDir(), "index.db"), RegistryPath: filepath.Join(configDir, "vaults.json"), PluginCachePath: filepath.Join(t.TempDir(), "plugins")},
	})
	cfg = applySettings(t, cfg)

	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, testsupport.GetEpoch()); err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.AddVault(root, "secrets"); err != nil {
		t.Fatal(err)
	}
	installation, _, err := openRegistered(cfg, t.Context(), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	path, handler := numenv1connect.NewSettingsServiceHandler(editor.HandlerOf[*windowsettings.Handler](installation))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	w := &window{
		client:   numenv1connect.NewSettingsServiceClient(server.Client(), server.URL),
		host:     host,
		probes:   probes,
		file:     file,
		pluginsd: pluginsDir,
	}
	if keys, ok := keys.(*secretKeys); ok {
		w.keys = keys
	}
	for _, id := range active {
		if _, err := host.Activation.EnsureActive(t.Context(), id); err != nil {
			t.Fatal(err)
		}
		wiretest.WaitUntil(t, id+" is configured", func() bool { return probes[id].count() > 0 })
	}
	return w
}

func openFlashcardsOver(t *testing.T, pluginsDir, file string, keys pluginport.KeyStore) *pluginhost.Host {
	t.Helper()
	host := pluginhost.New(testsupport.GetMachine(), "desktop.flashcards", pluginsDir)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	configDir := t.TempDir()
	cfg := container.NewConfig(container.Config{
		PluginHost: container.AdaptPluginHost(host),
		KeyStore:   keys,
		Paths:      container.Paths{SettingsPath: file, IndexPath: filepath.Join(t.TempDir(), "index.db"), RegistryPath: filepath.Join(configDir, "vaults.json"), PluginCachePath: filepath.Join(t.TempDir(), "plugins")},
	})
	applySettings(t, cfg)
	return host
}

func allPlugins() map[string]string {
	return map[string]string{"alpha": alphaManifest, "beta": betaManifest, "gamma": gammaManifest}
}

func TestTheNextStartRemovesThePermissionsOfAnUninstalledPlugin(t *testing.T) {
	pluginsDir := t.TempDir()
	file := filepath.Join(t.TempDir(), "numen.json")
	writeFile(t, file, `{}`)
	keys := &secretKeys{items: map[string]string{}}

	t.Run("first run", func(t *testing.T) {
		first := openOverFile(t, pluginsDir, file, allPlugins(), keys, []string{alphaID, betaID, gammaID})
		id := first.create(t, "mistral", "Default", "invented-key-1")
		first.allow(t, id, gammaID, "apiKey", "mistral")
	})

	if err := os.RemoveAll(filepath.Join(pluginsDir, "gamma")); err != nil {
		t.Fatal(err)
	}
	restarted := openOverFile(t, pluginsDir, file, map[string]string{"alpha": alphaManifest, "beta": betaManifest}, keys, []string{alphaID, betaID})

	if raw := restarted.readFile(t); strings.Contains(raw, gammaID) {
		t.Errorf("the settings file still holds a permission of the uninstalled plugin:\n%s", raw)
	}
}

func TestAReinstalledPluginReceivesNoSecretBeforeItIsAllowed(t *testing.T) {
	pluginsDir := t.TempDir()
	file := filepath.Join(t.TempDir(), "numen.json")
	writeFile(t, file, `{}`)
	keys := &secretKeys{items: map[string]string{}}

	var id string
	t.Run("first run", func(t *testing.T) {
		first := openOverFile(t, pluginsDir, file, allPlugins(), keys, []string{alphaID, betaID, gammaID})
		id = first.create(t, "mistral", "Default", "invented-key-1")
		first.allow(t, id, gammaID, "apiKey", "mistral")
		first.awaitIDs(t, gammaID, "apiKey", id)
	})
	if err := os.RemoveAll(filepath.Join(pluginsDir, "gamma")); err != nil {
		t.Fatal(err)
	}
	t.Run("restart without the plugin", func(t *testing.T) {
		openOverFile(t, pluginsDir, file, map[string]string{"alpha": alphaManifest, "beta": betaManifest}, keys, []string{alphaID, betaID})
	})

	reinstalled := openOverFile(t, pluginsDir, file, allPlugins(), keys, []string{alphaID, betaID, gammaID})
	if got := reinstalled.probes[gammaID].ids(t, "apiKey"); len(got) != 0 {
		t.Fatalf("the reinstalled plugin received %v, which the person never allowed after the reinstall", got)
	}
	reinstalled.allow(t, id, gammaID, "apiKey", "mistral")
	reinstalled.awaitIDs(t, gammaID, "apiKey", id)
}

func TestAnUpdateThatDropsADeclarationRemovesItsPermissionsAtTheNextStart(t *testing.T) {
	pluginsDir := t.TempDir()
	file := filepath.Join(t.TempDir(), "numen.json")
	writeFile(t, file, `{}`)
	keys := &secretKeys{items: map[string]string{}}

	t.Run("first run", func(t *testing.T) {
		first := openOverFile(t, pluginsDir, file, allPlugins(), keys, []string{alphaID, betaID, gammaID})
		id := first.create(t, "openai", "Default", "invented-key-1")
		first.allow(t, id, betaID, "apiKey", "openai")
		first.allow(t, id, alphaID, "apiKey", "openrouter")
	})

	t.Run("restart after the update", func(t *testing.T) {
		updated := allPlugins()
		updated["beta"] = betaWithoutSecrets
		restarted := openOverFile(t, pluginsDir, file, updated, keys, []string{alphaID, betaID, gammaID})
		if raw := restarted.readFile(t); strings.Contains(raw, betaID) {
			t.Errorf("the settings file still holds a permission of the dropped declaration:\n%s", raw)
		}
	})

	back := openOverFile(t, pluginsDir, file, allPlugins(), keys, []string{alphaID, betaID, gammaID})
	if got := back.probes[betaID].ids(t, "apiKey"); len(got) != 0 {
		t.Errorf("the declaration came back and received %v without being allowed again", got)
	}
}

func TestThePermissionsOfAnEditorOnlyPluginSurviveTheStartOfTheFlashcardsHost(t *testing.T) {
	pluginsDir := t.TempDir()
	file := filepath.Join(t.TempDir(), "numen.json")
	writeFile(t, file, `{}`)
	keys := &secretKeys{items: map[string]string{}}
	writeFile(t, filepath.Join(pluginsDir, "delta", "manifest.json"), deltaManifest)

	editorWindow := openOverFile(t, pluginsDir, file, allPlugins(), keys, []string{alphaID, betaID, gammaID})
	id := editorWindow.create(t, "mistral", "Default", "invented-key-1")
	editorWindow.allow(t, id, gammaID, "apiKey", "mistral")

	flashcards := openFlashcardsOver(t, pluginsDir, file, keys)
	if _, found := flashcards.Registry.GetPlugin(gammaID); found {
		t.Fatalf("the flashcards host knows %s, which targets the editor only", gammaID)
	}

	if raw := editorWindow.readFile(t); !strings.Contains(raw, gammaID) {
		t.Errorf("the start of the flashcards host removed the permission of an editor-only plugin:\n%s", raw)
	}
}

func TestAPluginThatBecomesActiveDuringASecretOperationIsConfiguredOnce(t *testing.T) {
	pluginsDir := t.TempDir()
	file := filepath.Join(t.TempDir(), "numen.json")
	writeFile(t, file, `{}`)
	keys := &secretKeys{items: map[string]string{}}
	w := openOverFile(t, pluginsDir, file, allPlugins(), keys, []string{alphaID, betaID})

	started := make(chan struct{})
	release := make(chan struct{})
	operationDone := make(chan error, 1)
	go func() {
		close(started)
		<-release
		operationDone <- w.host.Config.Refresh(t.Context())
	}()
	<-started

	activationDone := make(chan error, 1)
	go func() {
		_, err := w.host.Activation.EnsureActive(t.Context(), gammaID)
		activationDone <- err
	}()
	wiretest.WaitUntil(t, "the plugin is active", func() bool {
		status, found := w.host.Registry.GetPluginStatus(gammaID)
		return found && status.IsActive
	})
	close(release)
	if err := <-operationDone; err != nil {
		t.Fatal(err)
	}
	if err := <-activationDone; err != nil {
		t.Fatal(err)
	}

	if got := w.probes[gammaID].count(); got != 1 {
		t.Errorf("the plugin received %d Configure calls, want 1", got)
	}
}

// failingDelete is a key store that cannot remove a value.
type failingDelete struct{ *secretKeys }

func (failingDelete) DeleteKey(context.Context, string) error { return os.ErrPermission }

func TestAFailedDeleteNeverOrphansAValue(t *testing.T) {
	pluginsDir := t.TempDir()
	file := filepath.Join(t.TempDir(), "numen.json")
	writeFile(t, file, `{}`)
	inner := &secretKeys{items: map[string]string{}}
	w := openOverFile(t, pluginsDir, file, allPlugins(), failingDelete{inner}, []string{alphaID, betaID, gammaID})
	id := w.create(t, "openrouter", "Work", "invented-key-1")
	w.allow(t, id, alphaID, "apiKey", "openrouter")

	_, err := w.client.DeleteSecret(t.Context(), connect.NewRequest(&v1.DeleteSecretRequest{Id: id}))
	if err == nil {
		t.Fatal("DeleteSecret succeeded though the key store could not remove the value")
	}

	hasEntry := w.find(t, id) != nil
	hasValue := len(inner.snapshot()) > 0
	if !hasEntry && hasValue {
		t.Errorf("the entry is gone and %d value(s) stay in the key store, where nothing names them", len(inner.snapshot()))
	}
}
