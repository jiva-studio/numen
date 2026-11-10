// Package testsupport builds the EPUB files the tests of the backend packages read.
package testsupport

import (
	"archive/zip"
	"bytes"
	"maps"
	"slices"
	"strings"
	"testing"
)

const (
	packagePath       = "EPUB/package.opf"
	firstDocumentPath = "EPUB/c1.xhtml"
)

const pngText = "\x89PNG\r\n\x1a\nnot really a picture"

// GetPNGBytes is the bytes of the picture CreateImageEpub holds at EPUB/cover.png.
func GetPNGBytes() []byte {
	return []byte(pngText)
}

// CreateZipEpub is an archive with the EPUB mimetype, a container naming EPUB/package.opf, and entries.
func CreateZipEpub(t *testing.T, entries map[string][]byte) []byte {
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
	for _, name := range slices.Sorted(maps.Keys(entries)) {
		write(name, entries[name])
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// CreateImageEpub is a book of two documents and a cover picture, with two printed pages.
func CreateImageEpub(t *testing.T) []byte {
	t.Helper()
	return CreateZipEpub(t, map[string][]byte{
		packagePath: []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Pictured</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest>
    <item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="c2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`),
		firstDocumentPath: []byte(`<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head>
<body><h1>First part</h1><p>Hello reader epub</p><span epub:type="pagebreak" title="1"></span><span epub:type="pagebreak" title="2"></span></body></html>`),
		"EPUB/c2.xhtml": []byte(`<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Two</title></head>
<body><h1>Second part</h1><p>Goodbye reader epub</p></body></html>`),
		"EPUB/cover.png": GetPNGBytes(),
	})
}

// CreateEncryptedEpub is a book whose first document is encrypted.
func CreateEncryptedEpub(t *testing.T) []byte {
	t.Helper()
	return CreateZipEpub(t, map[string][]byte{
		"META-INF/encryption.xml": []byte(`<?xml version="1.0"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <EncryptedData xmlns="http://www.w3.org/2001/04/xmlenc#">
    <CipherData><CipherReference URI="EPUB/c1.xhtml"/></CipherData>
  </EncryptedData>
</encryption>`),
		packagePath: []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Encrypted</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="c1"/></spine>
</package>`),
		firstDocumentPath: []byte(`<html><body><p>Encrypted</p></body></html>`),
	})
}

// CreateGuidedEpub is a book of two documents with a guide, the first one long.
func CreateGuidedEpub(t *testing.T) []byte {
	t.Helper()
	return CreateZipEpub(t, map[string][]byte{
		packagePath: []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Guided</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest>
    <item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="c2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <guide><reference type="text" href="c1.xhtml"/></guide>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`),
		firstDocumentPath: []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head>
<body><h1>Prologue</h1><p>   ` + strings.Repeat("word ", 20_000) + `</p></body></html>`),
		"EPUB/c2.xhtml": []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Two</title></head>
<body><p>Epilogue of the  guided book</p></body></html>`),
	})
}

// CreateZipWithoutContainer is an archive that holds only the mimetype.
func CreateZipWithoutContainer(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("mimetype")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("application/epub+zip")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
