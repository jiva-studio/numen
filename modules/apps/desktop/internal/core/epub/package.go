package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

const (
	containerPath = "META-INF/container.xml"
	mediaPackage  = "application/oebps-package+xml"
	mediaNCX      = "application/x-dtbncx+xml"
)

// A prefix is the writer's choice, so a name that carries one is matched by its
// namespace. These are the namespaces a title is written in; the empty one is a
// book that declares none.
var titleNamespaces = map[string]bool{
	"":                                 true,
	"http://purl.org/dc/elements/1.1/": true,
	"http://purl.org/dc/terms/":        true,
}

// A packageDoc is the package document, read down to what a reader needs.
type packageDoc struct {
	title string
	// base is the directory the package document sits in. Every href in it is
	// relative to that.
	base string
	// spine is the archive path of each spine document, in reading order.
	spine []string
	// ncx and nav are the archive paths of the two kinds of navigation
	// document, empty when the book has neither.
	ncx string
	nav string
}

// archiveIndex maps each archive entry to its cleaned name.
func archiveIndex(archive *zip.Reader) map[string]*zip.File {
	files := make(map[string]*zip.File, len(archive.File))
	for _, f := range archive.File {
		if f == nil {
			continue
		}
		files[path.Clean(f.Name)] = f
	}
	return files
}

// contents reads one archive entry. A missing or unreadable entry is not there.
func contents(f *zip.File) ([]byte, bool) {
	if f == nil {
		return nil, false
	}
	r, err := f.Open()
	if err != nil {
		return nil, false
	}
	defer r.Close()
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return raw, true
}

// packagePath reads the container for the name of the package document.
func packagePath(files map[string]*zip.File) (string, error) {
	raw, ok := contents(files[containerPath])
	if !ok {
		return "", ErrNoContainer
	}
	var container struct {
		Rootfiles []struct {
			Path      string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := decodeXML(raw, &container); err != nil {
		return "", fmt.Errorf("%w: %w", ErrNoContainer, err)
	}

	first := ""
	for _, root := range container.Rootfiles {
		if root.Path == "" {
			continue
		}
		at := resolve("", root.Path)
		if root.MediaType == mediaPackage {
			return at, nil
		}
		if first == "" {
			first = at
		}
	}
	if first == "" {
		return "", fmt.Errorf("%w: it names no rootfile", ErrNoContainer)
	}
	return first, nil
}

// readPackage reads the manifest and the spine. The order is the spine's: file
// order inside the archive says nothing about the order of a book.
func readPackage(files map[string]*zip.File, opfPath string) (packageDoc, error) {
	raw, ok := contents(files[opfPath])
	if !ok {
		return packageDoc{}, fmt.Errorf("%w: %s is not in the archive", ErrNoPackage, opfPath)
	}

	var document struct {
		Metadata struct {
			Entries []struct {
				XMLName xml.Name
				Value   string `xml:",chardata"`
			} `xml:",any"`
		} `xml:"metadata"`
		Items []struct {
			ID         string `xml:"id,attr"`
			Href       string `xml:"href,attr"`
			MediaType  string `xml:"media-type,attr"`
			Properties string `xml:"properties,attr"`
		} `xml:"manifest>item"`
		Spine struct {
			TOC      string `xml:"toc,attr"`
			ItemRefs []struct {
				IDRef string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
	}
	if err := decodeXML(raw, &document); err != nil {
		return packageDoc{}, fmt.Errorf("%w: %w", ErrNoPackage, err)
	}

	read := packageDoc{base: path.Dir(opfPath)}
	for _, entry := range document.Metadata.Entries {
		if entry.XMLName.Local == "title" && titleNamespaces[entry.XMLName.Space] {
			if title := tidy(entry.Value); title != "" {
				read.title = title
				break
			}
		}
	}

	type item struct {
		at        string
		mediaType string
	}
	manifest := make(map[string]item, len(document.Items))
	for _, entry := range document.Items {
		if entry.ID == "" || entry.Href == "" {
			continue
		}
		at := resolve(read.base, entry.Href)
		manifest[entry.ID] = item{at: at, mediaType: entry.MediaType}

		switch {
		case entry.MediaType == mediaNCX:
			read.ncx = at
		case hasToken(entry.Properties, "nav"):
			read.nav = at
		}
	}
	if named, ok := manifest[document.Spine.TOC]; ok && named.mediaType == mediaNCX {
		read.ncx = named.at
	}

	for _, ref := range document.Spine.ItemRefs {
		named, ok := manifest[ref.IDRef]
		if !ok {
			// A spine may name an item the manifest does not describe.
			continue
		}
		read.spine = append(read.spine, named.at)
	}
	return read, nil
}

// decodeXML reads one of the documents that are XML: the container, the package
// and the navigation control file.
func decodeXML(raw []byte, into any) error {
	d := xml.NewDecoder(bytes.NewReader(raw))
	d.Strict = false
	// A declared encoding is taken as the bytes are. A label with no decoder
	// still leaves the names and the hrefs readable.
	d.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	return d.Decode(into)
}

// resolve turns an href into an archive path, relative to the directory the
// document holding it sits in.
func resolve(base, href string) string {
	if href == "" {
		return ""
	}
	if unescaped, err := url.PathUnescape(href); err == nil {
		href = unescaped
	}
	if strings.HasPrefix(href, "/") {
		return path.Clean(strings.TrimPrefix(href, "/"))
	}
	return path.Join(base, href)
}

// splitHref separates the document an href names from the place inside it.
func splitHref(href string) (target, fragment string) {
	target, fragment, _ = strings.Cut(href, "#")
	return target, fragment
}

// hasToken reports whether a space-separated attribute carries a value.
func hasToken(attribute, token string) bool {
	for _, field := range strings.Fields(attribute) {
		if strings.EqualFold(field, token) {
			return true
		}
	}
	return false
}
