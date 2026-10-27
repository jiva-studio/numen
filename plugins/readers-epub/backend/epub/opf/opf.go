package opf

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/archive"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/href"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/words"
)

const (
	mediaNCX = "application/x-dtbncx+xml"
	// MediaSVG is what the manifest calls a document that is an SVG.
	MediaSVG = "image/svg+xml"
)

// ErrNoPackage: the package document is missing or cannot be read.
var ErrNoPackage = errors.New("epub: no package document")

// A prefix is the writer's choice, so a name that carries one is matched by its
// namespace. These are the namespaces a title is written in; the empty one is a
// book that declares none.
var titleNamespaces = map[string]bool{
	"":                                 true,
	"http://purl.org/dc/elements/1.1/": true,
	"http://purl.org/dc/terms/":        true,
}

// A Package is the package document, read down to what a reader needs.
type Package struct {
	Title string
	// relative to that.
	base string
	// Spine is the documents of the spine, in reading order.
	Spine []SpineItem
	// SpineStep is the CFI step of the spine element among the package element's children.
	SpineStep int
	// NCX and Nav are the archive paths of the two kinds of navigation
	// document, empty when the book has neither.
	NCX string
	Nav string

	Layout    Layout
	Direction Direction
}

// A SpineItem is one itemref of the spine and the manifest item it names.
type SpineItem struct {
	Path      string
	MediaType string
	IsLinear  bool
	Layout    Layout
	// Step is the CFI step of the itemref among the spine element's children.
	Step int
}

type rawPackageDocument struct {
	Metadata struct {
		Entries []struct {
			XMLName  xml.Name
			Property string `xml:"property,attr"`
			Refines  string `xml:"refines,attr"`
			Value    string `xml:",chardata"`
		} `xml:",any"`
	} `xml:"metadata"`
	Items []struct {
		ID         string `xml:"id,attr"`
		Href       string `xml:"href,attr"`
		MediaType  string `xml:"media-type,attr"`
		Properties string `xml:"properties,attr"`
	} `xml:"manifest>item"`
	Spine struct {
		TOC       string `xml:"toc,attr"`
		Direction string `xml:"page-progression-direction,attr"`
		ItemRefs  []struct {
			IDRef      string `xml:"idref,attr"`
			Linear     string `xml:"linear,attr"`
			Properties string `xml:"properties,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

type manifestItem struct {
	path      string
	mediaType string
}

func readPackageMetadata(read *Package, doc *rawPackageDocument) {
	for _, entry := range doc.Metadata.Entries {
		if entry.XMLName.Local == "title" && titleNamespaces[entry.XMLName.Space] &&
			read.Title == "" {
			read.Title = words.NormaliseSpaces(entry.Value)
		}
		if entry.XMLName.Local == "meta" && entry.Refines == "" &&
			entry.Property == "rendition:layout" {
			read.Layout = parseLayout(entry.Value, read.Layout)
		}
	}
}

func readManifest(read *Package, doc *rawPackageDocument) map[string]manifestItem {
	manifest := make(map[string]manifestItem, len(doc.Items))
	for _, entry := range doc.Items {
		if entry.ID == "" || entry.Href == "" {
			continue
		}
		pos := href.Resolve(read.base, entry.Href)
		manifest[entry.ID] = manifestItem{path: pos, mediaType: entry.MediaType}

		switch {
		case entry.MediaType == mediaNCX:
			read.NCX = pos
		case words.HasToken(entry.Properties, "nav"):
			read.Nav = pos
		}
	}
	if named, ok := manifest[doc.Spine.TOC]; ok && named.mediaType == mediaNCX {
		read.NCX = named.path
	}
	return manifest
}

func readSpine(read *Package, doc *rawPackageDocument, manifest map[string]manifestItem, steps []int) {
	for i, ref := range doc.Spine.ItemRefs {
		named := manifest[ref.IDRef]
		step := 2 * (i + 1)
		if i < len(steps) {
			step = steps[i]
		}
		read.Spine = append(read.Spine, SpineItem{
			Path:      named.path,
			MediaType: named.mediaType,
			IsLinear:  !strings.EqualFold(strings.TrimSpace(ref.Linear), "no"),
			Layout:    parseItemLayout(ref.Properties, read.Layout),
			Step:      step,
		})
	}
}

// readSpineSteps is the CFI step of the spine element among the package element's children, and the
// step of each itemref among the spine's children. A step counts the elements before it, whatever
// else the package document holds (EPUB CFI 1.1 §3.1.3).
func readSpineSteps(raw []byte) (int, []int) {
	d := archive.NewXMLDecoder(raw)
	var steps spineSteps
	for {
		token, err := d.Token()
		if err != nil {
			return steps.spine, steps.items
		}
		switch t := token.(type) {
		case xml.StartElement:
			steps.openElement(t.Name.Local)
		case xml.EndElement:
			steps.closeElement()
		}
	}
}

// spineSteps counts the elements of a package document as they open.
type spineSteps struct {
	depth    int
	children int
	inSpine  bool
	// spineChildren counts the children of the spine element.
	spineChildren int
	spine         int
	items         []int
}

func (s *spineSteps) openElement(name string) {
	s.depth++
	switch {
	case s.depth == 2:
		s.children++
		if name == "spine" && s.spine == 0 {
			s.spine, s.inSpine = 2*s.children, true
		}
	case s.depth == 3 && s.inSpine:
		s.spineChildren++
		if name == "itemref" {
			s.items = append(s.items, 2*s.spineChildren)
		}
	}
}

func (s *spineSteps) closeElement() {
	if s.depth == 2 {
		s.inSpine = false
	}
	s.depth--
}

// Read reads the manifest and the spine. The order is the spine's: file
// order inside the archive says nothing about the order of a book.
func Read(files map[string]*zip.File, opfPath string) (Package, error) {
	raw, ok := archive.ReadEntry(files[opfPath])
	if !ok {
		return Package{}, fmt.Errorf("%w: %s is not in the archive", ErrNoPackage, opfPath)
	}

	var document rawPackageDocument
	if err := archive.DecodeXML(raw, &document); err != nil {
		return Package{}, fmt.Errorf("%w: %w", ErrNoPackage, err)
	}

	read := Package{
		base:      path.Dir(opfPath),
		Layout:    Reflowable,
		Direction: parseDirection(document.Spine.Direction),
	}
	readPackageMetadata(&read, &document)
	manifest := readManifest(&read, &document)
	spineStep, itemSteps := readSpineSteps(raw)
	read.SpineStep = spineStep
	readSpine(&read, &document, manifest, itemSteps)

	return read, nil
}

// parseLayout returns the layout a rendition:layout value names, and inForce
// for a value that names neither.
func parseLayout(value string, inForce Layout) Layout {
	switch Layout(strings.ToLower(strings.TrimSpace(value))) {
	case PrePaginated:
		return PrePaginated
	case Reflowable:
		return Reflowable
	}
	return inForce
}

// parseItemLayout returns the layout an itemref's properties name, and the
// book's when they name none.
func parseItemLayout(properties string, book Layout) Layout {
	switch {
	case words.HasToken(properties, "rendition:layout-pre-paginated"):
		return PrePaginated
	case words.HasToken(properties, "rendition:layout-reflowable"):
		return Reflowable
	}
	return book
}

// parseDirection returns the direction a page-progression-direction value names.
func parseDirection(value string) Direction {
	named := Direction(strings.ToLower(strings.TrimSpace(value)))
	if named == LeftToRight || named == RightToLeft {
		return named
	}
	return DefaultDirection
}
