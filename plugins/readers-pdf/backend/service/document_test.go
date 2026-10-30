package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

func TestPdfServerVaultBoundary(t *testing.T) {
	outsideDir := t.TempDir()

	files := hosttest.OnDisk(t, "vault", map[string]string{
		"inside.pdf": "%PDF-1.4 dummy",
		"a..b.pdf":   "%PDF-1.4 dummy",
	})
	vaultDir := files.Vault().Path
	outsidePath := filepath.Join(outsideDir, "outside.pdf")
	writeFile(t, outsidePath, []byte("%PDF-1.4 dummy"), 0o644)

	documents := NewDocumentServer(files, hosttest.NewTexts("vault", pluginID))
	ctx := t.Context()

	// 1. GetDocument outside vault -> PermissionDenied
	_, err := documents.GetDocument(ctx, GetDocumentRequest{
		Path: outsidePath,
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected permission_denied for GetDocument outside vault, got %v", err)
	}

	// 4. Traversal path -> PermissionDenied
	_, err = documents.GetDocument(ctx, GetDocumentRequest{
		Path: filepath.Join(vaultDir, "..", "outside.pdf"),
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected permission_denied for traversal path, got %v", err)
	}

	// 5. GetHighlights outside vault -> PermissionDenied
	_, err = documents.GetHighlights(ctx, GetHighlightsRequest{
		Path: outsidePath,
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected permission_denied for GetHighlights outside vault, got %v", err)
	}

	// 6. Double-dot filename inside vault -> check passes (does not return PermissionDenied)
	_, err = documents.GetDocument(ctx, GetDocumentRequest{
		Path: "a..b.pdf",
	})
	if codeOf(err) == server.CodePermissionDenied {
		t.Fatalf("expected double-dot filename not to be permission_denied, got %v", err)
	}

	// 7. Broken symlink pointing outside vault -> PermissionDenied
	brokenLink := filepath.Join(vaultDir, "broken.pdf")
	if err := os.Symlink(filepath.Join(outsideDir, "missing.pdf"), brokenLink); err != nil {
		t.Fatal(err)
	}
	_, err = documents.GetDocument(ctx, GetDocumentRequest{
		Path: brokenLink,
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected permission_denied for broken outside symlink, got %v", err)
	}

}

func TestDocumentServerEmptyPath(t *testing.T) {
	documents := NewDocumentServer(hosttest.New("vault"), hosttest.NewTexts("vault", pluginID))
	ctx := t.Context()

	// 1. GetDocument with empty path
	_, err := documents.GetDocument(ctx, GetDocumentRequest{Path: ""})
	if err == nil || codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument for GetDocument with empty path, got %v", err)
	}

	// 4. GetHighlights with empty path
	_, err = documents.GetHighlights(ctx, GetHighlightsRequest{Path: ""})
	if err == nil || codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument for GetHighlights with empty path, got %v", err)
	}
}

// codeOf returns the code of an API error, and the empty code for any other error.
func codeOf(err error) server.ErrorCode {
	var failure *server.APIError
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}

// writeFile writes content to path and fails the test when writing fails.
func writeFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatal(err)
	}
}
