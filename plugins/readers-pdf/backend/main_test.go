package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/service"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/document"
)

func TestPdfServerVaultBoundary(t *testing.T) {
	vaultDir := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.pdf")
	vault := hosttest.New("vault")
	texts := hosttest.NewTexts("vault", "md.numen.readers.pdf")
	documents := service.NewDocumentServer(
		document.NewReader(
			func(id domain.VaultID) host.FileReader { return vault.With(id) },
			func(id domain.VaultID) document.LiveTexts { return texts.With(id) },
			pdf.NewWorkers(
				pdf.Options{
					Report:           func(what string, err error) { t.Errorf("%s: %v", what, err) },
					CompilationCache: nil,
					AfterFunc:        nil,
					Workers:          0,
					IdleWorkers:      0,
				},
			),
		),
	)

	_, err := documents.GetDocument(t.Context(), api.GetDocumentRequest{Path: outsidePath})
	assertDenied(t, err, "GetDocument outside the vault")

	_, err = documents.GetDocument(t.Context(), api.GetDocumentRequest{Path: filepath.Join(vaultDir, "..", "outside.pdf")})
	assertDenied(t, err, "GetDocument of a traversal path")

	_, err = documents.GetHighlights(t.Context(), api.GetHighlightsRequest{Path: outsidePath})
	assertDenied(t, err, "GetHighlights outside the vault")
}

func assertDenied(t *testing.T, err error, operation string) {
	t.Helper()
	var failure *server.APIError
	if !errors.As(err, &failure) || failure.Code != server.CodePermissionDenied {
		t.Fatalf("%s = %v, want permission_denied", operation, err)
	}
}
