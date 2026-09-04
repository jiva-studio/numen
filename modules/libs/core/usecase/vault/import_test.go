package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// outside is a folder on this machine holding those files, and where it is.
func outside(t *testing.T, files map[string]string) string {
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

// arrived is what the vault holds at a path, and the test stops where it holds
// nothing.
func arrived(t *testing.T, root, path string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("%s did not arrive: %v", path, err)
	}
	return string(body)
}

// Files a person lets go of over a folder are copied into it, whatever kind of
// file they are, and what they let go of stays where it was.
func TestFilesAreBroughtIntoTheFolderTheyWereLetGoOver(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, map[string]string{"physics/Entropy.md": "# Entropy\n"})
	from := outside(t, map[string]string{"Cover.png": "PNG", "Notes.md": "# Notes\n"})
	bring := vaults.Import{Writers: filesystem.VaultWriters{}, Files: filesystem.ImportedFiles{}}

	brought, err := bring.Execute(t.Context(), v, "physics", []string{
		filepath.Join(from, "Cover.png"),
		filepath.Join(from, "Notes.md"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Refused) != 0 {
		t.Fatalf("files were refused: %v", brought.Refused)
	}

	if body := arrived(t, v.Path, "physics/Cover.png"); body != "PNG" {
		t.Errorf("the picture arrived as %q", body)
	}
	if body := arrived(t, v.Path, "physics/Notes.md"); body != "# Notes\n" {
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
	from := outside(t, map[string]string{
		"scans/Cover.png":       "PNG",
		"scans/pages/One.png":   "ONE",
		"scans/pages/Two.png":   "TWO",
		"scans/read/Kelvin.txt": "KELVIN",
	})
	if err := os.MkdirAll(filepath.Join(from, "scans", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	bring := vaults.Import{Writers: filesystem.VaultWriters{}, Files: filesystem.ImportedFiles{}}

	brought, err := bring.Execute(t.Context(), v, "", []string{filepath.Join(from, "scans")})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Refused) != 0 {
		t.Fatalf("files were refused: %v", brought.Refused)
	}
	// Four files, and the folder itself with the three under it.
	if len(brought.Landed) != 8 {
		t.Errorf("%d files and folders landed", len(brought.Landed))
	}

	if body := arrived(t, v.Path, "scans/pages/Two.png"); body != "TWO" {
		t.Errorf("a file under the folder arrived as %q", body)
	}
	info, err := os.Stat(filepath.Join(v.Path, "scans", "empty"))
	if err != nil || !info.IsDir() {
		t.Errorf("the empty folder did not arrive: %v", err)
	}
}

// A name the folder already carries is a question only the person can answer,
// so that file stays outside and the rest arrive.
func TestANameAlreadyThereIsRefusedAndTheRestArrive(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, map[string]string{"Cover.png": "MINE"})
	from := outside(t, map[string]string{"Cover.png": "THEIRS", "Kelvin.md": "# Kelvin\n"})
	bring := vaults.Import{Writers: filesystem.VaultWriters{}, Files: filesystem.ImportedFiles{}}

	brought, err := bring.Execute(t.Context(), v, "", []string{
		filepath.Join(from, "Cover.png"),
		filepath.Join(from, "Kelvin.md"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(brought.Refused) != 1 || brought.Refused[0].Name != "Cover.png" {
		t.Fatalf("what was refused: %v", brought.Refused)
	}
	if !errors.Is(brought.Refused[0].Why, port.ErrOccupied) {
		t.Errorf("the refusal: want ErrOccupied, got %v", brought.Refused[0].Why)
	}
	if body := arrived(t, v.Path, "Cover.png"); body != "MINE" {
		t.Errorf("the file that was there was replaced with %q", body)
	}
	if body := arrived(t, v.Path, "Kelvin.md"); body != "# Kelvin\n" {
		t.Errorf("the note beside it arrived as %q", body)
	}
}

// A folder the vault itself sits in would be copied into itself for as long as
// the disk lasted.
func TestAFolderHoldingTheVaultIsRefused(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, nil)
	bring := vaults.Import{Writers: filesystem.VaultWriters{}, Files: filesystem.ImportedFiles{}}

	brought, err := bring.Execute(t.Context(), v, "", []string{filepath.Dir(v.Path)})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Refused) != 1 {
		t.Fatalf("what was refused: %v", brought.Refused)
	}
	if len(brought.Landed) != 0 {
		t.Errorf("%d files landed", len(brought.Landed))
	}
}
