package vault

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	vaultDir := t.TempDir()

	if _, err := ValidatePath(""); !errors.Is(err, ErrEmptyPath) {
		t.Errorf("empty path: err = %v", err)
	}
	if _, err := ValidatePath("../escape.pdf"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("traversal: err = %v", err)
	}
	if _, err := ValidatePath("a/../b/../../escape.pdf"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("nested traversal: err = %v", err)
	}

	t.Setenv("NUMEN_VAULT_PATH", vaultDir)

	relPath := "books/sample.pdf"
	wantRel := filepath.Join(vaultDir, relPath)
	if clean, err := ValidatePath(relPath); err != nil || clean != wantRel {
		t.Errorf("relative path: got %q, err %v, want %q", clean, err, wantRel)
	}
	if clean, err := ValidatePath(vaultDir); err != nil || clean != vaultDir {
		t.Errorf("vault root: got %q, err %v", clean, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.pdf")
	if _, err := ValidatePath(outside); !errors.Is(err, ErrOutsideVault) {
		t.Errorf("outside the vault: err = %v", err)
	}

	t.Setenv("NUMEN_VAULT_PATH", "")
	if clean, err := ValidatePath("relative.pdf"); err != nil || clean != "relative.pdf" {
		t.Errorf("no vault root: got %q, err %v", clean, err)
	}
}

func TestATempDocumentIsAllowedOnlyWhenTheCallerAllowsIt(t *testing.T) {
	t.Setenv("NUMEN_VAULT_PATH", t.TempDir())
	tempDoc := filepath.Join(os.TempDir(), "numen-doc-test123")

	if _, err := ValidatePath(tempDoc); !errors.Is(err, ErrOutsideVault) {
		t.Errorf("temp document without allow: err = %v", err)
	}
	if clean, err := ValidatePath(tempDoc, IsTempDocument); err != nil || clean != tempDoc {
		t.Errorf("temp document: got %q, err %v", clean, err)
	}
	other := filepath.Join(os.TempDir(), "other-file")
	if _, err := ValidatePath(other, IsTempDocument); !errors.Is(err, ErrOutsideVault) {
		t.Errorf("other temp file: err = %v", err)
	}
}

// A vault root given to ValidatePathIn holds the path, whatever the environment says.
func TestValidatePathInHoldsThePathToTheRootItIsGiven(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NUMEN_VAULT_PATH", "")

	inside, err := ValidatePathIn(root, "notes/a.md")
	if err != nil || inside != filepath.Join(root, "notes", "a.md") {
		t.Errorf("a relative path = %q, %v; want it joined to the root", inside, err)
	}
	if _, err := ValidatePathIn(root, filepath.Join(t.TempDir(), "b.md")); !errors.Is(err, ErrOutsideVault) {
		t.Errorf("a path outside the root = %v, want ErrOutsideVault", err)
	}
}
