package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const frontendOnlyManifest = `{
	"id": "md.numen.views.sketch",
	"name": "Sketch",
	"version": "1.0.0",
	"targets": ["desktop.editor"],
	"contributes": {
		"readers": [{"id": "sketch", "contentTypes": ["application/pdf"], "rank": "default"}]
	},
	"settings": [{"key": "grid", "type": "boolean", "title": "Grid", "default": true}],
	"frontend": {}
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
	host := openHost(testsupport.GetMachine(), "desktop.editor", dir)
	host.ContentTypes.SetContentTypeTable(contenttype.Open())
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	launches := 0
	host.Processes.SetLauncher(func(context.Context, manifest.Discovered, int, string) (*os.Process, io.ReadCloser, error) {
		launches++
		return nil, nil, errors.New("no process is started in this test")
	})
	return host, &launches
}

func TestAPluginWithoutABackendIsNeverLaunched(t *testing.T) {
	host, launches := newFrontendOnlyHost(t)
	var failures []plugindomain.Failure
	host.Failures.SetOnPluginFailed(func(failure plugindomain.Failure) { failures = append(failures, failure) })

	_, err := host.Activation.EnsureActive(t.Context(), "md.numen.views.sketch")

	var unavailable *plugindomain.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != plugindomain.WithoutBackend {
		t.Fatalf("EnsureActive = %v, want a plugin without a backend", err)
	}
	if *launches != 0 {
		t.Fatalf("the host launched %d processes for a plugin without a backend", *launches)
	}
	if len(failures) != 0 {
		t.Fatalf("the host reported %v for a plugin without a backend", failures)
	}
	for _, info := range host.Registry.GetPluginInfos() {
		if info.Failure != nil {
			t.Fatalf("plugin %s is listed as failed: %v", info.ID, info.Failure.Reason)
		}
	}
}

func TestAnEventForTheFilesOfAFrontendPluginStartsNothing(t *testing.T) {
	host, launches := newFrontendOnlyHost(t)

	activated, err := host.Events.TriggerEvent(t.Context(), "onFile:pdf")
	if err != nil {
		t.Fatalf("TriggerEvent: %v", err)
	}
	if len(activated) != 0 || *launches != 0 {
		t.Fatalf("activated %v with %d launches, want none", activated, *launches)
	}
}

func TestPluginInfosMarkThePartsOfThePlugins(t *testing.T) {
	host, _ := newFrontendOnlyHost(t)

	infos := host.Registry.GetPluginInfos()
	if len(infos) != 1 || !infos[0].HasFrontend || infos[0].HasBackend {
		t.Fatalf("GetPluginInfos = %+v, want the sketch plugin with a frontend and no backend", infos)
	}
	want := []plugindomain.ReaderInfo{{ID: "sketch", Extensions: []string{"pdf"}}}
	if !reflect.DeepEqual(infos[0].Readers, want) {
		t.Fatalf("Readers = %+v, want %+v", infos[0].Readers, want)
	}
}
