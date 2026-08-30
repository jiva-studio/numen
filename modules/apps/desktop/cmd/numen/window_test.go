package main

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// windowOn is a window open on a vault of this test's own.
func windowOn(t *testing.T) (*webui.Opened, container.Config) {
	t.Helper()

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	add := usecase.Add{Identity: cfg.VaultIdentity(), Registry: registry, Now: time.Now}
	if _, err := add.Execute(root, "one"); err != nil {
		t.Fatal(err)
	}

	opened, err := webui.Open(t.Context(), cfg, "one", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { opened.Close() })
	return opened, cfg
}
