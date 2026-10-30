package container_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

func TestOpenIndexRefusesConfigWithoutIndexPath(t *testing.T) {
	cfg := container.Config{RegistryPath: filepath.Join(t.TempDir(), "vaults.json")}
	if _, err := cfg.OpenIndex(t.Context()); !errors.Is(err, container.ErrNoIndexPath) {
		t.Fatalf("OpenIndex without IndexPath: %v, want ErrNoIndexPath", err)
	}
	if _, err := cfg.GetEditorAssembly(t.Context()); !errors.Is(err, container.ErrNoIndexPath) {
		t.Fatalf("GetEditorAssembly without IndexPath: %v, want ErrNoIndexPath", err)
	}
}

func TestOpenIndexOpensTheNamedDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "index.db")
	db, err := container.Config{IndexPath: path}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)
	if db.Path() != path {
		t.Errorf("opened %q, want %q", db.Path(), path)
	}
}
