package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// createOutsideFolder is a folder on this machine holding those files, and
// where it is.
func createOutsideFolder(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// readArrived is what the vault holds at a path, and the test stops where it
// holds nothing.
func readArrived(t *testing.T, root, path string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("%s did not arrive: %v", path, err)
	}
	return string(body)
}

// Files dropped onto a folder are copied into it, whatever their kind, and the
// source files stay in place.
func TestFilesAreBroughtIntoTheFolderTheyWereLetGoOver(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})
	from := createOutsideFolder(t, map[string]string{"Cover.png": "PNG", "Notes.md": "# Notes\n"})
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, filesystem.ImportedFiles{})

	brought, err := bring.Execute(t.Context(), v, "physics", []string{
		filepath.Join(from, "Cover.png"),
		filepath.Join(from, "Notes.md"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 0 {
		t.Fatalf("files stayed outside: %v", brought.Errors)
	}

	if body := readArrived(t, v.Path, "physics/Cover.png"); body != "PNG" {
		t.Errorf("the picture arrived as %q", body)
	}
	if body := readArrived(t, v.Path, "physics/Notes.md"); body != "# Notes\n" {
		t.Errorf("the note arrived as %q", body)
	}
	if _, err := os.Stat(filepath.Join(from, "Cover.png")); err != nil {
		t.Errorf("what was let go of did not stay where it was: %v", err)
	}
}

// A folder arrives with everything under it, and an empty folder inside it is
// still a folder.
func TestAFolderIsBroughtInWhole(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, nil)
	from := createOutsideFolder(t, map[string]string{
		"scans/Cover.png":       "PNG",
		"scans/pages/One.png":   "ONE",
		"scans/pages/Two.png":   "TWO",
		"scans/read/Kelvin.txt": "KELVIN",
	})
	if err := os.MkdirAll(filepath.Join(from, "scans", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, filesystem.ImportedFiles{})

	brought, err := bring.Execute(t.Context(), v, "", []string{filepath.Join(from, "scans")})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 0 {
		t.Fatalf("files stayed outside: %v", brought.Errors)
	}
	// Four files, and the folder itself with the three under it.
	if len(brought.Landed) != 8 {
		t.Errorf("%d files and folders landed", len(brought.Landed))
	}

	if body := readArrived(t, v.Path, "scans/pages/Two.png"); body != "TWO" {
		t.Errorf("a file under the folder arrived as %q", body)
	}
	info, err := os.Stat(filepath.Join(v.Path, "scans", "empty"))
	if err != nil || !info.IsDir() {
		t.Errorf("the empty folder did not arrive: %v", err)
	}
}

// A name that already exists in the folder is a conflict only the user can
// resolve, so that file is skipped and the rest are copied.
func TestANameAlreadyThereIsRefusedAndTheRestArrive(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, map[string]string{"Cover.png": "MINE"})
	from := createOutsideFolder(t, map[string]string{"Cover.png": "THEIRS", "Kelvin.md": "# Kelvin\n"})
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, filesystem.ImportedFiles{})

	brought, err := bring.Execute(t.Context(), v, "", []string{
		filepath.Join(from, "Cover.png"),
		filepath.Join(from, "Kelvin.md"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(brought.Errors) != 1 || brought.Errors[0].Name != "Cover.png" {
		t.Fatalf("what stayed outside: %v", brought.Errors)
	}
	if !errors.Is(brought.Errors[0].Why, vaultport.ErrAlreadyExists) {
		t.Errorf("the error: want ErrAlreadyExists, got %v", brought.Errors[0].Why)
	}
	if body := readArrived(t, v.Path, "Cover.png"); body != "MINE" {
		t.Errorf("the file that was there was replaced with %q", body)
	}
	if body := readArrived(t, v.Path, "Kelvin.md"); body != "# Kelvin\n" {
		t.Errorf("the note beside it arrived as %q", body)
	}
}

// the disk lasted.
func TestAFolderContainingTheVaultIsRefused(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, nil)
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, filesystem.ImportedFiles{})

	brought, err := bring.Execute(t.Context(), v, "", []string{filepath.Dir(v.Path)})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 1 {
		t.Fatalf("what stayed outside: %v", brought.Errors)
	}
	if len(brought.Landed) != 0 {
		t.Errorf("%d files landed", len(brought.Landed))
	}
}
