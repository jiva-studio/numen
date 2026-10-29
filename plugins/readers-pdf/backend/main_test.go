package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/service"
)

func TestPdfServerVaultBoundary(t *testing.T) {
	vaultDir := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.pdf")
	documents := service.NewDocumentServer(hosttest.New("vault"), hosttest.NewTexts("vault", "md.numen.readers.pdf"))

	_, err := documents.GetDocument(t.Context(), service.GetDocumentRequest{Path: outsidePath})
	assertDenied(t, err, "GetDocument outside the vault")

	_, err = documents.GetDocument(t.Context(), service.GetDocumentRequest{Path: filepath.Join(vaultDir, "..", "outside.pdf")})
	assertDenied(t, err, "GetDocument of a traversal path")

	_, err = documents.GetHighlights(t.Context(), service.GetHighlightsRequest{Path: outsidePath})
	assertDenied(t, err, "GetHighlights outside the vault")
}

func assertDenied(t *testing.T, err error, operation string) {
	t.Helper()
	var failure *server.APIError
	if !errors.As(err, &failure) || failure.Code != server.CodePermissionDenied {
		t.Fatalf("%s = %v, want permission_denied", operation, err)
	}
}
