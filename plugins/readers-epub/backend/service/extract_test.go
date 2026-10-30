package service

import (
	"archive/zip"
	"bytes"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/vault"
)

var pngBytes = []byte("\x89PNG\r\n\x1a\nnot really a picture")

func createZipEpub(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name string, content []byte) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	write("mimetype", []byte("application/epub+zip"))
	write("META-INF/container.xml", []byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="EPUB/package.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`))
	for name, content := range entries {
		write(name, content)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// createImageEpub is a book with one chapter and one picture entry.
func createImageEpub(t *testing.T) []byte {
	t.Helper()
	return createZipEpub(t, map[string][]byte{
		"EPUB/package.opf": []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Pictured</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest>
    <item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="c2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`),
		"EPUB/c1.xhtml": []byte(`<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head>
<body><h1>First part</h1><p>Hello reader epub</p><span epub:type="pagebreak" title="1"></span><span epub:type="pagebreak" title="2"></span></body></html>`),
		"EPUB/c2.xhtml": []byte(`<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Two</title></head>
<body><h1>Second part</h1><p>Goodbye reader epub</p></body></html>`),
		"EPUB/cover.png": pngBytes,
	})
}

func writeBook(t *testing.T) (files *hosttest.Vault, name string, size, mtime int64) {
	t.Helper()
	name = "pictured.epub"
	files = hosttest.OnDisk(t, "vault", map[string]string{name: string(createImageEpub(t))})
	info, err := files.StatFile(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	return files, name, info.Size, info.ModTime.UnixNano()
}

func readEntry(t *testing.T, files *hosttest.Vault, path, entry string, baseline *api.FileStat) (api.ReadBookEntryResponse, error) {
	t.Helper()
	books := NewBookServer(files)
	t.Cleanup(books.Close)
	return books.ReadBookEntry(t.Context(),
		api.ReadBookEntryRequest{Path: path, Entry: entry, Baseline: baseline})
}

func TestReadBookEntryAnswersTheBytesOfTheEntry(t *testing.T) {
	files, name, size, mtime := writeBook(t)
	resp, err := readEntry(t, files, name, "EPUB/cover.png", &api.FileStat{Path: name, Size: size, Mtime: mtime})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resp.Body, pngBytes) {
		t.Errorf("the entry is %q, want the picture's bytes", resp.Body)
	}
}

func TestReadBookEntryAnswersWithoutABaseline(t *testing.T) {
	files, name, _, _ := writeBook(t)
	if _, err := readEntry(t, files, name, "EPUB/cover.png", nil); err != nil {
		t.Fatal(err)
	}
}

func TestReadBookEntry_ErrorsOnChangedFile(t *testing.T) {
	files, name, size, mtime := writeBook(t)
	cases := map[string]*api.FileStat{
		"another size":  {Path: name, Size: size + 1, Mtime: mtime},
		"another stamp": {Path: name, Size: size, Mtime: mtime + 1},
	}
	for label, baseline := range cases {
		t.Run(label, func(t *testing.T) {
			_, err := readEntry(t, files, name, "EPUB/cover.png", baseline)
			if codeOf(err) != server.CodeFailedPrecondition {
				t.Fatalf("code = %v (%v), want failed precondition", codeOf(err), err)
			}
		})
	}
}

func TestReadBookEntry_ErrorsOnMissingArchiveEntry(t *testing.T) {
	files, name, _, _ := writeBook(t)
	_, err := readEntry(t, files, name, "EPUB/missing.png", nil)
	if codeOf(err) != server.CodeNotFound {
		t.Fatalf("code = %v (%v), want not found", codeOf(err), err)
	}
}

func TestReadBookEntry_ErrorsOnPathOutsideVault(t *testing.T) {
	files, _, _, _ := writeBook(t)
	outside := filepath.Join(t.TempDir(), "outside.epub")
	if err := os.WriteFile(outside, createImageEpub(t), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readEntry(t, files, outside, "EPUB/cover.png", nil)
	if codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("code = %v (%v), want permission denied", codeOf(err), err)
	}
	if !strings.Contains(err.Error(), vault.ErrOutsideVault.Error()) {
		t.Fatalf("err = %v, want the outside-vault message", err)
	}
}

func TestReadBookEntry_ErrorsOnInvalidBook(t *testing.T) {
	files, _, _, _ := writeBook(t)
	files.File("junk.epub", "junk")
	_, err := readEntry(t, files, "junk.epub", "EPUB/cover.png", nil)
	if codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", codeOf(err), err)
	}
}

func TestReadBookEntry_ErrorsOnEmptyPath(t *testing.T) {
	files, _, _, _ := writeBook(t)
	_, err := readEntry(t, files, "", "EPUB/cover.png", nil)
	if codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", codeOf(err), err)
	}
}

func TestReadBookEntry_ErrorsOnEmptyEntry(t *testing.T) {
	files, name, _, _ := writeBook(t)
	_, err := readEntry(t, files, name, "", nil)
	if codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", codeOf(err), err)
	}
}

func TestReadBookEntry_ErrorsOnMissingBook(t *testing.T) {
	files, _, _, _ := writeBook(t)
	subDir := filepath.Join(files.Vault().Path, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"nonexistent.epub", "sub"} {
		t.Run(target, func(t *testing.T) {
			_, err := readEntry(t, files, target, "EPUB/cover.png", nil)
			if codeOf(err) != server.CodeNotFound {
				t.Fatalf("code = %v (%v), want not found", codeOf(err), err)
			}
		})
	}
}

func TestReadBookEntryAnswersWithZeroMtimeBaseline(t *testing.T) {
	files, name, size, _ := writeBook(t)
	if _, err := readEntry(t, files, name, "EPUB/cover.png", &api.FileStat{Path: name, Size: size, Mtime: 0}); err != nil {
		t.Fatalf("expected success with Mtime == 0, got %v", err)
	}
}

func createEncryptedEpub(t *testing.T) []byte {
	t.Helper()
	return createZipEpub(t, map[string][]byte{
		"META-INF/encryption.xml": []byte(`<?xml version="1.0"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <EncryptedData xmlns="http://www.w3.org/2001/04/xmlenc#">
    <CipherData><CipherReference URI="EPUB/c1.xhtml"/></CipherData>
  </EncryptedData>
</encryption>`),
		"EPUB/package.opf": []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Encrypted</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="c1"/></spine>
</package>`),
		"EPUB/c1.xhtml": []byte(`<html><body><p>Encrypted</p></body></html>`),
	})
}
