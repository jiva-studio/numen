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
// An archive's declared entry sizes are not trusted: a few hundred kilobytes of
// zeros expand to the maximum the format allows, and reading that would
// exhaust memory and terminate the process. An entry over the bound is not
// read and is treated as absent.
const (
	MostPerDocument = 16 << 20
	MostPerBook     = 256 << 20
)

// ReadBounded is the contents of one entry, up to the bound given.
func ReadBounded(f *zip.File, most int64) ([]byte, bool) {
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
	defer r.Close()
	// The archive reader rejects an entry whose byte count differs from its
	// declared size.
	raw, err := io.ReadAll(r)
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
