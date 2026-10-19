package service

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
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
	vaultDir := t.TempDir()
	outsideDir := t.TempDir()

	insidePath := filepath.Join(vaultDir, "inside.epub")
	outsidePath := filepath.Join(outsideDir, "outside.epub")
	data := createMinimalEpub(t)
	_ = os.WriteFile(insidePath, data, 0o644)
	_ = os.WriteFile(outsidePath, data, 0o644)

	t.Setenv("NUMEN_VAULT_PATH", vaultDir)
	server := NewBookServer()
	ctx := context.Background()

	// 1. GetBook outside vault -> PermissionDenied
	_, err := server.GetBook(ctx, connect.NewRequest(&numenv1.GetBookRequest{
		Path: outsidePath,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetBook outside vault, got %v", err)
	}

	// 2. ReadBookMarkup outside vault -> PermissionDenied
	_, err = server.ReadBookMarkup(ctx, connect.NewRequest(&numenv1.ReadBookMarkupRequest{
		Path:     outsidePath,
		Document: "EPUB/chapter1.xhtml",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for ReadBookMarkup outside vault, got %v", err)
	}

	// 3. Inside vault -> Success
	resp, err := server.GetBook(ctx, connect.NewRequest(&numenv1.GetBookRequest{
		Path: insidePath,
	}))
	if err != nil {
		t.Fatalf("expected success for GetBook inside vault, got %v", err)
	}
	if resp.Msg.GetTitle() != "Test Book" {
		t.Fatalf("expected title 'Test Book', got %q", resp.Msg.GetTitle())
	}

	// 4. GetBook with path traversal -> PermissionDenied
	_, err = server.GetBook(ctx, connect.NewRequest(&numenv1.GetBookRequest{
		Path: "../outside.epub",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetBook path traversal, got %v", err)
	}

	// 5. ReadBookMarkup with path traversal -> PermissionDenied
	_, err = server.ReadBookMarkup(ctx, connect.NewRequest(&numenv1.ReadBookMarkupRequest{
		Path:     "../outside.epub",
		Document: "EPUB/chapter1.xhtml",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for ReadBookMarkup path traversal, got %v", err)
	}
}
