package epub_test

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"hash/crc32"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/epub"
)

const (
	oneDoc = "OEBPS/one.xhtml"
	twoDoc = "OEBPS/two.xhtml"

	chapterOne = `<html><body><p>Alpha.</p></body></html>`
	chapterTwo = `<html><body><p>Beta.</p></body></html>`

	containerHead = `<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles>`
	containerFoot = `</rootfiles></container>`
)

// buildOpf is a package document of the metadata, manifest and spine given.
func buildOpf(metadata, manifest, spine string) string {
	return `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">` + metadata + `</metadata>
  <manifest>` + manifest + `</manifest>
  ` + spine + `
</package>`
}

const twoItems = `<item id="a" href="one.xhtml" media-type="application/xhtml+xml"/>
<item id="b" href="two.xhtml" media-type="application/xhtml+xml"/>`

const twoRefs = `<spine><itemref idref="a"/><itemref idref="b"/></spine>`

// buildNCX is a navigation control file naming the two chapters with the titles given.
func buildNCX(first, second string) string {
	return `<?xml version="1.0"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><navMap>
  <navPoint id="n1"><navLabel><text>` + first + `</text></navLabel><content src="one.xhtml"/></navPoint>
  <navPoint id="n2"><navLabel><text>` + second + `</text></navLabel><content src="two.xhtml"/></navPoint>
</navMap></ncx>`
}

func getDocumentPaths(book *epub.Book) []string {
	var out []string
	for _, doc := range book.Documents {
		out = append(out, doc.Path)
	}
	return out
}

func TestTheTitleIsTheFirstTitleOfTheMetadata(t *testing.T) {
	metadata := `<dc:creator>Someone</dc:creator>
		<x:title xmlns:x="urn:somewhere:else">Wrong</x:title>
		<dc:title>Right</dc:title>
		<dc:title>Later</dc:title>`
	book := read(t, buildSpineArchive(t, buildOpf(metadata, twoItems, twoRefs), map[string]string{
		oneDoc: chapterOne, twoDoc: chapterTwo,
	}))

	if book.Title != "Right" {
		t.Errorf("title = %q, want the first title of the Dublin Core", book.Title)
	}
}

func TestOnlyTheBooksOwnMetaSetsTheLayout(t *testing.T) {
	for name, metadata := range map[string]string{
		"a meta that refines another element": `<meta refines="#title" property="rendition:layout">pre-paginated</meta>`,
		"a meta of another property":          `<meta property="rendition:orientation">pre-paginated</meta>`,
		"an element that is not a meta":       `<dc:subject property="rendition:layout">pre-paginated</dc:subject>`,
	} {
		t.Run(name, func(t *testing.T) {
			book := read(t, buildSpineArchive(t, buildOpf(metadata, twoItems, twoRefs), map[string]string{
				oneDoc: chapterOne, twoDoc: chapterTwo,
			}))

			if book.Layout != epub.Reflowable {
				t.Errorf("layout = %q, want the book to reflow", book.Layout)
			}
		})
	}
}

func TestThePageProgressionDirection(t *testing.T) {
	for _, c := range []struct {
		value string
		want  epub.Direction
	}{
		{"ltr", epub.LeftToRight},
		{" LTR ", epub.LeftToRight},
		{"rtl", epub.RightToLeft},
		{"default", epub.DefaultDirection},
		{"", epub.DefaultDirection},
	} {
		t.Run(c.value, func(t *testing.T) {
			spine := `<spine page-progression-direction="` + c.value + `"><itemref idref="a"/></spine>`
			book := read(t, buildSpineArchive(t, buildOpf("", twoItems, spine), map[string]string{
				oneDoc: chapterOne, twoDoc: chapterTwo,
			}))

			if book.Direction != c.want {
				t.Errorf("direction = %q, want %q", book.Direction, c.want)
			}
		})
	}
}

func TestTheManifestItemsThatNameNothingAreNotInTheBook(t *testing.T) {
	manifest := `<item href="ghost.xhtml" media-type="application/xhtml+xml"/>
` + twoItems + `
		<item id="n1" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
		<item id="n2" media-type="application/x-dtbncx+xml"/>`
	spine := `<spine><itemref idref="a"/><itemref idref="b"/><itemref/></spine>`
	book := read(t, buildSpineArchive(t, buildOpf("", manifest, spine), map[string]string{
		oneDoc:              chapterOne,
		twoDoc:              chapterTwo,
		"OEBPS/ghost.xhtml": `<html><body><p>Ghost.</p></body></html>`,
		"OEBPS/toc.ncx":     buildNCX("Start", "Middle"),
	}))

	if got, want := getDocumentPaths(book), []string{oneDoc, twoDoc}; !slices.Equal(got, want) {
		t.Errorf("documents = %v, want %v", got, want)
	}
	if book.Tier != epub.FromNavigation || !slices.Equal(titles(book), []string{"Start", "Middle"}) {
		t.Errorf("tier = %q, parts = %v, want the navigation control file", book.Tier, titles(book))
	}
}

func TestWhichNavigationControlFileIsRead(t *testing.T) {
	const ncxItem = `<item id="n1" href="first.ncx" media-type="application/x-dtbncx+xml"/>`
	const secondNcxItem = `<item id="n2" href="second.ncx" media-type="application/x-dtbncx+xml"/>`
	parts := map[string]string{
		oneDoc:             chapterOne,
		twoDoc:             chapterTwo,
		"OEBPS/first.ncx":  buildNCX("First A", "First B"),
		"OEBPS/second.ncx": buildNCX("Second A", "Second B"),
	}

	for _, c := range []struct {
		name     string
		manifest string
		spine    string
		want     []string
	}{
		{"declared by its media type alone", twoItems + ncxItem, twoRefs, []string{"First A", "First B"}},
		{"the one the spine names", twoItems + ncxItem + secondNcxItem,
			`<spine toc="n1"><itemref idref="a"/><itemref idref="b"/></spine>`, []string{"First A", "First B"}},
		{"the last one when the spine names none", twoItems + ncxItem + secondNcxItem, twoRefs,
			[]string{"Second A", "Second B"}},
		{"the one by media type when the spine names a chapter", twoItems + ncxItem,
			`<spine toc="a"><itemref idref="a"/><itemref idref="b"/></spine>`, []string{"First A", "First B"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			book := read(t, buildSpineArchive(t, buildOpf("", c.manifest, c.spine), parts))

			if book.Tier != epub.FromNavigation || !slices.Equal(titles(book), c.want) {
				t.Errorf("tier = %q, parts = %v, want %v", book.Tier, titles(book), c.want)
			}
		})
	}
}

func TestAnItemrefOfAnUndeclaredItemLeavesTheRestOfTheSpine(t *testing.T) {
	spine := `<spine><itemref idref="a"/><itemref idref="nothing"/><itemref idref="b"/></spine>`
	book := read(t, buildSpineArchive(t, buildOpf("", twoItems, spine), map[string]string{
		oneDoc: chapterOne, twoDoc: chapterTwo,
	}))

	if got, want := getDocumentPaths(book), []string{oneDoc, twoDoc}; !slices.Equal(got, want) {
		t.Errorf("documents = %v, want %v", got, want)
	}
}

func TestHowAnHrefNamesAnArchiveEntry(t *testing.T) {
	manifest := `<item id="a" href="one%20a.xhtml" media-type="application/xhtml+xml"/>
		<item id="b" href="/shared/two.xhtml" media-type="application/xhtml+xml"/>`
	book := read(t, buildSpineArchive(t, buildOpf("", manifest, twoRefs), map[string]string{
		"OEBPS/one a.xhtml": chapterOne,
		"shared/two.xhtml":  chapterTwo,
		twoDoc:              `<html><body><p>Wrong.</p></body></html>`,
	}))

	if got, want := getDocumentPaths(book), []string{"OEBPS/one a.xhtml", "shared/two.xhtml"}; !slices.Equal(got, want) {
		t.Errorf("documents = %v, want %v", got, want)
	}
}

func TestADocumentWrittenLoosely(t *testing.T) {
	t.Run("an ampersand that escapes nothing", func(t *testing.T) {
		book := read(t, buildSpineArchive(t, buildOpf(`<dc:title>Tom & Jerry</dc:title>`, twoItems, twoRefs),
			map[string]string{oneDoc: chapterOne, twoDoc: chapterTwo}))

		if book.Title != "Tom & Jerry" {
			t.Errorf("title = %q", book.Title)
		}
	})

	t.Run("an encoding no decoder is registered for", func(t *testing.T) {
		opf := strings.Replace(buildOpf(`<dc:title>Latin</dc:title>`, twoItems, twoRefs),
			`<?xml version="1.0"?>`, `<?xml version="1.0" encoding="ISO-8859-1"?>`, 1)
		book := read(t, buildSpineArchive(t, opf, map[string]string{oneDoc: chapterOne, twoDoc: chapterTwo}))

		if book.Title != "Latin" {
			t.Errorf("title = %q", book.Title)
		}
	})
}

func TestWhichRootfileIsThePackage(t *testing.T) {
	opf := buildOpf(`<dc:title>Real</dc:title>`, twoItems, twoRefs)
	other := buildOpf(`<dc:title>Other</dc:title>`, twoItems, twoRefs)

	for _, c := range []struct {
		name      string
		rootfiles string
		want      string
	}{
		{"one that names no path is passed over",
			`<rootfile full-path="" media-type="application/oebps-package+xml"/>
			<rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>`, "Real"},
		{"the first of the package media type",
			`<rootfile full-path="OEBPS/other.opf" media-type="text/plain"/>
			<rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>`, "Real"},
		{"the first named when none has the package media type",
			`<rootfile full-path="OEBPS/content.opf"/>
			<rootfile full-path="OEBPS/other.opf" media-type="text/plain"/>`, "Real"},
	} {
		t.Run(c.name, func(t *testing.T) {
			book := read(t, buildArchive(t, map[string][]byte{
				"mimetype":               []byte("application/epub+zip"),
				"META-INF/container.xml": []byte(containerHead + c.rootfiles + containerFoot),
				"OEBPS/content.opf":      []byte(opf),
				"OEBPS/other.opf":        []byte(other),
				oneDoc:                   []byte(chapterOne),
				twoDoc:                   []byte(chapterTwo),
			}))

			if book.Title != c.want {
				t.Errorf("title = %q, want %q", book.Title, c.want)
			}
		})
	}
}

func TestAContainerThatNamesNoPackage(t *testing.T) {
	tests := []struct {
		name      string
		container map[string][]byte
		wantCause bool
	}{
		{"there is none", nil, false},
		{"it names no rootfile", map[string][]byte{
			"META-INF/container.xml": []byte(containerHead + containerFoot),
		}, false},
		{"it is cut off after a rootfile", map[string][]byte{
			"META-INF/container.xml": []byte(containerHead +
				`<rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>`),
			"OEBPS/content.opf": []byte(buildOpf("", twoItems, twoRefs)),
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := map[string][]byte{"mimetype": []byte("application/epub+zip")}
			maps.Copy(parts, tt.container)

			book, err := epub.Read(buildArchive(t, parts))

			if !errors.Is(err, epub.ErrNoContainer) {
				t.Fatalf("err = %v, want %v", err, epub.ErrNoContainer)
			}
			if book != nil {
				t.Error("a book was returned with an error")
			}
			var syntax *xml.SyntaxError
			if got := errors.As(err, &syntax); got != tt.wantCause {
				t.Errorf("the error holds the syntax error = %v, want %v: %v", got, tt.wantCause, err)
			}
			if tt.name == "there is none" && err.Error() != epub.ErrNoContainer.Error() {
				t.Errorf("err = %q, want the container named as missing and nothing else", err)
			}
		})
	}
}

func TestAPackageThatCannotBeRead(t *testing.T) {
	t.Run("the archive does not hold it", func(t *testing.T) {
		_, err := epub.Read(buildArchive(t, map[string][]byte{
			"mimetype": []byte("application/epub+zip"),
			"META-INF/container.xml": []byte(containerHead +
				`<rootfile full-path="OEBPS/missing.opf" media-type="application/oebps-package+xml"/>` + containerFoot),
		}))

		if !errors.Is(err, epub.ErrNoPackage) {
			t.Fatalf("err = %v, want %v", err, epub.ErrNoPackage)
		}
		if !strings.Contains(err.Error(), "OEBPS/missing.opf") {
			t.Errorf("err = %q, want it to name the missing package", err)
		}
	})

	t.Run("it is malformed", func(t *testing.T) {
		_, err := epub.Read(buildSpineArchive(t, `<package><metadata><dc:title>Cut`, nil))

		if !errors.Is(err, epub.ErrNoPackage) {
			t.Fatalf("err = %v, want %v", err, epub.ErrNoPackage)
		}
		var syntax *xml.SyntaxError
		if !errors.As(err, &syntax) {
			t.Errorf("err = %v, want it to hold the syntax error", err)
		}
	})
}

// padChapter is a chapter of exactly size bytes.
func padChapter(word string, size int) []byte {
	head, foot := "<html><body><p>"+word+"</p>", "</body></html>"
	return []byte(head + strings.Repeat(" ", size-len(head)-len(foot)) + foot)
}

func TestHowLargeADocumentMayBe(t *testing.T) {
	const manifest = `<item id="a" href="over.xhtml" media-type="application/xhtml+xml"/>
		<item id="b" href="exact.xhtml" media-type="application/xhtml+xml"/>
		<item id="c" href="one.xhtml" media-type="application/xhtml+xml"/>`
	book := read(t, buildSpineArchive(t,
		buildOpf("", manifest, `<spine><itemref idref="a"/><itemref idref="b"/><itemref idref="c"/></spine>`),
		map[string]string{
			"OEBPS/over.xhtml":  string(padChapter("Over", mostPerDocument+1)),
			"OEBPS/exact.xhtml": string(padChapter("Exact", mostPerDocument)),
			oneDoc:              chapterOne,
		}))

	if got, want := getDocumentPaths(book), []string{"OEBPS/exact.xhtml", oneDoc}; !slices.Equal(got, want) {
		t.Errorf("documents = %v, want the ones within the bound %v", got, want)
	}
	if !strings.Contains(book.Text, "Exact") || strings.Contains(book.Text, "Over") {
		t.Errorf("text = %q", book.Text)
	}

	if raw, err := book.Entry("OEBPS/exact.xhtml"); err != nil || len(raw) != mostPerDocument {
		t.Errorf("entry at the bound: %d bytes, %v", len(raw), err)
	}
	if _, err := book.Entry("OEBPS/over.xhtml"); !errors.Is(err, epub.ErrNoEntry) {
		t.Errorf("entry over the bound: err = %v, want %v", err, epub.ErrNoEntry)
	}
}

func TestAnEntryTheArchiveCannotOpenIsLeftUnread(t *testing.T) {
	const unsupportedMethod = 99
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	write := func(name string, content string, method uint16) {
		t.Helper()
		if method == zip.Deflate {
			w, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: method})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(content)); err != nil {
				t.Fatal(err)
			}
			return
		}
		w, err := archive.CreateRaw(&zip.FileHeader{
			Name: name, Method: method, CRC32: crc32.ChecksumIEEE([]byte(content)),
			CompressedSize64: uint64(len(content)), UncompressedSize64: uint64(len(content)),
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	write("mimetype", "application/epub+zip", zip.Deflate)
	write("META-INF/container.xml", containerHead+
		`<rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>`+containerFoot, zip.Deflate)
	write("OEBPS/content.opf", buildOpf("", twoItems, twoRefs), zip.Deflate)
	write(oneDoc, chapterOne, unsupportedMethod)
	write(twoDoc, chapterTwo, zip.Deflate)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	book := read(t, out.Bytes())

	if got, want := getDocumentPaths(book), []string{twoDoc}; !slices.Equal(got, want) {
		t.Errorf("documents = %v, want %v", got, want)
	}
	if _, err := book.Entry(oneDoc); !errors.Is(err, epub.ErrNoEntry) {
		t.Errorf("entry: err = %v, want %v", err, epub.ErrNoEntry)
	}
}

func TestHowTheLayoutValuesAreRead(t *testing.T) {
	for _, c := range []struct {
		name     string
		metadata string
		want     epub.Layout
	}{
		{"a later value that reflows", `<meta property="rendition:layout">pre-paginated</meta>
			<meta property="rendition:layout"> Reflowable </meta>`, epub.Reflowable},
		{"a later value that names neither", `<meta property="rendition:layout">pre-paginated</meta>
			<meta property="rendition:layout">fixed</meta>`, epub.PrePaginated},
	} {
		t.Run(c.name, func(t *testing.T) {
			book := read(t, buildSpineArchive(t, buildOpf(c.metadata, twoItems, twoRefs), map[string]string{
				oneDoc: chapterOne, twoDoc: chapterTwo,
			}))

			if book.Layout != c.want {
				t.Errorf("layout = %q, want %q", book.Layout, c.want)
			}
		})
	}

	t.Run("a document laid out once in a book that reflows", func(t *testing.T) {
		spine := `<spine><itemref idref="a" properties="rendition:layout-pre-paginated"/><itemref idref="b"/></spine>`
		book := read(t, buildSpineArchive(t, buildOpf("", twoItems, spine), map[string]string{
			oneDoc: chapterOne, twoDoc: chapterTwo,
		}))

		if got := book.Documents[0].Layout; got != epub.PrePaginated {
			t.Errorf("first document layout = %q, want %q", got, epub.PrePaginated)
		}
		if got := book.Documents[1].Layout; got != epub.Reflowable {
			t.Errorf("second document layout = %q, want %q", got, epub.Reflowable)
		}
	})
}

func TestResolveEmptyHref(t *testing.T) {
	if got := epub.Resolve("OEBPS", ""); got != "" {
		t.Errorf("resolve(\"OEBPS\", \"\") = %q, want \"\"", got)
	}
}

func TestResolveHrefWhenTargetResolvesToEmpty(t *testing.T) {
	if got := epub.ResolveHref("OEBPS", "#fragment"); got != "" {
		t.Errorf("ResolveHref(\"OEBPS\", \"#fragment\") = %q, want \"\"", got)
	}
}
