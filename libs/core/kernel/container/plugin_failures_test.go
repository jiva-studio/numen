package container

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const sampleManifest = `{
	"id": "md.numen.sample",
	"name": "Sample",
	"version": "1.0.0",
	"capabilities": [],
	"targets": ["desktop.editor"],
	"permissions": ["vault:read"],
	"services": ["md.numen.sample.SampleService"], "backend": {"entrypoint": "bin/sample"}
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
	host := openPluginRoles(t, "desktop.editor", root)
	host.Processes.SetLauncher(func(context.Context, manifest.Discovered, int, string) (*os.Process, io.ReadCloser, error) {
		return nil, nil, errors.New("exec format error")
	})
	return NewConfig(Config{PluginHost: AdaptPluginHost(host)})
}

func TestReportPluginFailures_TaskPersistsUntilRecovery(t *testing.T) {
	cfg := newFailingConfig(t)
	tasks := run.NewTasks()
	reportPluginFailures(plugins(cfg), tasks)

	failure, err := getPluginRetrier(plugins(cfg))(t.Context(), "md.numen.sample")
	if err != nil || failure == nil {
		t.Fatalf("retry = %+v, %v, want the plugin to fail again", failure, err)
	}

	list := tasks.List()
	if len(list) != 1 || list[0].About != "Sample" || list[0].Error != failure.Reason ||
		list[0].PluginID != "md.numen.sample" || list[0].PluginName != "Sample" {
		t.Fatalf("tasks = %+v, want one error naming Sample with %q", list, failure.Reason)
	}

	if err := cfg.PluginHost.ApplySetting(t.Context(), "md.numen.sample", "enabled", true); err != nil {
		t.Fatal(err)
	}
	if list := tasks.List(); len(list) != 0 {
		t.Fatalf("tasks after recovery = %+v, want none", list)
	}
}

func TestAnInstallationWithNoPluginHostRetriesNothing(t *testing.T) {
	cfg := NewConfig(Config{})
	reportPluginFailures(plugins(cfg), run.NewTasks())

	_, err := getPluginRetrier(plugins(cfg))(t.Context(), "md.numen.sample")

	var unavailable *plugin.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != plugin.NotInstalled {
		t.Fatalf("retry = %v, want a plugin that is not installed", err)
	}
}

func TestGetPluginNameReadsTheManifest(t *testing.T) {
	cfg := newFailingConfig(t)

	if got := pluginsGetPluginName(plugins(cfg), "md.numen.sample"); got != "Sample" {
		t.Errorf("getPluginName(md.numen.sample) = %q, want Sample", got)
	}
	if got := pluginsGetPluginName(plugins(cfg), "md.numen.absent"); got != "md.numen.absent" {
		t.Errorf("getPluginName(md.numen.absent) = %q, want the id", got)
	}
}
