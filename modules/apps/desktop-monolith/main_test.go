package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// newLinkedConfig is a configuration whose plugin host serves every plugin this binary links in,
// and no plugin from disk.
func newLinkedConfig(t *testing.T) container.Config {
	t.Helper()
	host := container.NewPluginHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Second) })
	registerPlugins(func(manifest []byte, serve serveFunc) {
		if err := host.RegisterBuiltIn(manifest, serve); err != nil {
			t.Fatalf("RegisterBuiltIn: %v", err)
		}
	})
	return container.Config{PluginHost: host}
}

// Each linked plugin is registered once, by the id its own manifest.json gives.
func TestEachPluginIsRegisteredOnceByItsOwnManifest(t *testing.T) {
	var ids []string
	registerPlugins(func(manifest []byte, serve serveFunc) {
		var declared struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(manifest, &declared); err != nil {
			t.Fatalf("a manifest does not parse: %v", err)
		}
		if serve == nil {
			t.Errorf("%s has no backend", declared.ID)
		}
		ids = append(ids, declared.ID)
	})

	slices.Sort(ids)
	if want := []string{"md.numen.epub", "md.numen.pdf"}; !slices.Equal(ids, want) {
		t.Fatalf("registered %v, want %v", ids, want)
	}
}

// A document is read by the PDF reader linked into this binary, through the plugin host.
func TestADocumentIsReadByTheLinkedReader(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "plugins", "reader-pdf", "backend", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	layer, err := newLinkedConfig(t).TextExtractor().Read(t.Context(), raw)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(layer.Pages) == 0 || strings.TrimSpace(layer.Text) == "" {
		t.Fatalf("read %d pages and %q", len(layer.Pages), layer.Text)
	}
}
