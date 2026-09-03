package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// adding is the use case over a registry of its own, and the registry, which a
// test reads to see what was written.
func adding(t *testing.T) (usecase.Add, *appstate.VaultRegistry) {
	t.Helper()
	registry := registryAt(t)
	return usecase.Add{
		Identity: filesystem.VaultIdentity{},
		Registry: registry,
		Now:      time.Now,
	}, registry
}

func registryAt(t *testing.T) *appstate.VaultRegistry {
	t.Helper()
	return appstate.At(filepath.Join(t.TempDir(), "state", "vaults.json"))
}

// folder makes a directory named name, under a parent of its own.
func folder(t *testing.T, name string) string {
	t.Helper()
	at := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(at, 0o755); err != nil {
		t.Fatal(err)
	}
	return at
}

func TestANameAnotherVaultHasGetsANumber(t *testing.T) {
	t.Parallel()
	add, registry := adding(t)
	var names []string
	for range 3 {
		v, err := add.Execute(folder(t, "notes"), "")
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, v.Name)
	}
	want := []string{"notes", "notes 2", "notes 3"}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("the vaults are called %v, want %v", names, want)
			break
		}
	}

	known, err := registry.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 3 {
		t.Errorf("the list holds %d vaults, want 3", len(known))
	}
}

func TestANameIsTakenWhateverItsCaseAndComposition(t *testing.T) {
	t.Parallel()
	// A folder name from a file picker arrives decomposed and the same name
	// typed at a command line arrives composed.
	add, _ := adding(t)
	if _, err := add.Execute(folder(t, "first"), norm.NFC.String("Café")); err != nil {
		t.Fatal(err)
	}
	v, err := add.Execute(folder(t, "second"), norm.NFD.String("café"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != norm.NFD.String("café")+" 2" {
		t.Errorf("the second vault is called %q, want the taken name numbered", v.Name)
	}
}

func TestAVaultInsideAnotherIsRefused(t *testing.T) {
	t.Parallel()
	add, _ := adding(t)
	outer := folder(t, "outer")
	if _, err := add.Execute(outer, ""); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "projects", "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(inner, ""); !errors.Is(err, usecase.ErrOverlaps) {
		t.Errorf("a folder inside a vault was answered %v", err)
	}
	if _, carriesOne, err := (filesystem.VaultIdentity{}).Of(inner); err != nil || carriesOne {
		t.Errorf("the refused folder was given an identity: %v %v", carriesOne, err)
	}
}

func TestAVaultHoldingAnotherIsRefused(t *testing.T) {
	t.Parallel()
	add, _ := adding(t)
	outer := folder(t, "outer")
	inner := filepath.Join(outer, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(inner, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(outer, ""); !errors.Is(err, usecase.ErrOverlaps) {
		t.Errorf("a folder holding a vault was answered %v", err)
	}
}

func TestAFolderReachedThroughASymlinkIsTheFolderItself(t *testing.T) {
	t.Parallel()
	add, registry := adding(t)
	target := folder(t, "notes")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("this machine does not make symlinks: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}

	first, err := add.Execute(resolved, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := add.Execute(link, "")
	if err != nil {
		t.Fatalf("the same folder under two names: %v", err)
	}
	if second.Path != resolved {
		t.Errorf("added at %s, want %s", second.Path, resolved)
	}
	if second.ID != first.ID {
		t.Errorf("one folder carries two identities: %s and %s", first.ID, second.ID)
	}

	known, err := registry.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 1 {
		t.Errorf("the list holds %v, want one vault", known)
	}
}
