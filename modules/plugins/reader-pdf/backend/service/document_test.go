package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestPdfServerVaultBoundary(t *testing.T) {
	vaultDir := t.TempDir()
	outsideDir := t.TempDir()

	insidePath := filepath.Join(vaultDir, "inside.pdf")
	outsidePath := filepath.Join(outsideDir, "outside.pdf")
	_ = os.WriteFile(insidePath, []byte("%PDF-1.4 dummy"), 0o644)
	_ = os.WriteFile(outsidePath, []byte("%PDF-1.4 dummy"), 0o644)

	t.Setenv("NUMEN_VAULT_PATH", vaultDir)
	server := NewDocumentServer()
	ctx := context.Background()

	// 1. GetDocument outside vault -> PermissionDenied
	_, err := server.GetDocument(ctx, connect.NewRequest(&numenv1.GetDocumentRequest{
		Path: outsidePath,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetDocument outside vault, got %v", err)
	}

	// 2. RenderPage outside vault -> PermissionDenied
	_, err = server.RenderPage(ctx, connect.NewRequest(&numenv1.RenderPageRequest{
		Path: outsidePath,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for RenderPage outside vault, got %v", err)
	}

	// 3. ExtractText outside vault -> PermissionDenied
	_, err = server.ExtractText(ctx, connect.NewRequest(&numenv1.ExtractTextRequest{
		Path: outsidePath,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for ExtractText outside vault, got %v", err)
	}

	// 4. Traversal path -> PermissionDenied
	_, err = server.GetDocument(ctx, connect.NewRequest(&numenv1.GetDocumentRequest{
		Path: filepath.Join(vaultDir, "..", "outside.pdf"),
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for traversal path, got %v", err)
	}

	// 5. GetHighlights outside vault -> PermissionDenied
	_, err = server.GetHighlights(ctx, connect.NewRequest(&numenv1.GetHighlightsRequest{
		Path: outsidePath,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetHighlights outside vault, got %v", err)
	}
}

func TestValidateVaultPath(t *testing.T) {
	vaultDir := t.TempDir()

	// Empty path
	if _, err := ValidateVaultPath(""); err == nil {
		t.Error("expected error for empty path")
	}

	// Traversal
	if _, err := ValidateVaultPath("../escape.pdf"); err == nil {
		t.Error("expected error for path traversal")
	}
	if _, err := ValidateVaultPath("a/../b/../../escape.pdf"); err == nil {
		t.Error("expected error for nested traversal")
	}

	// Temp directory path
	tempDoc := filepath.Join(os.TempDir(), "numen-doc-test123")
	if clean, err := ValidateVaultPath(tempDoc); err != nil || clean != tempDoc {
		t.Errorf("expected temp doc path %q allowed, got clean %q, err %v", tempDoc, clean, err)
	}

	// With NUMEN_VAULT_PATH set
	t.Setenv("NUMEN_VAULT_PATH", vaultDir)

	// Relative path inside vault
	relPath := "books/sample.pdf"
	wantRel := filepath.Join(vaultDir, relPath)
	if clean, err := ValidateVaultPath(relPath); err != nil || clean != wantRel {
		t.Errorf("expected %q, got %q, err %v", wantRel, clean, err)
	}

	// Clean vault root itself
	if clean, err := ValidateVaultPath(vaultDir); err != nil || clean != vaultDir {
		t.Errorf("expected vault root %q allowed, got %q, err %v", vaultDir, clean, err)
	}

	// Outside vault root
	outsideDir := t.TempDir()
	if _, err := ValidateVaultPath(filepath.Join(outsideDir, "outside.pdf")); err == nil {
		t.Error("expected error for path outside vault root")
	}

	// Unset NUMEN_VAULT_PATH
	t.Setenv("NUMEN_VAULT_PATH", "")
	if clean, err := ValidateVaultPath("relative.pdf"); err != nil || clean != "relative.pdf" {
		t.Errorf("expected 'relative.pdf', got %q, err %v", clean, err)
	}
}

func TestDocumentServerEmptyPath(t *testing.T) {
	server := NewDocumentServer()
	ctx := context.Background()

	// 1. GetDocument with empty path
	_, err := server.GetDocument(ctx, connect.NewRequest(&numenv1.GetDocumentRequest{Path: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for GetDocument with empty path, got %v", err)
	}

	// 2. RenderPage with empty path
	_, err = server.RenderPage(ctx, connect.NewRequest(&numenv1.RenderPageRequest{Path: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for RenderPage with empty path, got %v", err)
	}

	// 3. ExtractText with empty path and empty data
	_, err = server.ExtractText(ctx, connect.NewRequest(&numenv1.ExtractTextRequest{Path: "", Data: nil}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for ExtractText with empty path and data, got %v", err)
	}

	// 4. GetHighlights with empty path
	_, err = server.GetHighlights(ctx, connect.NewRequest(&numenv1.GetHighlightsRequest{Path: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for GetHighlights with empty path, got %v", err)
	}
}
