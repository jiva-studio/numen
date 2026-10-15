package main

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestLifecycleServerPingAndShutdown(t *testing.T) {
	shutdownTrigger := make(chan struct{}, 1)
	life := &lifecycleServer{shutdownTrigger: shutdownTrigger}

	ctx := context.Background()
	pingResp, err := life.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("unexpected ping error: %v", err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Fatalf("expected status 'ok', got %q", pingResp.Msg.GetStatus())
	}

	shutResp, err := life.Shutdown(ctx, connect.NewRequest(&numenv1.ShutdownRequest{}))
	if err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
	if !shutResp.Msg.GetSuccess() {
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
		ctx, cancel := context.WithTimeout(context.Background(), 2*testingTimeout())
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
	_, _ = w.Write([]byte(containerXML))

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
	_, _ = w.Write([]byte(contentOPF))

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
	_, _ = w.Write([]byte(docHTML))

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestBookServerGetBookAndMarkup(t *testing.T) {
	tempDir := t.TempDir()
	epubBytes := createMinimalEpub(t)
	epubPath := filepath.Join(tempDir, "test.epub")
	if err := os.WriteFile(epubPath, epubBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	server := &bookServer{}
	ctx := context.Background()

	// Test GetBook
	resp, err := server.GetBook(ctx, connect.NewRequest(&numenv1.GetBookRequest{
		Path: epubPath,
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
	markupResp, err := server.ReadBookMarkup(ctx, connect.NewRequest(&numenv1.ReadBookMarkupRequest{
		Path:     epubPath,
		Document: "OEBPS/doc1.xhtml",
	}))
	if err != nil {
		t.Fatalf("unexpected ReadBookMarkup error: %v", err)
	}
	if markupResp.Msg.GetMarkup() == "" {
		t.Fatal("expected non-empty markup response")
	}

	// Test non-existent book
	_, err = server.GetBook(ctx, connect.NewRequest(&numenv1.GetBookRequest{
		Path: filepath.Join(tempDir, "nonexistent.epub"),
	}))
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}
