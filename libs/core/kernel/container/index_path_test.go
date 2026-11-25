package container_test

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/indexfolder"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestOpenIndex_RejectsConfigWithoutIndexPath(t *testing.T) {
	cfg := container.NewTestInstallation(container.Config{
		Paths: container.Paths{RegistryPath: filepath.Join(t.TempDir(), "vaults.json")},
	}, nil)
	if _, err := container.OpenIndex(cfg.Config, cfg.Resources, t.Context()); !errors.Is(err, indexfolder.ErrNoPath) {
		t.Fatalf("OpenIndex without IndexPath: %v, want ErrNoIndexPath", err)
	}
	if _, err := cfg.OpenWindow(cfg.Execution, cfg.Preferences, cfg.Endpoints, cfg.Resources, t.Context(), container.WindowRequest{Named: "", Out: io.Discard}); !errors.Is(err, indexfolder.ErrNoPath) {
		t.Fatalf("OpenWindow without IndexPath: %v, want ErrNoIndexPath", err)
	}
}

func TestOpenIndexOpensTheNamedDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "index.db")
	db, err := container.OpenIndex(container.Config{
		Paths: container.Paths{IndexPath: path},
	}, container.NewResources(), t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)
	if db.File != path {
		t.Errorf("opened %q, want %q", db.File, path)
	}
}
