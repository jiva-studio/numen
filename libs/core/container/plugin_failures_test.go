package container

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/usecase/run"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const sampleManifest = `{
	"id": "md.numen.sample",
	"name": "Sample",
	"version": "1.0.0",
	"capabilities": ["document"],
	"targets": ["desktop.editor"],
	"permissions": ["vault:read"],
	"backend": {"type": "process", "entrypoint": "bin/sample", "serviceName": "numen.v1.DocumentService"}
}`

func newFailingConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "sample")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(sampleManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewPluginHost("desktop.editor", root)
	host.SetLauncher(func(context.Context, plugindir.DiscoveredPlugin, int, string) (*os.Process, io.ReadCloser, error) {
		return nil, nil, errors.New("exec format error")
	})
	return Config{PluginHost: host}
}

func TestAPluginHeldOffExistsInTheTaskListUntilItRecovers(t *testing.T) {
	cfg := newFailingConfig(t)
	tasks := run.NewTasks()
	cfg.ReportPluginFailures(tasks)

	failure, err := cfg.GetPluginRetrier()(t.Context(), "md.numen.sample")
	if err != nil || failure == nil {
		t.Fatalf("retry = %+v, %v, want the plugin to fail again", failure, err)
	}

	list := tasks.List()
	if len(list) != 1 || list[0].About != "Sample" || list[0].Error != failure.Reason {
		t.Fatalf("tasks = %+v, want one error naming Sample with %q", list, failure.Reason)
	}

	if err := cfg.PluginHost.UpdatePluginSetting(t.Context(), "md.numen.sample", "enabled", true); err != nil {
		t.Fatal(err)
	}
	if list := tasks.List(); len(list) != 0 {
		t.Fatalf("tasks after recovery = %+v, want none", list)
	}
}

// An installation with no plugin host has nothing to retry and nothing to report.
func TestAnInstallationWithNoPluginHostRetriesNothing(t *testing.T) {
	cfg := Config{}
	cfg.ReportPluginFailures(run.NewTasks())

	_, err := cfg.GetPluginRetrier()(t.Context(), "md.numen.sample")

	var unavailable *pluginport.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != pluginport.PluginNotInstalled {
		t.Fatalf("retry = %v, want a plugin that is not installed", err)
	}
}
