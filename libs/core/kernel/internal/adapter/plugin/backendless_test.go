package plugin_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
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

func newFrontendOnlyHost(t *testing.T) (*container.PluginRoles, *int) {
	t.Helper()
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "views-sketch")
	if err := os.MkdirAll(pluginDir, 0o750); err != nil {
		t.Fatalf("make plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(frontendOnlyManifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", dir)
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
	_, err := host.Activation.EnsureActive(t.Context(), "md.numen.views.sketch")

	var unavailable *plugindomain.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != plugindomain.WithoutBackend {
		t.Fatalf("EnsureActive = %v, want a plugin without a backend", err)
	}
	if *launches != 0 {
		t.Fatalf("the host launched %d processes for a plugin without a backend", *launches)
	}
	if items := host.Health.List(); len(items) != 0 {
		t.Fatalf("the host listed %+v for a plugin without a backend", items)
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
