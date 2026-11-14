package books

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/testsupport"
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

func TestReaderVaultBoundary(t *testing.T) {
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.epub")
	data := createMinimalEpub(t)
	writeFile(t, outsidePath, data, 0o644)

	files := hosttest.OnDisk(t, "vault", map[string]string{
		"inside.epub": string(data),
		"a..b.epub":   string(data),
	})
	vaultDir := files.Vault().Path

	books := NewReader(func(id domain.VaultID) host.FileReader { return files.With(id) })
	ctx := t.Context()

	// 1. GetBook outside vault -> PermissionDenied
	_, err := books.GetBook(ctx, outsidePath)
	if err == nil || codeOf(err) != KindPermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetBook outside vault, got %v", err)
	}

	// 2. ReadBookMarkup outside vault -> PermissionDenied
	_, err = books.ReadBookMarkup(ctx, outsidePath, "EPUB/chapter1.xhtml")
	if err == nil || codeOf(err) != KindPermissionDenied {
		t.Fatalf("expected CodePermissionDenied for ReadBookMarkup outside vault, got %v", err)
	}

	// 3. Inside vault -> Success
	resp, err := books.GetBook(ctx, "inside.epub")
	if err != nil {
		t.Fatalf("expected success for GetBook inside vault, got %v", err)
	}
	if resp.Title != "Test Book" {
		t.Fatalf("expected title 'Test Book', got %q", resp.Title)
	}

	// 4. GetBook with path traversal -> PermissionDenied
	_, err = books.GetBook(ctx, "../outside.epub")
	if err == nil || codeOf(err) != KindPermissionDenied {
		t.Fatalf("expected CodePermissionDenied for GetBook path traversal, got %v", err)
	}

	// 5. ReadBookMarkup with path traversal -> PermissionDenied
	_, err = books.ReadBookMarkup(ctx, "../outside.epub", "EPUB/chapter1.xhtml")
	if err == nil || codeOf(err) != KindPermissionDenied {
		t.Fatalf("expected CodePermissionDenied for ReadBookMarkup path traversal, got %v", err)
	}

	// 6. Double dot in legitimate filename inside vault -> Success
	resp, err = books.GetBook(ctx, "a..b.epub")
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
	_, err = books.GetBook(ctx, brokenLink)
	if err == nil || codeOf(err) != KindPermissionDenied {
		t.Fatalf("expected CodePermissionDenied for broken outside symlink, got %v", err)
	}

	// 8. Hidden vault -> PermissionDenied
	hiddenVault := hosttest.New("vault")
	hiddenVault.Hide()
	emptyServer := NewReader(func(id domain.VaultID) host.FileReader { return hiddenVault.With(id) })
	_, err = emptyServer.GetBook(ctx, "inside.epub")
	if err == nil || codeOf(err) != KindPermissionDenied {
		t.Fatalf("expected CodePermissionDenied for hidden vault, got %v", err)
	}
}

func TestReaderPartsPagesAndErrors(t *testing.T) {
	files, name, _, _ := writeBook(t)
	files.File("bad.epub", "not an epub")
	files.File("enc.epub", string(testsupport.CreateEncryptedEpub(t)))
	books := NewReader(func(id domain.VaultID) host.FileReader { return files.With(id) })
	t.Cleanup(books.Close)
	ctx := t.Context()

	resp, err := books.GetBook(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Parts) != 2 || len(resp.Pages) != 2 {
		t.Errorf("got %d parts and %d pages, want 2 and 2", len(resp.Parts), len(resp.Pages))
	}

	if _, err := books.GetBook(ctx, "bad.epub"); codeOf(err) != KindInvalidArgument {
		t.Errorf("GetBook(bad.epub) = %v, want invalid_argument", err)
	}
	if _, err := books.ReadBookMarkup(ctx, "", ""); codeOf(err) != KindInvalidArgument {
		t.Errorf("ReadBookMarkup(\"\") = %v, want invalid_argument", err)
	}
	if _, err := books.ReadBookEntry(ctx, "enc.epub", "EPUB/c1.xhtml", nil); codeOf(err) != KindFailedPrecondition {
		t.Errorf("ReadBookEntry(enc.epub) = %v, want failed_precondition", err)
	}
}

func writeFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatal(err)
	}
}
