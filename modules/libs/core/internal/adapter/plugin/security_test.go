package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
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
		if !errors.Is(err, pluginport.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})

	t.Run("relative path traversal escape rejected", func(t *testing.T) {
		traversalPath := "../outside.pdf"
		_, err := validator.ValidatePath(vaultDir, traversalPath)
		if !errors.Is(err, pluginport.ErrPathTraversal) {
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

	t.Run("legitimate filename with double dots accepted", func(t *testing.T) {
		got, err := validator.ValidatePath(vaultDir, "a..b.pdf")
		if err != nil {
			t.Fatalf("unexpected error for double-dot filename: %v", err)
		}
		want := filepath.Join(vaultDir, "a..b.pdf")
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	})

	t.Run("symlink pointing outside rejected when target does not exist", func(t *testing.T) {
		outsideDir := t.TempDir()
		brokenLink := filepath.Join(vaultDir, "broken_link.pdf")
		if err := os.Symlink(filepath.Join(outsideDir, "missing.pdf"), brokenLink); err != nil {
			t.Fatal(err)
		}
		_, err := validator.ValidatePath(vaultDir, "broken_link.pdf")
		if !errors.Is(err, pluginport.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal for broken outside symlink, got %v", err)
		}
	})

	t.Run("symlink pointing outside rejected when target exists", func(t *testing.T) {
		outsideDir := t.TempDir()
		outsideFile := filepath.Join(outsideDir, "real.pdf")
		if err := os.WriteFile(outsideFile, []byte("secret"), 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(vaultDir, "real_link.pdf")
		if err := os.Symlink(outsideFile, link); err != nil {
			t.Fatal(err)
		}
		_, err := validator.ValidatePath(vaultDir, "real_link.pdf")
		if !errors.Is(err, pluginport.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal for outside symlink, got %v", err)
		}
	})

	t.Run("missing target inside vault accepted", func(t *testing.T) {
		got, err := validator.ValidatePath(vaultDir, "nonexistent.pdf")
		if err != nil {
			t.Fatalf("unexpected error for missing target in vault: %v", err)
		}
		want := filepath.Join(vaultDir, "nonexistent.pdf")
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
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

		if err := sup.CheckPermission("vault:read"); !errors.Is(err, pluginport.ErrPermissionDenied) {
			t.Fatalf("expected ErrPermissionDenied, got %v", err)
		}

		if _, err := sup.ValidatePath(vaultDir, "doc.pdf"); !errors.Is(err, pluginport.ErrPermissionDenied) {
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
		if !errors.Is(err, pluginport.ErrPathTraversal) {
			t.Fatalf("expected ErrPathTraversal, got %v", err)
		}
	})
}

func TestValidateExecutablePath(t *testing.T) {
	t.Run("valid binary inside regular directory", func(t *testing.T) {
		pluginDir := filepath.Join(string(filepath.Separator), "opt", "numen", "plugins", "ocr")
		got, err := validateExecutablePath(pluginDir, "bin/engines.ocr")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(pluginDir, "bin", "engines.ocr")
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

	t.Run("symlink escaping plugin directory rejected", func(t *testing.T) {
		pluginDir := createPluginTestDir(t, "plugin-test-*")
		outsideDir := createPluginTestDir(t, "outside-test-*")

		outsideBin := filepath.Join(outsideDir, "target.bin")
		if err := os.WriteFile(outsideBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		symlink := filepath.Join(pluginDir, "escape.bin")
		if err := os.Symlink(outsideBin, symlink); err != nil {
			t.Fatal(err)
		}

		_, err := validateExecutablePath(pluginDir, "escape.bin")
		if !errors.Is(err, errEntrypointEscape) {
			t.Fatalf("expected errEntrypointEscape, got %v", err)
		}
	})

	t.Run("symlink pointing into system temp directory rejected", func(t *testing.T) {
		pluginDir := createPluginTestDir(t, "plugin-test-*")

		tempFile := filepath.Join(t.TempDir(), "evil.bin")
		if err := os.WriteFile(tempFile, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		symlink := filepath.Join(pluginDir, "temp-link.bin")
		if err := os.Symlink(tempFile, symlink); err != nil {
			t.Fatal(err)
		}

		_, err := validateExecutablePath(pluginDir, "temp-link.bin")
		if !errors.Is(err, errProhibitedTempDir) {
			t.Fatalf("expected errProhibitedTempDir, got %v", err)
		}
	})

	t.Run("symlink inside plugin directory accepted", func(t *testing.T) {
		pluginDir := createPluginTestDir(t, "plugin-test-*")

		realBin := filepath.Join(pluginDir, "actual.bin")
		if err := os.WriteFile(realBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		symlink := filepath.Join(pluginDir, "link.bin")
		if err := os.Symlink("actual.bin", symlink); err != nil {
			t.Fatal(err)
		}

		got, err := validateExecutablePath(pluginDir, "link.bin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		realAbs, err := filepath.EvalSymlinks(realBin)
		if err != nil {
			t.Fatal(err)
		}
		if got != realAbs {
			t.Fatalf("got %s, want %s", got, realAbs)
		}
	})
}

func createPluginTestDir(t *testing.T, pattern string) string {
	t.Helper()
	//nolint:usetesting // plugin directories inside system temporary directory are rejected by validateExecutablePath
	dir, err := os.MkdirTemp(".", pattern)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
