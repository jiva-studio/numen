package main

import (
	"archive/zip"
	"bytes"
	"context"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	epubv1 "github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

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
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), 2*testingTimeout())
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

	server := service.NewBookServer(files)
	ctx := t.Context()

	// Test GetBook
	resp, err := server.GetBook(ctx, connect.NewRequest(&epubv1.GetBookRequest{
		Path: "test.epub",
	}))
	if err != nil {
		t.Fatalf("unexpected GetBook error: %v", err)
	}

	if resp.Msg.GetTitle() != "Test Book" {
		t.Fatalf("expected title 'Test Book', got %q", resp.Msg.GetTitle())
	}
	if len(resp.Msg.GetDocuments()) != 1 {
		t.Fatalf("expected 1 document, got %d", len(resp.Msg.GetDocuments()))
	}
	if resp.Msg.GetDocuments()[0].GetPath() != "OEBPS/doc1.xhtml" {
		t.Fatalf("unexpected document path: %s", resp.Msg.GetDocuments()[0].GetPath())
	}

	// Test ReadBookMarkup
	markupResp, err := server.ReadBookMarkup(ctx, connect.NewRequest(&epubv1.ReadBookMarkupRequest{
		Path:     "test.epub",
		Document: "OEBPS/doc1.xhtml",
	}))
	if err != nil {
		t.Fatalf("unexpected ReadBookMarkup error: %v", err)
	}
	if markupResp.Msg.GetMarkup() == "" {
		t.Fatal("expected non-empty markup response")
	}

	// Test non-existent book
	_, err = server.GetBook(ctx, connect.NewRequest(&epubv1.GetBookRequest{
		Path: "nonexistent.epub",
	}))
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
	server := service.NewBookServer(files)
	ctx := t.Context()

	// 1. Path outside vault should fail with permission denied
	_, err := server.GetBook(ctx, connect.NewRequest(&epubv1.GetBookRequest{
		Path: outsidePath,
	}))
	if err == nil {
		t.Fatal("expected error for path outside vault root, got nil")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied, got %v", connect.CodeOf(err))
	}

	// 2. Path traversal should fail with permission denied
	_, err = server.GetBook(ctx, connect.NewRequest(&epubv1.GetBookRequest{
		Path: "../outside.epub",
	}))
	if err == nil {
		t.Fatal("expected error for traversal path, got nil")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied, got %v", connect.CodeOf(err))
	}

	// 3. Relative path inside vault should succeed
	resp, err := server.GetBook(ctx, connect.NewRequest(&epubv1.GetBookRequest{
		Path: "inside.epub",
	}))
	if err != nil {
		t.Fatalf("unexpected error for relative path inside vault: %v", err)
	}
	if resp.Msg.GetTitle() != "Test Book" {
		t.Fatalf("expected title 'Test Book', got %q", resp.Msg.GetTitle())
	}
}

func TestAuthCheckInterceptor(t *testing.T) {
	interceptor := server.NewAuthInterceptor("secret-token")
	wrapped := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})

	ctx := t.Context()

	// Missing token -> unauthenticated
	req := connect.NewRequest(&epubv1.GetBookRequest{})
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
