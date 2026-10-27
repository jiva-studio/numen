package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	pdfv1 "github.com/jiva-studio/numen/plugins/readers-pdf/gen/numen/pdf/v1"
)

func TestPdfServerVaultBoundary(t *testing.T) {
	outsideDir := t.TempDir()

	files := hosttest.OnDisk(t, "vault", map[string]string{
		"inside.pdf": "%PDF-1.4 dummy",
		"a..b.pdf":   "%PDF-1.4 dummy",
	})
	vaultDir := files.Vault().Path
	outsidePath := filepath.Join(outsideDir, "outside.pdf")
	_ = os.WriteFile(outsidePath, []byte("%PDF-1.4 dummy"), 0o644)

	server := NewDocumentServer(files, hosttest.NewTexts("vault", pluginID))
	ctx := context.Background()

	// 1. GetDocument outside vault -> PermissionDenied
	_, err := server.GetDocument(ctx, connect.NewRequest(&pdfv1.GetDocumentRequest{
		Path: outsidePath,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetDocument outside vault, got %v", err)
	}

	// 4. Traversal path -> PermissionDenied
	_, err = server.GetDocument(ctx, connect.NewRequest(&pdfv1.GetDocumentRequest{
		Path: filepath.Join(vaultDir, "..", "outside.pdf"),
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for traversal path, got %v", err)
	}

	// 5. GetHighlights outside vault -> PermissionDenied
	_, err = server.GetHighlights(ctx, connect.NewRequest(&pdfv1.GetHighlightsRequest{
		Path: outsidePath,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetHighlights outside vault, got %v", err)
	}

	// 6. Double-dot filename inside vault -> check passes (does not return PermissionDenied)
	_, err = server.GetDocument(ctx, connect.NewRequest(&pdfv1.GetDocumentRequest{
		Path: "a..b.pdf",
	}))
	if connect.CodeOf(err) == connect.CodePermissionDenied {
		t.Fatalf("expected double-dot filename not to be PermissionDenied, got %v", err)
	}

	// 7. Broken symlink pointing outside vault -> PermissionDenied
	brokenLink := filepath.Join(vaultDir, "broken.pdf")
	if err := os.Symlink(filepath.Join(outsideDir, "missing.pdf"), brokenLink); err != nil {
		t.Fatal(err)
	}
	_, err = server.GetDocument(ctx, connect.NewRequest(&pdfv1.GetDocumentRequest{
		Path: brokenLink,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for broken outside symlink, got %v", err)
	}

}

func TestDocumentServerEmptyPath(t *testing.T) {
	server := NewDocumentServer(hosttest.New("vault"), hosttest.NewTexts("vault", pluginID))
	ctx := context.Background()

	// 1. GetDocument with empty path
	_, err := server.GetDocument(ctx, connect.NewRequest(&pdfv1.GetDocumentRequest{Path: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for GetDocument with empty path, got %v", err)
	}

	// 4. GetHighlights with empty path
	_, err = server.GetHighlights(ctx, connect.NewRequest(&pdfv1.GetHighlightsRequest{Path: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for GetHighlights with empty path, got %v", err)
	}
}
