package container

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/onnxruntime"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestAProvisioningIsATaskUntilItIsReady(t *testing.T) {
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
			t.Error(err)
		}
	})
	registry := provision.NewRegistry()
	host.Services.SetDependencies(registry)
	tasks := run.NewTasks()
	reportDependencies(plugins(NewConfig(Config{PluginHost: AdaptPluginHost((*PluginRoles)(host))})), tasks)
	req := manifest.DependencyRequirement{ID: "whisper", Kind: manifest.DependencyKindModel}

	registry.Observe(req, plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning, BytesDownloaded: 5, BytesTotal: 10})
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "provision:whisper" || list[0].Title != "Downloading whisper" ||
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
	t.Setenv(onnxruntime.EnvMockML, "true")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	registry := newDependencyRegistry("")

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

	runtime := manifest.DependencyRequirement{ID: plugindomain.RuntimeONNX, Kind: manifest.DependencyKindRuntime}
	resolved, err = registry.Provision(t.Context(), runtime, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.EnvVars["NUMEN_ONNX_RUNTIME_LIB"] != resolved.PrimaryPath || resolved.EnvVars["NUMEN_ONNX_PROVIDER"] != "cpu" {
		t.Errorf("the runtime is named %v", resolved.EnvVars)
	}
}

func TestAModelIsFetchedIntoTheFolderTheHostShares(t *testing.T) {
	t.Setenv(onnxruntime.EnvMockML, "")
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

	resolved, err := newDependencyRegistry(models).Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}

	if rel, err := filepath.Rel(models, resolved.PrimaryPath); err != nil || strings.HasPrefix(rel, "..") {
		t.Errorf("the model is at %s, not under %s", resolved.PrimaryPath, models)
	}
}
