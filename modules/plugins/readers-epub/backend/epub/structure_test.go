package epub_test

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"errors"
	"hash/crc32"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/readers-epub/backend/epub"
)

const alphaBravoOne = `<html><body><h2 id="a">Alpha</h2><p>first words</p></body></html>`

const alphaBravoTwo = `<html><body><h2 id="b">Bravo</h2><p>second words <span epub:type="pagebreak" title="mark"></span>tail</p></body></html>`

const ncxOpen = `<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">`

// ncxBook is a book of two documents, one.xhtml and two.xhtml, whose headings are
// Alpha and Bravo and whose one printed page is called "mark", with the given
// navigation control file.
func ncxBook(t *testing.T, ncx string) []byte {
	t.Helper()
	return buildSpineArchive(t, twoDocumentOpf, map[string]string{
		"OEBPS/one.xhtml": alphaBravoOne,
		"OEBPS/two.xhtml": alphaBravoTwo,
		"OEBPS/toc.ncx":   ncx,
	})
}

func pageLabels(book *epub.Book) []string {
	var out []string
	for _, page := range book.Pages {
		out = append(out, page.Label)
	}
	return out
}

func TestPartsAndPagesStandInTheOrderOfTheText(t *testing.T) {
	book := read(t, ncxBook(t, ncxOpen+`
		<navMap>
		  <navPoint><navLabel><text>Zed</text></navLabel><content src="two.xhtml"/></navPoint>
		  <navPoint><navLabel><text>Aye</text></navLabel><content src="one.xhtml"/></navPoint>
		  <navPoint><navLabel><text>Bee</text></navLabel><content src="one.xhtml"/></navPoint>
		</navMap>
		<pageList>
		  <pageTarget value="2"><content src="two.xhtml"/></pageTarget>
		  <pageTarget value="1"><content src="one.xhtml"/></pageTarget>
		  <pageTarget value="1b"><content src="one.xhtml"/></pageTarget>
		</pageList>
	</ncx>`))

	if book.Tier != epub.FromNavigation {
		t.Fatalf("structure = %q, want the control file's", book.Tier)
	}
	if got, want := titles(book), []string{"Aye", "Bee", "Zed"}; !slices.Equal(got, want) {
		t.Errorf("parts = %v, want %v: entries at one offset keep the order they are named in", got, want)
	}
	if got, want := pageLabels(book), []string{"1", "1b", "2"}; !slices.Equal(got, want) {
		t.Errorf("pages = %v, want %v: entries at one offset keep the order they are named in", got, want)
	}
}

func TestATruncatedControlFileNamesNothing(t *testing.T) {
	book := read(t, ncxBook(t, ncxOpen+`
		<navMap>
		  <navPoint><navLabel><text>Uno</text></navLabel><content src="one.xhtml"/></navPoint>
		  <navPoint><navLabel><text>Dos</text></navLabel><content src="two.xhtml"/></navPoint>
		</navMap>
		<pageList>
		  <pageTarget value="9"><content src="two.xhtml"/></pageTarget>`))

	if book.Tier != epub.FromHeadings {
		t.Errorf("structure = %q, want the headings: %v", book.Tier, titles(book))
	}
	if got, want := pageLabels(book), []string{"mark"}; !slices.Equal(got, want) {
		t.Errorf("pages = %v, want the one the markup marks", got)
	}
}

func TestAPageTargetIsNamedByItsValueOrItsLabel(t *testing.T) {
	book := read(t, ncxBook(t, ncxOpen+`
		<navMap></navMap>
		<pageList>
		  <pageTarget><navLabel><text>iv</text></navLabel><content src="one.xhtml"/></pageTarget>
		  <pageTarget value="5"><navLabel><text>five</text></navLabel></pageTarget>
		  <pageTarget value="3"><navLabel><text>three</text></navLabel><content src="two.xhtml"/></pageTarget>
		</pageList>
	</ncx>`))

	if got, want := pageLabels(book), []string{"iv", "3"}; !slices.Equal(got, want) {
		t.Errorf("pages = %v, want %v", got, want)
	}
}

const navOpf = `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Nav</dc:title></metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="a" href="one.xhtml" media-type="application/xhtml+xml"/>
    <item id="b" href="two.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="a"/><itemref idref="b"/></spine>
</package>`

func TestWhichNavigationElementIsTheTableOfContents(t *testing.T) {
	landmarks := `<nav epub:type="landmarks"><ol><li><a href="two.xhtml">Landmark</a></li></ol></nav>`
	chapters := `<ol><li><a href="one.xhtml">Chapter One</a></li><li><a href="two.xhtml">Chapter Two</a></li></ol>`
	both := []string{"Chapter One", "Chapter Two"}

	for _, c := range []struct {
		name string
		body string
		want []string
	}{{
		name: "a nav typed toc",
		body: landmarks + `<nav epub:type="toc">` + chapters + `</nav>`,
		want: both,
	}, {
		name: "a nav typed toc among other types",
		body: landmarks + `<nav epub:type="page-list toc">` + chapters + `</nav>`,
		want: both,
	}, {
		name: "a nav with the role of a table of contents",
		body: landmarks + `<nav role="doc-toc">` + chapters + `</nav>`,
		want: both,
	}, {
		name: "a nav inside other elements",
		body: landmarks + `<div><section><nav epub:type="toc">` + chapters + `</nav></section></div>`,
		want: both,
	}, {
		name: "an element that is no nav",
		body: `<div epub:type="toc"><a href="two.xhtml">Imposter</a></div><nav epub:type="toc">` + chapters + `</nav>`,
		want: both,
	}, {
		name: "a nav with no type is not the table of contents",
		body: `<nav><a href="two.xhtml">Plain</a></nav><nav epub:type="toc">` + chapters + `</nav>`,
		want: both,
	}, {
		name: "no table of contents reads every link",
		body: `<nav epub:type="landmarks"><a href="one.xhtml">Uno</a><span href="two.xhtml">Not a link</span><a href="two.xhtml">Dos</a></nav>`,
		want: []string{"Uno", "Dos"},
	}} {
		t.Run(c.name, func(t *testing.T) {
			book := read(t, buildSpineArchive(t, navOpf, map[string]string{
				"OEBPS/one.xhtml": alphaBravoOne,
				"OEBPS/two.xhtml": alphaBravoTwo,
				"OEBPS/nav.xhtml": `<html xmlns:epub="http://www.idpf.org/2007/ops"><body>` + c.body + `</body></html>`,
			}))

			if book.Tier != epub.FromNavigation {
				t.Errorf("structure = %q, want the navigation document's", book.Tier)
			}
			if got := titles(book); !slices.Equal(got, c.want) {
				t.Errorf("parts = %v, want %v", got, c.want)
			}
		})
	}
}

func TestAnEntryIsNamedOrItIsNotThere(t *testing.T) {
	book := read(t, buildSpineArchive(t, oneDocumentOpf, map[string]string{
		"OEBPS/one.xhtml": alphaBravoOne,
		"OEBPS/a.png":     "picture",
		// A directory entry for the archive's root.
		"./": "",
	}))

	if raw, err := book.Entry("OEBPS/a.png"); err != nil || string(raw) != "picture" {
		t.Errorf("entry = %q, %v", raw, err)
	}
	for _, c := range []struct{ name, want string }{
		{"", "not named"},
		{"OEBPS/gone.png", "OEBPS/gone.png"},
	} {
		raw, err := book.Entry(c.name)
		if !errors.Is(err, epub.ErrNoEntry) {
			t.Errorf("entry %q: err = %v, want %v (bytes %q)", c.name, err, epub.ErrNoEntry, raw)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("entry %q: err = %q, want it to say %q", c.name, err, c.want)
		}
	}
}

func TestAFileThatIsNoArchiveReturnsWhyItIsNone(t *testing.T) {
	_, err := epub.Read([]byte("PK, and then a lie about what follows"))
	if !errors.Is(err, epub.ErrNotArchive) || !errors.Is(err, zip.ErrFormat) {
		t.Errorf("err = %v, want it to wrap %v and %v", err, epub.ErrNotArchive, zip.ErrFormat)
	}
}

func TestAnOffsetBeforeTheBookIsNowhereInIt(t *testing.T) {
	book := read(t, tinyBook(t, nil))

	if got := book.Locate(-1); got != (epub.Location{}) {
		t.Errorf("locate(-1) = %+v, want nowhere", got)
	}
}

func TestATruncatedEncryptionFileLocksNothing(t *testing.T) {
	book := read(t, buildSpineArchive(t, laidOutOpf, map[string]string{
		"OEBPS/one.xhtml": `<html><body><p>Lambda.</p></body></html>`,
		"OEBPS/two.xhtml": `<html><body><p>Mu.</p></body></html>`,
		"META-INF/encryption.xml": `<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
			<EncryptedData xmlns="http://www.w3.org/2001/04/xmlenc#">
			  <CipherData><CipherReference URI="OEBPS/one.xhtml"/></CipherData>
			</EncryptedData>
			<EncryptedData`,
	}))

	if !strings.Contains(book.Text, "Lambda") {
		t.Errorf("the book was not read:\n%s", book.Text)
	}
}

func TestAnElementContainsNoTextOfItsOwn(t *testing.T) {
	_, markup := readOne(t, `<p>Hello <b>big</b> world<img src="x.png"/></p><p>end</p>`)

	for _, node := range getEveryNode(markup.Nodes) {
		if node.Name != "" && node.Text != "" {
			t.Errorf("the %s element says %q itself", node.Name, node.Text)
		}
	}
}

// The text of a book is bounded as a whole: a document is read only while the
// documents before it have left room.
const mostPerDocument = 16 << 20

func TestHowMuchOfABookIsRead(t *testing.T) {
	full := mostPerBook / mostPerDocument
	for _, c := range []struct {
		name  string
		sizes []int
		want  int
	}{{
		name:  "nothing is read once the bound is reached",
		sizes: append(repeat(full, mostPerDocument), 0),
		want:  full,
	}, {
		name:  "a document that fits the room that is left is read",
		sizes: append(append(repeat(full-1, mostPerDocument), mostPerDocument-1), 1),
		want:  full + 1,
	}} {
		t.Run(c.name, func(t *testing.T) {
			book := read(t, buildCommentBook(t, c.sizes))

			if len(book.Documents) != c.want {
				t.Errorf("documents = %d, want %d", len(book.Documents), c.want)
			}
		})
	}
}

func repeat(n, size int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = size
	}
	return out
}

// buildCommentBook is a book whose spine documents are each one comment of the
// given size in bytes, which costs the text nothing and the budget all of it. A
// size under the markup around the comment is that many bytes of text.
func buildCommentBook(t *testing.T, sizes []int) []byte {
	t.Helper()

	var manifest, spine strings.Builder
	for i := range sizes {
		id := "d" + strconv.Itoa(i)
		manifest.WriteString(`<item id="` + id + `" href="` + id + `.xhtml" media-type="application/xhtml+xml"/>`)
		spine.WriteString(`<itemref idref="` + id + `"/>`)
	}
	opf := `<?xml version="1.0"?>
		<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
		  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Budget</dc:title></metadata>
		  <manifest>` + manifest.String() + `</manifest>
		  <spine>` + spine.String() + `</spine>
		</package>`

	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	write := func(name, body string) {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	write("mimetype", "application/epub+zip")
	write("META-INF/container.xml", `<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
		<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
	</container>`)
	write("OEBPS/content.opf", opf)

	// Each distinct size is compressed once and written as many times as it is used.
	type packed struct {
		data []byte
		crc  uint32
	}
	cache := map[int]packed{}
	for i, size := range sizes {
		p, ok := cache[size]
		if !ok {
			body := strings.Repeat("x", size)
			if size >= len("<!---->") {
				body = "<!--" + strings.Repeat("a", size-len("<!---->")) + "-->"
			}
			var buf bytes.Buffer
			w, err := flate.NewWriter(&buf, flate.BestSpeed)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			p = packed{data: buf.Bytes(), crc: crc32.ChecksumIEEE([]byte(body))}
			cache[size] = p
		}
		entry, err := archive.CreateRaw(&zip.FileHeader{
			Name:               "OEBPS/d" + strconv.Itoa(i) + ".xhtml",
			Method:             zip.Deflate,
			CRC32:              p.crc,
			CompressedSize64:   uint64(len(p.data)),
			UncompressedSize64: uint64(size),
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(p.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestTheMarkupIsSavedIntoOneAllocation(t *testing.T) {
	const runs, width = 50, 200
	markup := &epub.Markup{Length: runs * width}
	for i := range runs {
		markup.Nodes = append(markup.Nodes, epub.Node{Offset: i, Text: strings.Repeat("a", width)})
	}
	if got := len(markup.HTML()); got <= markup.Length || got >= 2*markup.Length {
		t.Fatalf("the page is %d bytes for a text of %d: the fixture does not fit between one and two times it", got, markup.Length)
	}

	if got := testing.AllocsPerRun(10, func() { _ = markup.HTML() }); got > 1 {
		t.Errorf("the page took %v allocations, want the one that holds it", got)
	}
}

func TestALineEndAfterATableContainingOnlyAnElementBelongsToThatTable(t *testing.T) {
	_, markup := readOne(t, `<table><template><b>X</b></template></table><table><tr><td>Y</td></tr></table>`)

	if got := getNodeText(element(t, markup.Nodes, "td").Children); got != "Y\n" {
		t.Errorf("the cell says %q, want %q", got, "Y\n")
	}
}
