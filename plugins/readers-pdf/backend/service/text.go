package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// textKey is the key the plugin keeps the text layer of a file under.
const textKey = "layer"

// pagesPerBatch is how many pages of a document are cut into segments at once.
const pagesPerBatch = 16

// textKind is the kind of the text versions the plugin makes.
const textKind domain.TextKind = "page-text"

// TextServer writes the text layer of the vault's PDF files into the plugin's folder as text
// versions: every word with its page and the rectangle it covers in its selector. Each layer is
// written as background work in the host's queue.
type TextServer struct {
	files host.Vault
	texts host.Texts
	store host.Store
	queue host.Queue

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	last     layer
	running  map[string]context.CancelFunc
}

var _ host.TextProvider = (*TextServer)(nil)

// NewTextServer is the text layers of the files read through files, written into store and reported
// to texts, each written in a turn of queue.
func NewTextServer(files host.Vault, texts host.Texts, store host.Store, queue host.Queue) *TextServer {
	return &TextServer{
		files: files, texts: texts, store: store, queue: queue,
		running: map[string]context.CancelFunc{},
	}
}

// Wait waits for the text layers it started to be written.
func (s *TextServer) Wait() { s.wg.Wait() }

// Close stops the text layers it started and waits for them to end.
func (s *TextServer) Close() {
	s.mu.Lock()
	s.isClosed = true
	for _, cancel := range s.running {
		cancel()
	}
	s.mu.Unlock()
	s.Wait()
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
func (s *TextServer) readLayer(ctx context.Context, files host.Vault, path string) (layer, error) {
	raw, _, err := files.Read(ctx, path)
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

// CreateText starts writing the text layer of a file as a version made from the file, and returns.
// The plugin reads files, so a request for text of a version is invalid. A file whose layer is being
// written already is not started again.
func (s *TextServer) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Source != "" {
		return fmt.Errorf("%w: the PDF reader reads files, and %s is a version", host.ErrInvalidVersion, req.Source)
	}
	key := string(req.Vault) + "\x00" + req.Path
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isClosed || s.running[key] != nil {
		return nil
	}
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.running[key] = cancel
	s.wg.Go(func() {
		defer s.finish(key)
		defer cancel()
		s.writeLayer(runCtx, req)
	})
	return nil
}

func (s *TextServer) finish(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, key)
}

// writeLayer writes the text layer of a file in a turn of the queue. A failure stays shown as a task
// until the layer is written again.
func (s *TextServer) writeLayer(ctx context.Context, req host.TextCreation) {
	task := host.Task{
		ID: "text:" + req.Path, Title: "Reading the text of " + path.Base(req.Path), About: req.Path,
		IsUserStarted: req.IsUserStarted,
	}
	err := s.putLayer(ctx, req, task)
	if err != nil && ctx.Err() == nil {
		task.Error = err.Error()
		_ = s.queue.ReportTask(context.WithoutCancel(ctx), task)
		return
	}
	_ = s.queue.FinishTask(context.WithoutCancel(ctx), task.ID)
}

// putLayer reports the text layer of a file as a version made from the file. A file that is no PDF,
// a locked one and a scan have no text layer, and get no version.
func (s *TextServer) putLayer(ctx context.Context, req host.TextCreation, task host.Task) error {
	turn, err := s.queue.EnqueueWork(ctx, host.Work{IsUserStarted: req.IsUserStarted})
	if err != nil {
		return fmt.Errorf("wait for a turn to read %s: %w", req.Path, err)
	}
	defer turn.Release()
	if err := s.queue.ReportTask(ctx, task); err != nil {
		return err
	}
	files, texts, store := s.files, s.texts, s.store
	if req.Vault != "" {
		files, texts, store = files.With(req.Vault), texts.With(req.Vault), store.With(req.Vault)
	}
	read, err := s.readLayer(ctx, files, req.Path)
	if errors.Is(err, pdf.ErrNotPDF) || errors.Is(err, pdf.ErrEncrypted) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", req.Path, err)
	}
	if !read.hasText() {
		return nil
	}
	segments, err := read.getSegments()
	if err != nil {
		return fmt.Errorf("read the words of %s: %w", req.Path, err)
	}
	file := read.hash + ".jsonl"
	if err := host.WriteText(ctx, store, file, segments); err != nil {
		return err
	}
	_, err = texts.PutVersion(ctx, host.TextPut{
		Path: req.Path, Key: textKey, Kind: textKind, SourceHash: read.hash, File: file,
	})
	return err
}

// getSegments is the text layer of every page of the file, cut pagesPerBatch pages at a time.
func (l layer) getSegments() ([]domain.TextSegment, error) {
	var segments []domain.TextSegment
	pages := len(l.book.Pages)
	for first := 0; first < pages; first += pagesPerBatch {
		last := min(first+pagesPerBatch, pages)
		wanted := make([]int, 0, last-first)
		for i := first; i < last; i++ {
			wanted = append(wanted, i)
		}
		boxes, err := l.book.Highlights(l.raw, wanted)
		if err != nil {
			return nil, err
		}
		segments = append(segments, getLayerParts(l.book, first, last, boxes)...)
	}
	return segments, nil
}

// getLayerParts is the text of pages [first, last) as parts: each word with the rectangle it covers,
// the runs between words with none, and an empty part opening each outline entry where it begins.
// The boxes are the words of those pages, ascending by offset.
func getLayerParts(book *pdf.Book, first, last int, boxes []pdf.HighlightBox) []domain.TextSegment {
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
	parts    []domain.TextSegment
}

// addPage cuts the text of one page, [from, to), at the words of the boxes on it, and returns the
// boxes of the pages after it.
func (c *layerCut) addPage(page, from, to int, boxes []pdf.HighlightBox) []pdf.HighlightBox {
	pos := from
	for len(boxes) > 0 && boxes[0].Page <= page {
		box := boxes[0]
		boxes = boxes[1:]
		start, end := max(box.Start, pos), min(box.End, to)
		if box.Page < page || start >= end {
			continue
		}
		c.add(pos, start, page, nil)
		c.add(start, end, page, [][4]float32{{box.MinX, box.MinY, box.MaxX, box.MaxY}})
		pos = end
	}
	c.add(pos, to, page, nil)
	return boxes
}

// add is the text [from, to) on a page, split where an outline entry begins inside it.
func (c *layerCut) add(from, to, page int, boxes [][4]float32) {
	for from < to {
		for len(c.sections) > 0 && c.sections[0].Offset <= from {
			c.parts = append(c.parts, domain.TextSegment{Heading: c.sections[0].Title, Selector: host.Selector{Page: page + 1}.Encode()})
			c.sections = c.sections[1:]
		}
		end := to
		if len(c.sections) > 0 && c.sections[0].Offset < to {
			end = c.sections[0].Offset
		}
		c.parts = append(c.parts, domain.TextSegment{Text: c.text[from:end], Selector: host.Selector{Page: page + 1, Boxes: boxes}.Encode()})
		from = end
	}
}
