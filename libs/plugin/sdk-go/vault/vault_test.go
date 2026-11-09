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
	if _, err := ValidatePath("relative.pdf"); !errors.Is(err, ErrEmptyVault) {
		t.Errorf("empty vault root: err = %v, want ErrEmptyVault", err)
	}
}

func TestValidatePathAcceptsDoubleDotInFilename(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("NUMEN_VAULT_PATH", vaultDir)

	clean, err := ValidatePath("a..b.pdf")
	if err != nil {
		t.Fatalf("legitimate double-dot filename rejected: %v", err)
	}
	want := filepath.Join(vaultDir, "a..b.pdf")
	if clean != want {
		t.Errorf("got %q, want %q", clean, want)
	}
}

func TestValidatePathRejectsTraversal(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("NUMEN_VAULT_PATH", vaultDir)

	for _, p := range []string{
		"../escape.pdf",
		"foo/../../etc/passwd",
		"sub/../..",
		"..",
	} {
		if _, err := ValidatePath(p); !errors.Is(err, ErrPathTraversal) {
			t.Errorf("path %q: expected ErrPathTraversal, got %v", p, err)
		}
	}
}

func TestValidatePathRejectsSymlinkEscapes(t *testing.T) {
	vaultDir := t.TempDir()
	outsideDir := t.TempDir()
	t.Setenv("NUMEN_VAULT_PATH", vaultDir)

	// Symlink pointing to nonexistent outside target.
	brokenLink := filepath.Join(vaultDir, "broken_link.pdf")
	if err := os.Symlink(filepath.Join(outsideDir, "nonexistent.pdf"), brokenLink); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePath("broken_link.pdf"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("symlink to missing outside target: expected ErrPathTraversal, got %v", err)
	}

	// Symlink pointing to existing outside target.
	outsideFile := filepath.Join(outsideDir, "secret.pdf")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	realLink := filepath.Join(vaultDir, "real_link.pdf")
	if err := os.Symlink(outsideFile, realLink); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePath("real_link.pdf"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("symlink to existing outside target: expected ErrPathTraversal, got %v", err)
	}

	// Symlink directory pointing outside.
	symDir := filepath.Join(vaultDir, "sym_dir")
	if err := os.Symlink(outsideDir, symDir); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePath("sym_dir/missing.pdf"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("missing file in outside symlinked dir: expected ErrPathTraversal, got %v", err)
	}
}

// A vault root passed to ValidatePathIn contains the path, whatever the environment holds.
func TestValidatePathInContainsThePathToTheRootItIsGiven(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NUMEN_VAULT_PATH", "")

	inside, err := ValidatePathIn(root, "notes/a.md")
	if err != nil || inside != filepath.Join(root, "notes", "a.md") {
		t.Errorf("a relative path = %q, %v; want it joined to the root", inside, err)
	}
	if _, err := ValidatePathIn(root, filepath.Join(t.TempDir(), "b.md")); !errors.Is(err, ErrOutsideVault) {
		t.Errorf("a path outside the root = %v, want ErrOutsideVault", err)
	}
	if _, err := ValidatePathIn("", "notes/a.md"); !errors.Is(err, ErrEmptyVault) {
		t.Errorf("empty vault root = %v, want ErrEmptyVault", err)
	}
	if _, err := ValidatePathIn("", "..", func(string) bool { return true }); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("dot-dot before allow = %v, want ErrPathTraversal", err)
	}
	allowedOut := filepath.Join(t.TempDir(), "allowed.md")
	gotAllowed, err := ValidatePathIn(root, allowedOut, func(string) bool { return false }, func(p string) bool { return p == allowedOut })
	if err != nil || gotAllowed != allowedOut {
		t.Errorf("allowed path = %q, %v; want %q", gotAllowed, err, allowedOut)
	}
	if _, err := ValidatePathIn(filepath.Join(t.TempDir(), "missing-vault"), "a.md"); err == nil {
		t.Error("expected error when vault root does not exist on disk")
	}
	if _, err := ValidatePathIn(root, filepath.Dir(root)); !errors.Is(err, ErrOutsideVault) {
		t.Errorf("parent dir of vault = %v, want ErrOutsideVault", err)
	}
	if _, err := ValidatePathIn(root, root+"/../escape.md"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("absolute path with dot-dot = %v, want ErrPathTraversal", err)
	}
	if err := checkRelativeContainment("/a/b", "/a", "rel"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("relative escape = %v, want ErrPathTraversal", err)
	}

	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	relValidLink := filepath.Join(root, "rel_valid.md")
	if err := os.Symlink("sub/new.md", relValidLink); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePathIn(root, "rel_valid.md"); err != nil {
		t.Errorf("relative broken symlink inside vault rejected: %v", err)
	}

	relEscapeLink := filepath.Join(root, "rel_escape.md")
	if err := os.Symlink("../outside_new.md", relEscapeLink); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePathIn(root, "rel_escape.md"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("relative broken symlink escaping vault = %v, want ErrPathTraversal", err)
	}

	regularFile := filepath.Join(root, "regular.md")
	if err := os.WriteFile(regularFile, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveSymlinkTarget(regularFile); err != nil || got != regularFile {
		t.Errorf("resolveSymlinkTarget(regularFile) = %q, %v", got, err)
	}
	if err := checkSymlinkContainment(root, filepath.Join(root, "rel_escape.md")); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("checkSymlinkContainment(rel_escape) = %v, want ErrPathTraversal", err)
	}
	badAncestorLink := filepath.Join(root, "bad_ancestor.md")
	if err := os.Symlink("relative-nonexistent-dir/file.md", badAncestorLink); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveSymlinkTarget(badAncestorLink); err != nil || got != filepath.Join(root, "relative-nonexistent-dir", "file.md") {
		t.Errorf("resolveSymlinkTarget(badAncestorLink) = %q, %v", got, err)
	}
	if _, err := ValidatePathIn(filepath.Join(root, "nonexistent-vault-root"), "note.md"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ValidatePathIn(nonexistent-vault-root) = %v, want os.ErrNotExist", err)
	}
}
