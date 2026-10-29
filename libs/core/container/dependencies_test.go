package container

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/usecase/run"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func TestAProvisioningIsATaskUntilItIsReady(t *testing.T) {
	host := plugin.NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	registry := newDependencies()
	host.SetDependencies(registry)
	tasks := run.NewTasks()
	Config{PluginHost: host}.ReportDependencies(tasks)
	req := pluginport.DependencyRequirement{ID: "whisper", Kind: pluginport.DependencyKindModel}

	registry.Observe(req, pluginport.DependencyStatus{State: pluginport.DependencyStateProvisioning, BytesDownloaded: 5, BytesTotal: 10})
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "provision:whisper" || list[0].Title != "Downloading whisper" ||
		list[0].Unit != run.Bytes || list[0].Count != 5 || list[0].Total != 10 || list[0].IsUserStarted {
		t.Fatalf("tasks = %+v", list)
	}
	registry.Observe(req, pluginport.DependencyStatus{State: pluginport.DependencyStateFailed, Error: "boom"})
	if list = tasks.List(); len(list) != 1 || list[0].Error != "boom" {
		t.Fatalf("a failure left tasks %+v", list)
	}
	registry.Observe(req, pluginport.DependencyStatus{State: pluginport.DependencyStateReady})
	if list = tasks.List(); len(list) != 0 {
		t.Errorf("a ready dependency left tasks %+v", list)
	}
}

func TestProvision_MockModelsReturnPlaceholders(t *testing.T) {
	t.Setenv(envMockML, "true")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	registry := newDependencies()

	model := pluginport.DependencyRequirement{
		ID: "pp-ocr-det", Kind: pluginport.DependencyKindModel,
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

	runtime := pluginport.DependencyRequirement{ID: pluginport.RuntimeONNX, Kind: pluginport.DependencyKindRuntime}
	resolved, err = registry.Provision(t.Context(), runtime, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.EnvVars["NUMEN_ONNX_RUNTIME_LIB"] != resolved.PrimaryPath || resolved.EnvVars["NUMEN_ONNX_PROVIDER"] != "cpu" {
		t.Errorf("the runtime is named %v", resolved.EnvVars)
	}
}
