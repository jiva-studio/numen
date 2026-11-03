package editor

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenVaultFileReadsAFileOfTheVault(t *testing.T) {
	vault := filepath.Join(t.TempDir(), "vault")
	writeAssetFile(t, filepath.Join(vault, "pictures", "kiln.png"), "glazed")

	file, info, err := openVaultFile(vault, filepath.Join("pictures", "kiln.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "glazed" || info.Name() != "kiln.png" {
		t.Fatalf("read %q named %q, want %q named kiln.png", content, info.Name(), "glazed")
	}
}

func TestOpenVaultFileRefusesWhatLiesOutsideTheVault(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	writeAssetFile(t, filepath.Join(vault, "notes", "a.md"), "inside")
	writeAssetFile(t, filepath.Join(dir, "vault-other", "secret.txt"), "outside")
	if err := os.Symlink(filepath.Join(dir, "vault-other"), filepath.Join(vault, "door")); err != nil {
		t.Skipf("no symbolic links here: %v", err)
	}

	for _, name := range []string{
		filepath.Join("..", "vault-other", "secret.txt"),
		filepath.Join("door", "secret.txt"),
		"notes",
		filepath.Join("notes", "missing.md"),
	} {
		file, _, err := openVaultFile(vault, name)
		if err == nil {
			t.Errorf("%s opened, want an error", name)
			if err := file.Close(); err != nil {
				t.Error(err)
			}
		}
	}
	if _, _, err := openVaultFile(vault, "notes"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a folder answers %v, want %v", err, os.ErrNotExist)
	}
}

func writeAssetFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
