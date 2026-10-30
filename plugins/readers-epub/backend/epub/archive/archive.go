package archive

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"path"
)

// CreateIndex maps each archive entry to its cleaned name.
func CreateIndex(reader *zip.Reader) map[string]*zip.File {
	files := make(map[string]*zip.File, len(reader.File))
	for _, f := range reader.File {
		files[path.Clean(f.Name)] = f
	}
	return files
}

// ReadEntry reads one archive entry. A missing or unreadable entry is not there.
func ReadEntry(f *zip.File) ([]byte, bool) {
	return ReadBounded(f, MostPerDocument)
}

// MostPerDocument is how much of one document inside an archive is read, and
// MostPerBook how much of all of them together.
//
// An archive says how large its entries are and is believed about nothing: a few
// hundred kilobytes of zeros expand to as much as the format allows, and reading
// that is how opening a book ends the process. What is over the bound is left
// unread, which is what a document the archive does not hold amounts to.
const (
	MostPerDocument = 16 << 20
	MostPerBook     = 256 << 20
)

// ReadBounded is the contents of one entry, up to the bound given.
func ReadBounded(f *zip.File, most int64) (raw []byte, ok bool) {
	if f == nil {
		return nil, false
	}
	if f.UncompressedSize64 > uint64(most) {
		return nil, false
	}
	r, err := f.Open()
	if err != nil {
		return nil, false
	}
	defer func() {
		if closeErr := r.Close(); closeErr != nil {
			raw, ok = nil, false
		}
	}()
	// The archive reader refuses an entry whose bytes differ in number from the
	// size it declares.
	raw, err = io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return raw, true
}

// DecodeXML reads one of the documents that are XML: the container, the package
// and the navigation control file.
func DecodeXML(raw []byte, into any) error {
	return NewXMLDecoder(raw).Decode(into)
}

// NewXMLDecoder is the decoder DecodeXML reads a document with, for a reader that walks its tokens.
func NewXMLDecoder(raw []byte) *xml.Decoder {
	d := xml.NewDecoder(bytes.NewReader(raw))
	d.Strict = false
	// A declared encoding is taken as the bytes are. A label with no decoder
	// still leaves the names and the hrefs readable.
	d.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	return d
}
