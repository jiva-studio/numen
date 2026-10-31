package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestfile"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const frontendOnlyManifest = `{
	"id": "md.numen.views.sketch",
	"name": "Sketch",
	"version": "1.0.0",
	"targets": ["desktop.editor"],
	"contentTypes": [{"type": "application/x-numen-sketch", "globs": ["*.sketch"]}],
	"readers": [{"id": "sketch", "contentTypes": ["application/x-numen-sketch"], "rank": "default"}],
	"settings": [{"key": "grid", "type": "boolean", "title": "Grid", "default": true}],
	"frontend": {"entrypoint": "dist/index.js"}
}`

func newFrontendOnlyHost(t *testing.T) (*Host, *int) {
	t.Helper()
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "views-sketch")
	if err := os.MkdirAll(pluginDir, 0o750); err != nil {
		t.Fatalf("make plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(frontendOnlyManifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	host := NewHost(testsupport.Machine, "desktop.editor", dir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	launches := 0
	host.SetLauncher(func(context.Context, manifestfile.DiscoveredPlugin, int, string) (*os.Process, io.ReadCloser, error) {
		launches++
		return nil, nil, errors.New("no process is started in this test")
	})
	return host, &launches
}

func TestAPluginWithoutABackendIsNeverLaunched(t *testing.T) {
	host, launches := newFrontendOnlyHost(t)
	var failures []pluginport.PluginFailure
	host.SetOnPluginFailed(func(failure pluginport.PluginFailure) { failures = append(failures, failure) })

	_, err := host.EnsureActive(t.Context(), "md.numen.views.sketch")

	var unavailable *pluginport.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != pluginport.PluginWithoutBackend {
		t.Fatalf("EnsureActive = %v, want a plugin without a backend", err)
	}
	if *launches != 0 {
		t.Fatalf("the host launched %d processes for a plugin without a backend", *launches)
	}
	if len(failures) != 0 {
		t.Fatalf("the host reported %v for a plugin without a backend", failures)
	}
	for _, info := range host.GetPluginInfos() {
		if info.Failure != nil {
			t.Fatalf("plugin %s is listed as failed: %v", info.ID, info.Failure.Reason)
		}
	}
}

func TestAnEventForTheFilesOfAFrontendPluginStartsNothing(t *testing.T) {
	host, launches := newFrontendOnlyHost(t)

	activated, err := host.TriggerEvent(t.Context(), "onFile:.sketch")
	if err != nil {
		t.Fatalf("TriggerEvent: %v", err)
	}
	if len(activated) != 0 || *launches != 0 {
		t.Fatalf("activated %v with %d launches, want none", activated, *launches)
	}
}

func TestPluginInfosMarkThePluginsWithAFrontend(t *testing.T) {
	host, _ := newFrontendOnlyHost(t)

	infos := host.GetPluginInfos()
	if len(infos) != 1 || !infos[0].HasFrontend {
		t.Fatalf("GetPluginInfos = %+v, want the sketch plugin with a frontend", infos)
	}
	want := []pluginport.PluginReaderInfo{{ID: "sketch", Extensions: []string{"sketch"}}}
	if !reflect.DeepEqual(infos[0].Readers, want) {
		t.Fatalf("Readers = %+v, want %+v", infos[0].Readers, want)
	}
}
