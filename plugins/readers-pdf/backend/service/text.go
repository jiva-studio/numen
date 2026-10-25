package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/vault"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// textKey is the key the plugin keeps the text layer of a file under.
const textKey = "layer"

// pagesPerTextPage is how many pages of a document one page of its text holds.
const pagesPerTextPage = 16

// TextServer gives the host the text layer of the vault's PDF files as positioned text: every word
// with the rectangle it covers on its page.
type TextServer struct {
	vaultRoot func() string
	texts     host.Texts

	mu   sync.Mutex
	last layer
}

var _ host.TextProvider = (*TextServer)(nil)

// NewTextServer is the text layers of the files of the vault at vaultRoot, reported to texts.
func NewTextServer(vaultRoot func() string, texts host.Texts) *TextServer {
	return &TextServer{vaultRoot: vaultRoot, texts: texts}
}

// layer is one file's bytes and the document read from them.
type layer struct {
	hash string
	raw  []byte
	book *pdf.Book
}

// hasText reports whether the layer says anything. A scan says nothing.
func (l layer) hasText() bool { return strings.TrimSpace(l.book.Text) != "" }

// readLayer reads the file at a path of the vault. The document last read is kept while the file is
// the same bytes.
func (s *TextServer) readLayer(path string) (layer, error) {
	clean, err := vault.ValidatePathIn(s.vaultRoot(), path)
	if err != nil {
		return layer{}, err
	}
	raw, err := os.ReadFile(clean)
	if err != nil {
		return layer{}, err
	}
	hash := domain.HashContent(raw)
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	if last.book != nil && last.hash == hash {
		return last, nil
	}
	book, err := pdf.Read(raw)
	if err != nil {
		return layer{}, err
	}
	read := layer{hash: hash, raw: raw, book: book}
	s.mu.Lock()
	s.last = read
	s.mu.Unlock()
	return read, nil
}

// CreateText reports the text layer of a file as a version made from the file. A file that is no
// PDF, a locked one and a scan have no text layer, and get no version.
func (s *TextServer) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Input != "" {
		return fmt.Errorf("%w: the PDF reader reads files, and %s is a version", host.ErrInvalidVersion, req.Input)
	}
	read, err := s.readLayer(req.Path)
	if errors.Is(err, pdf.ErrNotPDF) || errors.Is(err, pdf.ErrEncrypted) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", req.Path, err)
	}
	if !read.hasText() {
		return nil
	}
	_, err = s.texts.PutVersion(ctx, host.TextPut{
		Path:      req.Path,
		Key:       textKey,
		Format:    domain.TextFormatPositioned,
		InputHash: read.hash,
		Hash:      domain.HashContent([]byte(read.book.Text)),
	})
	return err
}

// GetText is the text layer of pagesPerTextPage pages of a file. The cursor is the index of the
// first page.
func (s *TextServer) GetText(_ context.Context, req host.TextRequest) (host.TextPage, error) {
	if req.Key != textKey {
		return host.TextPage{}, fmt.Errorf("%w: the PDF reader keeps no text under %q", host.ErrNotExist, req.Key)
	}
	first := 0
	if req.Cursor != "" {
		n, err := strconv.Atoi(req.Cursor)
		if err != nil || n < 0 {
			return host.TextPage{}, fmt.Errorf("cursor %q is not a page", req.Cursor)
		}
		first = n
	}
	read, err := s.readLayer(req.Path)
	if errors.Is(err, pdf.ErrNotPDF) || errors.Is(err, pdf.ErrEncrypted) {
		return host.TextPage{}, fmt.Errorf("%w: %s has no text layer: %w", host.ErrNotExist, req.Path, err)
	}
	if err != nil {
		return host.TextPage{}, fmt.Errorf("read %s: %w", req.Path, err)
	}
	if !read.hasText() {
		return host.TextPage{}, fmt.Errorf("%w: %s has no text layer", host.ErrNotExist, req.Path)
	}
	pages := len(read.book.Pages)
	if first >= pages {
		return host.TextPage{}, fmt.Errorf("cursor %q is past the last page of %s", req.Cursor, req.Path)
	}
	last := min(first+pagesPerTextPage, pages)
	wanted := make([]int, 0, last-first)
	for i := first; i < last; i++ {
		wanted = append(wanted, i)
	}
	boxes, err := read.book.Highlights(read.raw, wanted)
	if err != nil {
		return host.TextPage{}, fmt.Errorf("read the words of %s: %w", req.Path, err)
	}
	page := host.TextPage{Parts: getLayerParts(read.book, first, last, boxes)}
	if last < pages {
		page.NextCursor = strconv.Itoa(last)
	}
	return page, nil
}

// DeleteText keeps nothing to delete: the text layer is the file's own.
func (s *TextServer) DeleteText(context.Context, domain.VaultID, string, string) error {
	return nil
}

// getLayerParts is the text of pages [first, last) as parts: each word with the rectangle it covers,
// the runs between words with none, and an empty part opening each outline entry where it begins.
// The boxes are the words of those pages, ascending by offset.
func getLayerParts(book *pdf.Book, first, last int, boxes []pdf.HighlightBox) []domain.TextPart {
	end := len(book.Text)
	if last < len(book.Pages) {
		end = book.Pages[last].Offset
	}
	layers := layerCut{text: book.Text}
	for _, part := range book.Parts {
		if part.Offset >= book.Pages[first].Offset && part.Offset < end {
			layers.sections = append(layers.sections, part)
		}
	}
	for page := first; page < last; page++ {
		pageEnd := end
		if page+1 < len(book.Pages) {
			pageEnd = book.Pages[page+1].Offset
		}
		boxes = layers.addPage(page, book.Pages[page].Offset, pageEnd, boxes)
	}
	return layers.parts
}

// layerCut is the parts of a text layer cut so far, and the outline entries still to open.
type layerCut struct {
	text     string
	sections []pdf.Part
	parts    []domain.TextPart
}

// addPage cuts the text of one page, [from, to), at the words of the boxes on it, and returns the
// boxes of the pages after it.
func (c *layerCut) addPage(page, from, to int, boxes []pdf.HighlightBox) []pdf.HighlightBox {
	pos := from
	for len(boxes) > 0 && boxes[0].Page <= page {
		box := boxes[0]
		boxes = boxes[1:]
		start, end := max(box.From, pos), min(box.To, to)
		if box.Page < page || start >= end {
			continue
		}
		c.add(pos, start, page, nil)
		c.add(start, end, page, []domain.Rect{{MinX: box.MinX, MinY: box.MinY, MaxX: box.MaxX, MaxY: box.MaxY}})
		pos = end
	}
	c.add(pos, to, page, nil)
	return boxes
}

// add is the text [from, to) on a page, split where an outline entry begins inside it.
func (c *layerCut) add(from, to, page int, rects []domain.Rect) {
	for from < to {
		for len(c.sections) > 0 && c.sections[0].Offset <= from {
			c.parts = append(c.parts, domain.TextPart{Section: c.sections[0].Title, Place: domain.TextPlace{Page: page}})
			c.sections = c.sections[1:]
		}
		end := to
		if len(c.sections) > 0 && c.sections[0].Offset < to {
			end = c.sections[0].Offset
		}
		c.parts = append(c.parts, domain.TextPart{Text: c.text[from:end], Place: domain.TextPlace{Page: page, Rects: rects}})
		from = end
	}
}
