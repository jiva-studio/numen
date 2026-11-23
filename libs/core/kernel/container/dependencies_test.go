package container

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestAProvisioningIsATaskUntilItIsReady(t *testing.T) {
	host := newTestPluginRoles(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Management.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
			t.Error(err)
		}
	})
	registry := provision.NewObservedRegistry(host.Supervision.Services.Observe)
	host.Supervision.Dependencies.Replace(registry)
	tasks := run.NewTasks()
	execution, _, _ := CreatePluginHost(host.Management, host.Supervision)
	reportDependencies(execution.Watch.dependencyReporter, tasks, nil)
	req := manifest.DependencyRequirement{ID: "whisper", Kind: manifest.DependencyKindModel}

	registry.Observe(req, plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning, BytesDownloaded: 5, BytesTotal: 10})
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "provision:whisper" || list[0].Title != "Downloading" || list[0].About != "whisper" ||
		list[0].Unit != work.Bytes || list[0].Count != 5 || list[0].Total != 10 || list[0].IsUserStarted {
		t.Fatalf("tasks = %+v", list)
	}
	registry.Observe(req, plugindomain.DependencyStatus{State: plugindomain.DependencyStateFailed, Error: "boom"})
	if list = tasks.List(); len(list) != 1 || list[0].Error != "boom" {
		t.Fatalf("a failure left tasks %+v", list)
	}
	registry.Observe(req, plugindomain.DependencyStatus{State: plugindomain.DependencyStateReady})
	if list = tasks.List(); len(list) != 0 {
		t.Errorf("a ready dependency left tasks %+v", list)
	}
}

func TestProvision_MockModelsReturnPlaceholders(t *testing.T) {
	t.Setenv(provision.EnvMockML, "true")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	registry := newDependencyRegistry(GetClock(NewResources()), nil, "")

	model := manifest.DependencyRequirement{
		ID: "pp-ocr-det", Kind: manifest.DependencyKindModel,
		SourceURL:      "https://models.invalid/det.onnx",
		ChecksumSHA256: "a431985659dc921974177a95adcfbb90fd9e51989a5e04d70d0b75f597b6e61d",
		Metadata:       map[string]string{"fileName": "pp-ocr-det.onnx", "envVar": "NUMEN_DEPENDENCY_PP_OCR_DET"},
	}
	resolved, err := registry.Provision(t.Context(), model, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := resolved.EnvVars["NUMEN_DEPENDENCY_PP_OCR_DET"]; got != resolved.PrimaryPath || filepath.Base(got) != "pp-ocr-det.onnx" {
		t.Errorf("the model is named %v, want the placeholder %s", resolved.EnvVars, resolved.PrimaryPath)
	}
	if info, err := os.Stat(resolved.PrimaryPath); err != nil || info.Size() != 0 {
		t.Errorf("the placeholder is %v, %v; want an empty file", info, err)
	}

	runtime := manifest.DependencyRequirement{ID: "sample-runtime", Kind: manifest.DependencyKindRuntime, Metadata: map[string]string{"envVar": "SAMPLE_RUNTIME_LIB"}}
	resolved, err = registry.Provision(t.Context(), runtime, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.EnvVars) != 1 || resolved.EnvVars["SAMPLE_RUNTIME_LIB"] != resolved.PrimaryPath {
		t.Errorf("the runtime is named %v", resolved.EnvVars)
	}
}

func TestAModelIsFetchedIntoTheFolderTheHostShares(t *testing.T) {
	t.Setenv(provision.EnvMockML, "")
	t.Setenv("NUMEN_MODELS_DIR", "")
	body := []byte("model weights")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	sum := sha256.Sum256(body)
	models := t.TempDir()
	req := manifest.DependencyRequirement{
		ID: "shared-model", Kind: manifest.DependencyKindModel,
		SourceURL: server.URL + "/model.onnx", ChecksumSHA256: hex.EncodeToString(sum[:]),
	}

	resolved, err := newDependencyRegistry(GetClock(NewResources()), nil, models).Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}

	if rel, err := filepath.Rel(models, resolved.PrimaryPath); err != nil || strings.HasPrefix(rel, "..") {
		t.Errorf("the model is at %s, not under %s", resolved.PrimaryPath, models)
	}
}

type heldProvisioner struct{ started chan struct{} }

func (heldProvisioner) CanProvision(manifest.DependencyRequirement) bool { return true }

func (heldProvisioner) Resolve(req manifest.DependencyRequirement) (plugindomain.ResolvedDependency, plugindomain.DependencyState, error) {
	return plugindomain.ResolvedDependency{ID: req.ID}, plugindomain.DependencyStateMissing, nil
}

func (p heldProvisioner) Provision(
	ctx context.Context, _ manifest.DependencyRequirement, onProgress func(plugindomain.DependencyStatus),
) (plugindomain.ResolvedDependency, error) {
	onProgress(plugindomain.DependencyStatus{BytesDownloaded: 1, BytesTotal: 2})
	close(p.started)
	<-ctx.Done()
	return plugindomain.ResolvedDependency{}, ctx.Err()
}

func startHeldDownload(t *testing.T) (*testPluginRoles, *run.Tasks, <-chan error) {
	t.Helper()
	host := newTestPluginRoles(testsupport.GetMachine(), "desktop.editor")
	provisioner := heldProvisioner{started: make(chan struct{})}
	host.Supervision.Dependencies.Replace(provision.NewObservedRegistry(host.Supervision.Services.Observe, provisioner))
	tasks := run.NewTasks()
	execution, _, _ := CreatePluginHost(host.Management, host.Supervision)
	reportDependencies(execution.Watch.dependencyReporter, tasks, nil)
	answer := make(chan error, 1)
	go func() {
		_, err := host.Supervision.Services.GetDependencies().Provision(
			t.Context(), manifest.DependencyRequirement{ID: "whisper", Kind: manifest.DependencyKindModel}, nil)
		answer <- err
	}()
	<-provisioner.started
	return host, tasks, answer
}

func TestTheCancelOfTheTaskOfADownloadStopsTheDownload(t *testing.T) {
	_, tasks, answer := startHeldDownload(t)

	if !tasks.Cancel("provision:whisper") {
		t.Fatalf("the task of the download cannot be canceled: %+v", tasks.List())
	}
	if err := <-answer; !errors.Is(err, context.Canceled) {
		t.Errorf("the caller received %v, want the cancellation", err)
	}
	if list := tasks.List(); len(list) != 0 {
		t.Errorf("a canceled download left tasks %+v", list)
	}
}

func TestStoppingThePluginsStopsTheDownloadsInProgress(t *testing.T) {
	host, _, answer := startHeldDownload(t)

	if err := StopPlugins(AdaptTestPlugins(host.Management, host.Supervision).Execution.Watch, context.WithoutCancel(t.Context())); err != nil {
		t.Fatal(err)
	}
	if err := <-answer; !errors.Is(err, context.Canceled) {
		t.Errorf("the caller received %v, want the cancellation", err)
	}
}
