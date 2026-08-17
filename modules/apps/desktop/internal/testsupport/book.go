package testsupport

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// WriteBook writes a book at path inside root, for a test about a vault that
// holds more than notes. The path is slashed and relative to the vault.
func WriteBook(tb testing.TB, root, path string) {
	tb.Helper()
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(target, Book(tb), 0o644); err != nil {
		tb.Fatal(err)
	}
}

// Book is the bytes of an EPUB with one chapter in it. It is built here, so the
// fixture vault stays text a reader can check.
func Book(tb testing.TB) []byte {
	tb.Helper()

	parts := []struct {
		name string
		body string
		// The mimetype is stored uncompressed and written first, which is what
		// the format asks for.
		method uint16
	}{
		{"mimetype", "application/epub+zip", zip.Store},
		{"META-INF/container.xml", containerXML, zip.Deflate},
		{"OEBPS/content.opf", packageOPF, zip.Deflate},
		{"OEBPS/first.xhtml", chapterXHTML, zip.Deflate},
	}

	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	for _, part := range parts {
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: part.name, Method: part.method})
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := entry.Write([]byte(part.body)); err != nil {
			tb.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		tb.Fatal(err)
	}
	return out.Bytes()
}

const containerXML = `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
`

const packageOPF = `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>A Book In A Vault</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="id">testsupport-1</dc:identifier>
  </metadata>
  <manifest>
    <item id="first" href="first.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="first"/>
  </spine>
</package>
`

const chapterXHTML = `<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>First</title></head>
  <body>
    <h1>Of The Beginning</h1>
    <p>A book carries text and no title of its own in the graph.</p>
  </body>
</html>
`
