//go:build plugins_monolith

package platform

import (
	"os"
	"path/filepath"
	"testing"

	pdfservice "github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/service"
)

func TestValidateMonolithVaultPath(t *testing.T) {
	vaultDir := t.TempDir()
	t.Setenv("NUMEN_VAULT_PATH", vaultDir)

	inside := filepath.Join(vaultDir, "note.md")
	got, err := pdfservice.ValidateVaultPath(inside)
	if err != nil {
		t.Fatalf("expected valid path inside vault, got err: %v", err)
	}
	if got != inside {
		t.Fatalf("expected %s, got %s", inside, got)
	}

	got, err = pdfservice.ValidateVaultPath("relative.md")
	if err != nil {
		t.Fatalf("expected relative path to resolve, got err: %v", err)
	}
	if got != filepath.Join(vaultDir, "relative.md") {
		t.Fatalf("expected %s, got %s", filepath.Join(vaultDir, "relative.md"), got)
	}

	_, err = pdfservice.ValidateVaultPath("../outside.md")
	if err == nil {
		t.Fatal("expected error on path traversal")
	}

	tempFile := filepath.Join(os.TempDir(), "arbitrary.pdf")
	_, err = pdfservice.ValidateVaultPath(tempFile)
	if err == nil {
		t.Fatal("expected error on arbitrary temp file")
	}

	docTemp := filepath.Join(os.TempDir(), "numen-doc-test.pdf")
	got, err = pdfservice.ValidateVaultPath(docTemp)
	if err != nil {
		t.Fatalf("expected numen-doc- temp file to be accepted, got err: %v", err)
	}
	if got != docTemp {
		t.Fatalf("expected %s, got %s", docTemp, got)
	}
}

func TestMonolithFlags(t *testing.T) {
	if !IsMonolith() {
		t.Fatal("expected IsMonolith() to be true")
	}
	if BuildMode != "monolith" {
		t.Fatalf("expected BuildMode to be 'monolith', got %s", BuildMode)
	}
}
