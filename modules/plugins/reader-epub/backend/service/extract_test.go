package service

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

var pngBytes = []byte("\x89PNG\r\n\x1a\nnot really a picture")

// createPictureEpub is a book with one chapter and one picture entry.
func createPictureEpub(t *testing.T) []byte {
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
	write("EPUB/package.opf", []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Pictured</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="c1"/></spine>
</package>`))
	write("EPUB/c1.xhtml", []byte(`<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head>
<body><h1>First part</h1><p>Hello reader epub</p></body></html>`))
	write("EPUB/cover.png", pngBytes)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractBookTextAnswersTheTextAndTheParts(t *testing.T) {
	server := NewBookServer()
	resp, err := server.ExtractBookText(context.Background(),
		connect.NewRequest(&numenv1.ExtractBookTextRequest{Data: createPictureEpub(t)}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Msg.GetText(), "Hello reader epub") {
		t.Errorf("the text is %q, want the chapter's words", resp.Msg.GetText())
	}
	parts := resp.Msg.GetParts()
	if len(parts) == 0 || parts[0].GetTitle() != "First part" {
		t.Errorf("the parts are %v, want the heading the chapter names", parts)
	}
	for _, part := range parts {
		if int(part.GetOffset()) > len(resp.Msg.GetText()) {
			t.Errorf("part %q stands at %d, outside a text of %d bytes", part.GetTitle(), part.GetOffset(), len(resp.Msg.GetText()))
		}
	}
}

func TestExtractBookTextRefusesBytesThatAreNoBook(t *testing.T) {
	_, err := NewBookServer().ExtractBookText(context.Background(),
		connect.NewRequest(&numenv1.ExtractBookTextRequest{Data: []byte("this is not an archive")}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", connect.CodeOf(err), err)
	}
}

func TestExtractBookTextRefusesNoBytes(t *testing.T) {
	_, err := NewBookServer().ExtractBookText(context.Background(),
		connect.NewRequest(&numenv1.ExtractBookTextRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", connect.CodeOf(err), err)
	}
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

func readEntry(path, entry string, baseline *numenv1.Fingerprint) (*connect.Response[numenv1.ReadBookEntryResponse], error) {
	return NewBookServer().ReadBookEntry(context.Background(),
		connect.NewRequest(&numenv1.ReadBookEntryRequest{Path: path, Entry: entry, Baseline: baseline}))
}

func TestReadBookEntryAnswersTheBytesOfTheEntry(t *testing.T) {
	_, name, size, mtime := writeBook(t)
	resp, err := readEntry(name, "EPUB/cover.png", &numenv1.Fingerprint{Path: name, Size: size, Mtime: mtime})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resp.Msg.GetBody(), pngBytes) {
		t.Errorf("the entry is %q, want the picture's bytes", resp.Msg.GetBody())
	}
}

func TestReadBookEntryAnswersWithoutABaseline(t *testing.T) {
	_, name, _, _ := writeBook(t)
	if _, err := readEntry(name, "EPUB/cover.png", nil); err != nil {
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
			_, err := readEntry(name, "EPUB/cover.png", baseline)
			if connect.CodeOf(err) != connect.CodeAborted {
				t.Fatalf("code = %v (%v), want aborted", connect.CodeOf(err), err)
			}
		})
	}
}

func TestReadBookEntryRefusesAnEntryTheArchiveLacks(t *testing.T) {
	_, name, _, _ := writeBook(t)
	_, err := readEntry(name, "EPUB/missing.png", nil)
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
	_, err := readEntry(outside, "EPUB/cover.png", nil)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("code = %v (%v), want permission denied", connect.CodeOf(err), err)
	}
}

func TestReadBookEntryRefusesAFileThatIsNoBook(t *testing.T) {
	dir, _, _, _ := writeBook(t)
	if err := os.WriteFile(filepath.Join(dir, "junk.epub"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readEntry("junk.epub", "EPUB/cover.png", nil)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", connect.CodeOf(err), err)
	}
}

func TestReadBookEntryRefusesNoPath(t *testing.T) {
	_, err := readEntry("", "EPUB/cover.png", nil)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", connect.CodeOf(err), err)
	}
}
