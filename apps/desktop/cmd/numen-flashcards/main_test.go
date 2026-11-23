package main

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/ensureindexed"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func getNotes() map[string]string {
	return map[string]string{
		"Leaf mold.md":   "# Leaf mold\n\nCompost made of fallen leaves alone.\n",
		"garden/Beds.md": "# Beds\n\n[[Leaf mold]] goes on in autumn.\n",
	}
}

type testInstallation struct {
	cfg      container.Config
	services platform.Installation
}

func makeConfigWithVaults(t *testing.T) (testInstallation, []domain.Vault) {
	t.Helper()

	state := t.TempDir()
	cfg := container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(state, "index.db"), RegistryPath: filepath.Join(state, "vaults.json")},
	}
	execution, preferences, endpoints := container.PreparePluginHost(nil, nil, nil, false)
	install := testInstallation{cfg: cfg, services: platform.Installation{
		Execution: execution, Preferences: preferences, Endpoints: endpoints, Resources: container.NewResources(),
	}}
	return install, []domain.Vault{addVaultOf(t, install, "One"), addVaultOf(t, install, "Two")}
}

func openShell(t *testing.T, install testInstallation) *shell.API {
	t.Helper()

	services := install.services
	opening := container.ShellOpening{
		Target: platform.TargetFlashcards,
		Log:    container.ShellLog{Out: io.Discard, Label: "flashcards", Notify: func(string) {}},
	}
	started, err := install.cfg.OpenShell(services.Execution, services.Preferences, services.Endpoints, services.Resources, t.Context(), opening)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		started.CloseVaults()
		if err := started.CloseIndex(); err != nil {
			t.Error(err)
		}
	})
	return started.API
}

func addVaultOf(t *testing.T, install testInstallation, name string) domain.Vault {
	t.Helper()

	root := t.TempDir()
	for name, body := range getNotes() {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	added, err := install.cfg.EnsureVault(install.services.Resources, root, name, false)
	if err != nil {
		t.Fatal(err)
	}
	return added
}

// The page is given whether a vault is indexed and the plugins' services it
// forwards.
func TestThePageIsGivenWhatItWorksThrough(t *testing.T) {
	install, _ := makeConfigWithVaults(t)

	api := openShell(t, install)

	if api == nil {
		t.Fatal("the page has no API")
	}
	if api.PluginServices == nil {
		t.Error("the page forwards no services")
	}
	for name, wired := range map[string]any{
		"indexed": api.Ensure,
	} {
		if reflect.ValueOf(wired).IsZero() {
			t.Errorf("the page has no %s", name)
		}
	}
}

// The page resolves the links of a note against the vaults the application lists.
func TestTheFlashcardsPageIsGivenTheLinksOfItsVaults(t *testing.T) {
	install, _ := makeConfigWithVaults(t)
	api := openShell(t, install)

	if api.ResolveLinks == nil {
		t.Error("the page has no links")
	}
	if api.Window.Name != platform.TargetFlashcards {
		t.Errorf("the window answers as %q", api.Window.Name)
	}
}

// The window binds the indexer of its vaults: a vault is read into the index, and the read is reported
// as a change. An indexer set after that is the one a vault not read yet is read with.
func TestTheVaultIndexerIsBoundToTheWindow(t *testing.T) {
	install, target := makeConfigWithVaults(t)
	api := openShell(t, install)

	for {
		state, err := api.EnsureIndexed(t.Context(), target[0])
		if err != nil {
			t.Fatal(err)
		}
		if state.Status == plugin.IndexFailed {
			t.Fatalf("the window read no vault into the index, got %+v", state)
		}
		if state.Status == plugin.Indexed {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	moved, err := api.WatchIndexChanges(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ensureindexed.NewReportRead(&api.Runs, &api.Changes).Execute(target[0])
	select {
	case <-moved:
	case <-time.After(time.Second):
		t.Error("a vault read did not report a change")
	}

	api.SetVaultIndexer(t.Context(), func(context.Context, domain.Vault) error {
		return errors.New("boom")
	})
	for {
		state, err := api.EnsureIndexed(t.Context(), target[1])
		if err != nil {
			t.Fatal(err)
		}
		if state.Reason == "boom" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// The plugins receive every vault the installation holds, each keyed by its
// identity, and a vault that is not in the index is read into it when a plugin
// requests it.
func TestThePluginsAreLentTheVaultsOfTheInstallation(t *testing.T) {
	install, target := makeConfigWithVaults(t)
	execution, preferences, endpoints, err := container.NewPluginHost("", install.services.Resources, func(error) {}, "md.numen.test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	install.services.Execution, install.services.Preferences, install.services.Endpoints = execution, preferences, endpoints
	api := openShell(t, install)
	api.SetVaultIndexer(t.Context(), nil)
	lent := endpoints.Endpoint.GetPluginVault()

	recognized, listErr := lent.Switch.GetVaults()
	linked, withErr := lent.Switch.With(target[1].ID)
	_, hasVault := lent.Switch.GetVault()
	state, stateErr := linked.Switch.EnsureIndexed(t.Context())

	if listErr != nil || len(recognized) != 2 || withErr != nil || hasVault {
		t.Fatalf("got %v (%v), %v and the window showing a vault: %v", recognized, listErr, withErr, hasVault)
	}
	if stateErr != nil || state.Status != plugin.IndexFailed {
		t.Errorf("a vault nothing reads into the index stands as %+v (%v)", state, stateErr)
	}
}

// The page receives the plugins of the flashcards target and is served the
// files of their frontends.
func TestThePageLoadsThePluginsOfItsTarget(t *testing.T) {
	install, _ := makeConfigWithVaults(t)
	const id = "md.numen.example"
	root := t.TempDir()
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"md.numen.example","name":"Example","version":"1.0.0","protocol":1,"targets":["desktop.flashcards"],"frontend":{},"contributes":{"feature":"Example","category":"views"}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	management, supervision := container.NewPluginRoles(container.GetClock(install.services.Resources), platform.TargetFlashcards, root)
	if err := management.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	install.services.Execution, install.services.Preferences, install.services.Endpoints = container.AdaptPluginHost(management, supervision)
	api := openShell(t, install)
	listed, err := api.Frontends.ListPlugins(t.Context(), connect.NewRequest(&v1.ListPluginsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if plugins := listed.Msg.GetPlugins(); len(plugins) != 1 || plugins[0].GetId() != id {
		t.Fatalf("listed plugins %v, want %s", plugins, id)
	}
	response := httptest.NewRecorder()
	api.NewHandler(http.NotFoundHandler()).ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugin/"+id+"/manifest.json", nil))
	if response.Code != http.StatusOK || response.Body.String() != manifest {
		t.Fatalf("plugin manifest response %d: %s", response.Code, response.Body.String())
	}
}
