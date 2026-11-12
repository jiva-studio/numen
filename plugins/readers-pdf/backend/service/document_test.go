package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/document"
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

	documents := newDocumentServer(t, files)
	ctx := t.Context()

	// 1. GetDocument outside vault -> PermissionDenied
	_, err := documents.GetDocument(ctx, api.GetDocumentRequest{
		Path: outsidePath,
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected permission_denied for GetDocument outside vault, got %v", err)
	}

	// 4. Traversal path -> PermissionDenied
	_, err = documents.GetDocument(ctx, api.GetDocumentRequest{
		Path: filepath.Join(vaultDir, "..", "outside.pdf"),
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected permission_denied for traversal path, got %v", err)
	}

	// 5. GetHighlights outside vault -> PermissionDenied
	_, err = documents.GetHighlights(ctx, api.GetHighlightsRequest{
		Path: outsidePath,
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected permission_denied for GetHighlights outside vault, got %v", err)
	}

	// 6. Double-dot filename inside vault -> check passes (does not return PermissionDenied)
	_, err = documents.GetDocument(ctx, api.GetDocumentRequest{
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
	_, err = documents.GetDocument(ctx, api.GetDocumentRequest{
		Path: brokenLink,
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected permission_denied for broken outside symlink, got %v", err)
	}

}

func TestDocumentServerEmptyPath(t *testing.T) {
	vault := hosttest.New("vault")
	documents := newDocumentServer(t, vault)
	ctx := t.Context()

	// 1. GetDocument with empty path
	_, err := documents.GetDocument(ctx, api.GetDocumentRequest{Path: ""})
	if err == nil || codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument for GetDocument with empty path, got %v", err)
	}

	// 4. GetHighlights with empty path
	_, err = documents.GetHighlights(ctx, api.GetHighlightsRequest{Path: ""})
	if err == nil || codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument for GetHighlights with empty path, got %v", err)
	}
}

func codeOf(err error) server.ErrorCode {
	if failure, ok := errors.AsType[*server.APIError](err); ok {
		return failure.Code
	}
	return ""
}

func writeFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatal(err)
	}
}

func newReader(t *testing.T, vault *hosttest.Vault, texts *hosttest.Texts) *document.Reader {
	t.Helper()
	return document.NewReader(
		func(id domain.VaultID) host.FileReader { return vault.With(id) },
		func(id domain.VaultID) document.LiveTexts { return texts.With(id) },
		testsupport.NewWorkers(t))
}

func newDocumentServer(t *testing.T, vault *hosttest.Vault) *DocumentServer {
	t.Helper()
	return NewDocumentServer(newReader(t, vault, hosttest.NewTexts("vault", testsupport.PluginID)))
}
