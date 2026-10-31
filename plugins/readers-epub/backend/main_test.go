package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/service"
)

func TestLifecycleServerPingAndShutdown(t *testing.T) {
	shutdownTrigger := make(chan struct{}, 1)
	life := server.NewLifecycleServer(shutdownTrigger)

	ctx := t.Context()
	status, err := life.CheckStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected ping error: %v", err)
	}
	if status != "ok" {
		t.Fatalf("expected status 'ok', got %q", status)
	}

	shutOk, err := life.RequestShutdown(ctx)
	if err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
	if !shutOk {
		t.Fatalf("expected success true, got false")
	}

	select {
	case <-shutdownTrigger:
	case <-timeAfter(t):
		t.Fatal("expected shutdownTrigger to be closed")
	}
}

func timeAfter(t *testing.T) <-chan struct{} {
	ch := make(chan struct{})
	ctx, cancel := context.WithTimeout(t.Context(), 2*testingTimeout())
	go func() {
		defer cancel()
		<-ctx.Done()
		close(ch)
	}()
	return ch
}

func testingTimeout() time.Duration {
	return 200 * time.Millisecond
}

func createMinimalEpub(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	w, err := zw.Create("META-INF/container.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(containerXML)); err != nil {
		t.Fatal(err)
	}

	contentOPF := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="pub-id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:identifier id="pub-id">urn:uuid:test</dc:identifier>
  </metadata>
  <manifest>
    <item id="doc1" href="doc1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="doc1"/>
  </spine>
</package>`
	w, err = zw.Create("OEBPS/content.opf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(contentOPF)); err != nil {
		t.Fatal(err)
	}

	docHTML := `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
  <h1>Chapter 1</h1>
  <p>Hello reflowable epub reader.</p>
</body>
</html>`
	w, err = zw.Create("OEBPS/doc1.xhtml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(docHTML)); err != nil {
		t.Fatal(err)
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestBookServerGetBookAndMarkup(t *testing.T) {
	epubBytes := createMinimalEpub(t)
	files := hosttest.New("vault")
	files.File("test.epub", string(epubBytes))

	books := service.NewBookServer(files)
	ctx := t.Context()

	// Test GetBook
	resp, err := books.GetBook(ctx, api.GetBookRequest{
		Path: "test.epub",
	})
	if err != nil {
		t.Fatalf("unexpected GetBook error: %v", err)
	}

	if resp.Title != "Test Book" {
		t.Fatalf("expected title 'Test Book', got %q", resp.Title)
	}
	if len(resp.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(resp.Documents))
	}
	if resp.Documents[0].Path != "OEBPS/doc1.xhtml" {
		t.Fatalf("unexpected document path: %s", resp.Documents[0].Path)
	}

	// Test ReadBookMarkup
	markupResp, err := books.ReadBookMarkup(ctx, api.ReadBookMarkupRequest{
		Path:     "test.epub",
		Document: "OEBPS/doc1.xhtml",
	})
	if err != nil {
		t.Fatalf("unexpected ReadBookMarkup error: %v", err)
	}
	if markupResp.Markup == "" {
		t.Fatal("expected non-empty markup response")
	}

	// Test non-existent book
	_, err = books.GetBook(ctx, api.GetBookRequest{
		Path: "nonexistent.epub",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestBookServerVaultBoundary(t *testing.T) {
	outsideDir := t.TempDir()

	epubBytes := createMinimalEpub(t)
	outsidePath := filepath.Join(outsideDir, "outside.epub")
	if err := os.WriteFile(outsidePath, epubBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	files := hosttest.New("vault")
	files.File("inside.epub", string(epubBytes))
	books := service.NewBookServer(files)
	ctx := t.Context()

	// 1. Path outside vault should fail with permission denied
	_, err := books.GetBook(ctx, api.GetBookRequest{
		Path: outsidePath,
	})
	if err == nil {
		t.Fatal("expected error for path outside vault root, got nil")
	}
	if codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied, got %v", codeOf(err))
	}

	// 2. Path traversal should fail with permission denied
	_, err = books.GetBook(ctx, api.GetBookRequest{
		Path: "../outside.epub",
	})
	if err == nil {
		t.Fatal("expected error for traversal path, got nil")
	}
	if codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied, got %v", codeOf(err))
	}

	// 3. Relative path inside vault should succeed
	resp, err := books.GetBook(ctx, api.GetBookRequest{
		Path: "inside.epub",
	})
	if err != nil {
		t.Fatalf("unexpected error for relative path inside vault: %v", err)
	}
	if resp.Title != "Test Book" {
		t.Fatalf("expected title 'Test Book', got %q", resp.Title)
	}
}

func codeOf(err error) server.ErrorCode {
	var failure *server.APIError
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}
