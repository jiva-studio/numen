package epub_test

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// A header that disagrees with the entry is the answer the reader is built for:
// the entry beside it is read, and the one that lies is left unread.
func TestAnEntryThatMisdeclaresItsSizeIsLeftUnread(t *testing.T) {
	const centralSignature = 0x02014b50
	const uncompressedSizeOffset = 24

	chapter := "<html><body><p>" + strings.Repeat("lying ", 200) + "</p></body></html>"

	tests := []struct {
		name     string
		declared uint32
	}{
		{"declared shorter than the content", uint32(len(chapter)) - 10},
		{"declared longer than the content", uint32(len(chapter)) + 10},
		{"declared larger than any chapter", 1 << 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := buildArchive(t, map[string][]byte{
				"mimetype": []byte("application/epub+zip"),
				"META-INF/container.xml": []byte(`<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
					<rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles>
				</container>`),
				"content.opf": []byte(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
					<metadata><dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">Liar</dc:title></metadata>
					<manifest>
						<item id="a" href="lying.xhtml" media-type="application/xhtml+xml"/>
						<item id="b" href="ordinary.xhtml" media-type="application/xhtml+xml"/>
					</manifest>
					<spine><itemref idref="a"/><itemref idref="b"/></spine>
				</package>`),
				"lying.xhtml":    []byte(chapter),
				"ordinary.xhtml": []byte(`<html><body><p>a chapter of ordinary size</p></body></html>`),
			})

			// The central directory entry of the lying chapter is where the size is read from.
			at := bytes.LastIndex(raw, []byte("lying.xhtml"))
			if at < 46 || binary.LittleEndian.Uint32(raw[at-46:]) != centralSignature {
				t.Fatal("the central directory entry was not found")
			}
			binary.LittleEndian.PutUint32(raw[at-46+uncompressedSizeOffset:], tt.declared)

			book := read(t, raw)
			if !strings.Contains(book.Text, "ordinary size") {
				t.Error("the chapter beside it was not read")
			}
			if strings.Contains(book.Text, "lying") {
				t.Error("the entry that misdeclares its size was read")
			}
		})
	}
}
