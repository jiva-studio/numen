package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestVaultPathValidator(t *testing.T) {
	vaultDir := t.TempDir()
	subDir := filepath.Join(vaultDir, "notes")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	docFile := filepath.Join(subDir, "document.pdf")
	if err := os.WriteFile(docFile, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	validator := NewVaultPathValidator()

	t.Run("valid path inside vault", func(t *testing.T) {
		validPath := filepath.Join("notes", "document.pdf")
		got, err := validator.ValidatePath(vaultDir, validPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != docFile {
			t.Fatalf("got %s, want %s", got, docFile)
		}
	})

	t.Run("valid absolute path inside vault", func(t *testing.T) {
		got, err := validator.ValidatePath(vaultDir, docFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != docFile {
			t.Fatalf("got %s, want %s", got, docFile)
		}
	})

	t.Run("path traversal escape rejected", func(t *testing.T) {
		traversalPath := filepath.Join(subDir, "..", "..", "outside.pdf")
		_, err := validator.ValidatePath(vaultDir, traversalPath)
		if !errors.Is(err, port.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})

	t.Run("relative path traversal escape rejected", func(t *testing.T) {
		traversalPath := "../outside.pdf"
		_, err := validator.ValidatePath(vaultDir, traversalPath)
		if !errors.Is(err, port.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})

	t.Run("empty paths return error", func(t *testing.T) {
		if _, err := validator.ValidatePath(vaultDir, ""); err == nil {
			t.Fatal("expected error for empty target path")
		}
		if _, err := validator.ValidatePath("", "doc.pdf"); err == nil {
			t.Fatal("expected error for empty vault root")
		}
	})
}

func TestSupervisorPermissionsAndPathValidation(t *testing.T) {
	vaultDir := t.TempDir()
	docFile := filepath.Join(vaultDir, "doc.pdf")
	if err := os.WriteFile(docFile, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("permission denied without vault:read", func(t *testing.T) {
		sup := NewSupervisor()
		sup.SetPermissions([]string{"network:outbound"})

		if err := sup.CheckPermission("vault:read"); !errors.Is(err, port.ErrPermissionDenied) {
			t.Fatalf("expected ErrPermissionDenied, got %v", err)
		}

		if _, err := sup.ValidatePath(vaultDir, "doc.pdf"); !errors.Is(err, port.ErrPermissionDenied) {
			t.Fatalf("expected ErrPermissionDenied, got %v", err)
		}
	})

	t.Run("permission granted with vault:read", func(t *testing.T) {
		sup := NewSupervisor()
		sup.SetPermissions([]string{"vault:read", "vault:write"})

		if err := sup.CheckPermission("vault:read"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := sup.ValidatePath(vaultDir, "doc.pdf")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != docFile {
			t.Fatalf("got %s, want %s", got, docFile)
		}
	})

	t.Run("traversal blocked even with vault:read", func(t *testing.T) {
		sup := NewSupervisor()
		sup.SetPermissions([]string{"vault:read"})

		_, err := sup.ValidatePath(vaultDir, "../secret.txt")
		if !errors.Is(err, port.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})
}

func TestValidateExecutablePath(t *testing.T) {
	t.Run("valid binary inside regular directory", func(t *testing.T) {
		pluginDir := filepath.Join(string(filepath.Separator), "opt", "numen", "plugins", "ocr")
		got, err := validateExecutablePath(pluginDir, "bin/ocr")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(pluginDir, "bin", "ocr")
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	})

	t.Run("entrypoint escaping plugin directory rejected", func(t *testing.T) {
		pluginDir := filepath.Join(string(filepath.Separator), "opt", "numen", "plugins", "ocr")
		_, err := validateExecutablePath(pluginDir, "../../bin/escape")
		if !errors.Is(err, errEntrypointEscape) {
			t.Fatalf("expected errEntrypointEscape, got %v", err)
		}
	})

	t.Run("prohibits execution from system temp directory", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := validateExecutablePath(tempDir, "plugin.exe")
		if !errors.Is(err, errProhibitedTempDir) {
			t.Fatalf("expected errProhibitedTempDir, got %v", err)
		}
	})

	t.Run("empty entrypoint rejected", func(t *testing.T) {
		_, err := validateExecutablePath("/opt/plugin", "")
		if !errors.Is(err, errEmptyEntrypoint) {
			t.Fatalf("expected errEmptyEntrypoint, got %v", err)
		}
	})
}
