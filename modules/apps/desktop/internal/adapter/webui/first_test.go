package webui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

// A person who has added nothing is given somewhere to write.
//
// An application that answers "you have no vault" and stops asks somebody to
// read its manual before it will do anything at all.
func TestAnInstallationWithNoVaultIsGivenOne(t *testing.T) {
	documents := t.TempDir()
	t.Setenv("XDG_DOCUMENTS_DIR", documents)

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	opened, err := webui.Open(t.Context(), cfg, "", os.Stderr)
	if err != nil {
		t.Fatalf("an installation with no vault could not be opened: %v", err)
	}
	t.Cleanup(func() { opened.Close() })

	root := filepath.Join(documents, "numen")
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Fatalf("no folder to write in at %s: %v", root, err)
	}
	if opened.Showing().Path != root {
		t.Errorf("the vault opened is %s, want the one just made", opened.Showing().Path)
	}
	if opened.Showing().ID == "" {
		t.Error("the vault carries no identity, so moving its folder would lose it")
	}

	// It is registered, so the next start opens it rather than making another.
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	held, err := usecase.List{Registry: registry}.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 1 || held[0].Path != root {
		t.Errorf("the registry holds %+v, want the vault just made", held)
	}
}
