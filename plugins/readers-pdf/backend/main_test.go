package main

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/service"
	pdfv1 "github.com/jiva-studio/numen/plugins/readers-pdf/gen/numen/pdf/v1"
)

func TestPdfServerVaultBoundary(t *testing.T) {
	vaultDir := t.TempDir()
	outsideDir := t.TempDir()

	outsidePath := filepath.Join(outsideDir, "outside.pdf")

	server := service.NewDocumentServer(hosttest.New("vault"), hosttest.NewTexts("vault", "md.numen.readers.pdf"))
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
}

func TestPdfAuthCheckInterceptor(t *testing.T) {
	interceptor := server.NewAuthInterceptor("secret-token")
	wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})

	ctx := context.Background()

	// Missing token -> unauthenticated
	req := connect.NewRequest(&pdfv1.GetDocumentRequest{})
	_, err := wrapped(ctx, req)
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated for missing token, got %v", err)
	}

	// Wrong token -> unauthenticated
	req.Header().Set("Authorization", "Bearer wrong-token")
	_, err = wrapped(ctx, req)
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated for wrong token, got %v", err)
	}

	// Correct token -> success
	req.Header().Set("Authorization", "Bearer secret-token")
	_, err = wrapped(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error with valid token: %v", err)
	}
}
