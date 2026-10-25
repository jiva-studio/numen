package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/vault"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
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

// createPictureEpub is a book with one chapter and one picture entry.
func createPictureEpub(t *testing.T) []byte {
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

func writeBook(t *testing.T) (dir, name string, size, mtime int64) {
	t.Helper()
	dir = t.TempDir()
	name = "pictured.epub"
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, createPictureEpub(t), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(full)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("NUMEN_VAULT_PATH", dir)
	return dir, name, info.Size(), info.ModTime().UnixNano()
}

func readEntry(t *testing.T, path, entry string, baseline *numenv1.Fingerprint) (*connect.Response[numenv1.ReadBookEntryResponse], error) {
	t.Helper()
	server := NewBookServer()
	t.Cleanup(server.Close)
	return server.ReadBookEntry(context.Background(),
		connect.NewRequest(&numenv1.ReadBookEntryRequest{Path: path, Entry: entry, Baseline: baseline}))
}

func TestReadBookEntryAnswersTheBytesOfTheEntry(t *testing.T) {
	_, name, size, mtime := writeBook(t)
	resp, err := readEntry(t, name, "EPUB/cover.png", &numenv1.Fingerprint{Path: name, Size: size, Mtime: mtime})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resp.Msg.GetBody(), pngBytes) {
		t.Errorf("the entry is %q, want the picture's bytes", resp.Msg.GetBody())
	}
}

func TestReadBookEntryAnswersWithoutABaseline(t *testing.T) {
	_, name, _, _ := writeBook(t)
	if _, err := readEntry(t, name, "EPUB/cover.png", nil); err != nil {
		t.Fatal(err)
	}
}

func TestReadBookEntryRefusesAFileThatChanged(t *testing.T) {
	_, name, size, mtime := writeBook(t)
	cases := map[string]*numenv1.Fingerprint{
		"another size":  {Path: name, Size: size + 1, Mtime: mtime},
		"another stamp": {Path: name, Size: size, Mtime: mtime + 1},
	}
	for label, baseline := range cases {
		t.Run(label, func(t *testing.T) {
			_, err := readEntry(t, name, "EPUB/cover.png", baseline)
			if connect.CodeOf(err) != connect.CodeAborted {
				t.Fatalf("code = %v (%v), want aborted", connect.CodeOf(err), err)
			}
		})
	}
}

func TestReadBookEntryRefusesAnEntryTheArchiveLacks(t *testing.T) {
	_, name, _, _ := writeBook(t)
	_, err := readEntry(t, name, "EPUB/missing.png", nil)
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v (%v), want not found", connect.CodeOf(err), err)
	}
}

func TestReadBookEntryRefusesAPathOutsideTheVault(t *testing.T) {
	writeBook(t)
	outside := filepath.Join(t.TempDir(), "outside.epub")
	if err := os.WriteFile(outside, createPictureEpub(t), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readEntry(t, outside, "EPUB/cover.png", nil)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("code = %v (%v), want permission denied", connect.CodeOf(err), err)
	}
	if !errors.Is(err, vault.ErrOutsideVault) {
		t.Fatalf("err = %v, want errors.Is ErrOutsideVault", err)
	}
}

func TestReadBookEntryRefusesAFileThatIsNoBook(t *testing.T) {
	dir, _, _, _ := writeBook(t)
	if err := os.WriteFile(filepath.Join(dir, "junk.epub"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readEntry(t, "junk.epub", "EPUB/cover.png", nil)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", connect.CodeOf(err), err)
	}
}

func TestReadBookEntryRefusesNoPath(t *testing.T) {
	_, err := readEntry(t, "", "EPUB/cover.png", nil)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", connect.CodeOf(err), err)
	}
}

func TestReadBookEntryRefusesNoEntry(t *testing.T) {
	_, name, _, _ := writeBook(t)
	_, err := readEntry(t, name, "", nil)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", connect.CodeOf(err), err)
	}
}

func TestReadBookEntryRefusesNotFound(t *testing.T) {
	dir, _, _, _ := writeBook(t)
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"nonexistent.epub", "sub"} {
		t.Run(target, func(t *testing.T) {
			_, err := readEntry(t, target, "EPUB/cover.png", nil)
			if connect.CodeOf(err) != connect.CodeNotFound {
				t.Fatalf("code = %v (%v), want not found", connect.CodeOf(err), err)
			}
		})
	}
}

func TestReadBookEntryAnswersWithZeroMtimeBaseline(t *testing.T) {
	_, name, size, _ := writeBook(t)
	if _, err := readEntry(t, name, "EPUB/cover.png", &numenv1.Fingerprint{Path: name, Size: size, Mtime: 0}); err != nil {
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
