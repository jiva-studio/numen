package epub_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/epub"
)

// A book of many mid-size chapters, built in memory, so that opening a book is
// measurable without the corpus.

const (
	syntheticChapterCount = 200
	syntheticLeastBytes   = 50 << 10
	syntheticMostBytes    = 300 << 10
)

// createSyntheticBook is an archive of chapters of 50 to 300 KB of prose.
func createSyntheticBook(tb testing.TB) []byte {
	tb.Helper()
	rng := rand.New(rand.NewSource(426))
	words := strings.Fields("the quick brown fox jumps over a lazy dog while seven bright lanterns " +
		"drift along the quiet river and every reader turns another page of the long book")

	var manifest, spine strings.Builder
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	write := func(name string, method uint16, body []byte) {
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := entry.Write(body); err != nil {
			tb.Fatal(err)
		}
	}

	write("mimetype", zip.Store, []byte("application/epub+zip"))
	write("META-INF/container.xml", zip.Deflate, []byte(`<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
<rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`))

	for i := range syntheticChapterCount {
		size := syntheticLeastBytes + rng.Intn(syntheticMostBytes-syntheticLeastBytes)
		var chapter strings.Builder
		chapter.WriteString("<html><body>")
		for chapter.Len() < size {
			chapter.WriteString("<p>")
			for range 40 {
				chapter.WriteString(words[rng.Intn(len(words))])
				chapter.WriteByte(' ')
			}
			chapter.WriteString("</p>\n")
		}
		chapter.WriteString("</body></html>")

		name := fmt.Sprintf("chapter%03d.xhtml", i)
		write(name, zip.Deflate, []byte(chapter.String()))
		fmt.Fprintf(&manifest, `<item id="c%d" href="%s" media-type="application/xhtml+xml"/>`, i, name)
		fmt.Fprintf(&spine, `<itemref idref="c%d"/>`, i)
	}

	write("content.opf", zip.Deflate, []byte(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
<metadata><dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">Synthetic</dc:title></metadata>
<manifest>`+manifest.String()+`</manifest><spine>`+spine.String()+`</spine></package>`))
	if err := archive.Close(); err != nil {
		tb.Fatal(err)
	}
	return out.Bytes()
}

func BenchmarkReadingASyntheticBook(b *testing.B) {
	raw := createSyntheticBook(b)
	book, err := epub.Read(raw)
	if err != nil {
		b.Fatal(err)
	}
	if len(book.Documents) != syntheticChapterCount {
		b.Fatalf("documents = %d, want %d", len(book.Documents), syntheticChapterCount)
	}
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := epub.Read(raw); err != nil {
			b.Fatal(err)
		}
	}
}
