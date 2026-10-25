package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func TestPdfServerVaultBoundary(t *testing.T) {
	vaultDir := t.TempDir()
	outsideDir := t.TempDir()

	insidePath := filepath.Join(vaultDir, "inside.pdf")
	outsidePath := filepath.Join(outsideDir, "outside.pdf")
	_ = os.WriteFile(insidePath, []byte("%PDF-1.4 dummy"), 0o644)
	_ = os.WriteFile(outsidePath, []byte("%PDF-1.4 dummy"), 0o644)

	t.Setenv("NUMEN_VAULT_PATH", vaultDir)
	server := NewDocumentServer(hosttest.NewTexts("vault", pluginID))
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

	// 6. Double-dot filename inside vault -> check passes (does not return PermissionDenied)
	dotPath := filepath.Join(vaultDir, "a..b.pdf")
	_ = os.WriteFile(dotPath, []byte("%PDF-1.4 dummy"), 0o644)
	_, err = server.GetDocument(ctx, connect.NewRequest(&numenv1.GetDocumentRequest{
		Path: dotPath,
	}))
	if connect.CodeOf(err) == connect.CodePermissionDenied {
		t.Fatalf("expected double-dot filename not to be PermissionDenied, got %v", err)
	}

	// 7. Broken symlink pointing outside vault -> PermissionDenied
	brokenLink := filepath.Join(vaultDir, "broken.pdf")
	if err := os.Symlink(filepath.Join(outsideDir, "missing.pdf"), brokenLink); err != nil {
		t.Fatal(err)
	}
	_, err = server.GetDocument(ctx, connect.NewRequest(&numenv1.GetDocumentRequest{
		Path: brokenLink,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for broken outside symlink, got %v", err)
	}

	// 8. Empty vault root -> PermissionDenied
	emptyServer := NewDocumentServerIn("", hosttest.NewTexts("vault", pluginID))
	_, err = emptyServer.GetDocument(ctx, connect.NewRequest(&numenv1.GetDocumentRequest{
		Path: insidePath,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for empty vault root, got %v", err)
	}
}

func TestDocumentServerEmptyPath(t *testing.T) {
	server := NewDocumentServer(hosttest.NewTexts("vault", pluginID))
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

	// 4. GetHighlights with empty path
	_, err = server.GetHighlights(ctx, connect.NewRequest(&numenv1.GetHighlightsRequest{Path: ""}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for GetHighlights with empty path, got %v", err)
	}
}
