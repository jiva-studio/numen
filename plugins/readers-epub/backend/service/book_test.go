package service

import (
	"archive/zip"
	"bytes"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

func createMinimalEpub(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	write := func(name, content string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry %s: %v", name, err)
		}
	}

	write("mimetype", "application/epub+zip")
	write("META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="EPUB/package.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	write("EPUB/package.opf", `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="pub-id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="c1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="c1"/>
  </spine>
</package>`)
	write("EPUB/chapter1.xhtml", `<?xml version="1.0" encoding="utf-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body><p>Hello reader epub</p></body>
</html>`)

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestBookServerVaultBoundary(t *testing.T) {
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.epub")
	data := createMinimalEpub(t)
	writeFile(t, outsidePath, data, 0o644)

	files := hosttest.OnDisk(t, "vault", map[string]string{
		"inside.epub": string(data),
		"a..b.epub":   string(data),
	})
	vaultDir := files.Vault().Path

	books := NewBookServer(files)
	ctx := t.Context()

	// 1. GetBook outside vault -> PermissionDenied
	_, err := books.GetBook(ctx, api.GetBookRequest{
		Path: outsidePath,
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetBook outside vault, got %v", err)
	}

	// 2. ReadBookMarkup outside vault -> PermissionDenied
	_, err = books.ReadBookMarkup(ctx, api.ReadBookMarkupRequest{
		Path:     outsidePath,
		Document: "EPUB/chapter1.xhtml",
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for ReadBookMarkup outside vault, got %v", err)
	}

	// 3. Inside vault -> Success
	resp, err := books.GetBook(ctx, api.GetBookRequest{
		Path: "inside.epub",
	})
	if err != nil {
		t.Fatalf("expected success for GetBook inside vault, got %v", err)
	}
	if resp.Title != "Test Book" {
		t.Fatalf("expected title 'Test Book', got %q", resp.Title)
	}

	// 4. GetBook with path traversal -> PermissionDenied
	_, err = books.GetBook(ctx, api.GetBookRequest{
		Path: "../outside.epub",
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetBook path traversal, got %v", err)
	}

	// 5. ReadBookMarkup with path traversal -> PermissionDenied
	_, err = books.ReadBookMarkup(ctx, api.ReadBookMarkupRequest{
		Path:     "../outside.epub",
		Document: "EPUB/chapter1.xhtml",
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for ReadBookMarkup path traversal, got %v", err)
	}

	// 6. Double dot in legitimate filename inside vault -> Success
	resp, err = books.GetBook(ctx, api.GetBookRequest{
		Path: "a..b.epub",
	})
	if err != nil {
		t.Fatalf("expected success for double-dot filename inside vault, got %v", err)
	}
	if resp.Title != "Test Book" {
		t.Fatalf("expected title 'Test Book', got %q", resp.Title)
	}

	// 7. Symlink pointing outside vault rejected even when target does not exist -> PermissionDenied
	brokenLink := filepath.Join(vaultDir, "broken.epub")
	if err := os.Symlink(filepath.Join(outsideDir, "missing.epub"), brokenLink); err != nil {
		t.Fatal(err)
	}
	_, err = books.GetBook(ctx, api.GetBookRequest{
		Path: brokenLink,
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for broken outside symlink, got %v", err)
	}

	// 8. Hidden vault -> PermissionDenied
	hiddenVault := hosttest.New("vault")
	hiddenVault.Hide()
	emptyServer := NewBookServer(hiddenVault)
	_, err = emptyServer.GetBook(ctx, api.GetBookRequest{
		Path: "inside.epub",
	})
	if err == nil || codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for hidden vault, got %v", err)
	}
}

// writeFile writes content to path and fails the test when writing fails.
func writeFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatal(err)
	}
}
