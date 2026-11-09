package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// textKey is the key the plugin keeps the text layer of a file under.
const textKey = "layer"

// pagesPerBatch is how many pages of a document are cut into segments at once.
// gomutants:disable-next-line INTEGER_INCREMENT reason="pagesPerBatch is a batching chunk size"
const pagesPerBatch = 16

// textKind is the TextKind of the text versions the plugin creates.
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

	workers *pdf.Workers
}

// TextRequest names the file whose text layer is written, in the vault Vault names; empty is the
// vault the host shows when the layer is asked for.
type TextRequest struct {
	Vault domain.VaultID
	Path  string
}

// NewTextServer is the text layers of the files read through files, written into store and reported
// to texts, each written in a turn of queue.
func NewTextServer(
	files host.Vault, texts host.Texts, store host.Store, queue host.Queue, workers *pdf.Workers,
) *TextServer {
	return &TextServer{
		files: files, texts: texts, store: store, queue: queue,
		wg: sync.WaitGroup{}, mu: sync.Mutex{}, isClosed: false, last: layer{hash: "", raw: nil, book: nil},
		running: map[string]context.CancelFunc{}, workers: workers,
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

func (l layer) hasText() bool { return strings.TrimSpace(l.book.Text) != "" }

func (s *TextServer) readLayer(ctx context.Context, files host.Vault, path string) (layer, error) {
	raw, _, err := files.Read(ctx, path)
	// gomutants:disable-next-line BRANCH_IF reason="vault read failure stops layer extraction"
	if err != nil {
		return layer{}, err
	}
	hash := domain.HashContent(raw)
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_NEGATION reason="last.book is non-nil whenever last.hash matches a non-empty content hash"
	if last.book != nil && last.hash == hash {
		return last, nil
	}
	book, err := s.workers.Read(raw)
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
// An empty vault is the vault the host shows when CreateText is called, and the layer is written
// there whichever vault the host shows later. A file whose layer is being written already is not
// started again.
func (s *TextServer) CreateText(ctx context.Context, req TextRequest) error {
	if req.Vault == "" {
		shown, err := s.files.GetVault(ctx)
		if err != nil {
			return fmt.Errorf("get the shown vault: %w", err)
		}
		req.Vault = shown.ID
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

func (s *TextServer) writeLayer(ctx context.Context, req TextRequest) {
	task := host.Task{
		ID: "text:" + req.Path, Title: "Reading the text of " + path.Base(req.Path), Detail: req.Path,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="task counters start at zero"
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "", IsUserStarted: false,
	}
	err := s.putLayer(ctx, req, task)
	if err != nil && ctx.Err() == nil {
		task.FailureReason = err.Error()
		if reportErr := s.queue.ReportTask(context.WithoutCancel(ctx), task); reportErr != nil {
			fmt.Fprintf(os.Stderr, "readers-pdf: show the task %s: %v\n", task.ID, reportErr)
		}
		return
	}
	if finishErr := s.queue.FinishTask(context.WithoutCancel(ctx), task.ID); finishErr != nil {
		fmt.Fprintf(os.Stderr, "readers-pdf: finish the task %s: %v\n", task.ID, finishErr)
	}
}

func (s *TextServer) putLayer(ctx context.Context, req TextRequest, task host.Task) (err error) {
	turn, err := s.queue.EnqueueWork(ctx, host.Work{IsUserStarted: false, OnWaiting: nil})
	if err != nil {
		return fmt.Errorf("wait for a turn to read %s: %w", req.Path, err)
	}
	defer func() { err = errors.Join(err, turn.Release()) }()
	if err := s.queue.ReportTask(ctx, task); err != nil {
		return err
	}
	files, texts, store := s.files.With(req.Vault), s.texts.With(req.Vault), s.store.With(req.Vault)
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
	segments, err := read.getSegments(s.workers)
	if err != nil {
		return fmt.Errorf("read the words of %s: %w", req.Path, err)
	}
	file := read.hash + ".jsonl"
	if err := host.WriteText(ctx, store, file, segments); err != nil {
		return err
	}
	_, err = texts.PutVersion(ctx, host.TextPut{
		Path: req.Path, Key: textKey, Source: "", Kind: textKind, InputHash: read.hash, File: file,
	})
	return err
}

func (l layer) getSegments(workers *pdf.Workers) ([]domain.TextSegment, error) {
	var segments []domain.TextSegment
	pages := len(l.book.Pages)
	// gomutants:disable-next-line REMOVE_SELF_ASSIGNMENTS reason="pagesPerBatch loop advances across document pages"
	for first := 0; first < pages; first += pagesPerBatch {
		last := min(first+pagesPerBatch, pages)
		// gomutants:disable-next-line ARITHMETIC_BASE,INVERT_NEGATIVES reason="last-first is a slice capacity hint"
		wanted := make([]int, 0, last-first)
		for i := first; i < last; i++ {
			wanted = append(wanted, i)
		}
		boxes, err := l.book.Highlights(workers, l.raw, wanted)
		if err != nil {
			return nil, err
		}
		parts, err := getLayerParts(l.book, first, last, boxes)
		if err != nil {
			return nil, err
		}
		segments = append(segments, parts...)
	}
	return segments, nil
}

func getLayerParts(book *pdf.Book, first, last int, boxes []pdf.HighlightBox) ([]domain.TextSegment, error) {
	end := len(book.Text)
	if last < len(book.Pages) {
		end = book.Pages[last].Offset
	}
	layers := layerCut{text: book.Text, sections: nil, parts: nil}
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
		var err error
		boxes, err = layers.addPage(page, book.Pages[page].Offset, pageEnd, boxes)
		// gomutants:disable-next-line BRANCH_IF reason="pageSelector consists of primitive int and float32 values that always marshal"
		if err != nil {
			return nil, err
		}
	}
	return layers.parts, nil
}

// layerCut is the parts of a text layer cut so far, and the outline entries still to open.
type layerCut struct {
	text     string
	sections []pdf.Part
	parts    []domain.TextSegment
}

func (c *layerCut) addPage(page, from, to int, boxes []pdf.HighlightBox) ([]pdf.HighlightBox, error) {
	pos := from
	for len(boxes) > 0 && boxes[0].Page <= page {
		box := boxes[0]
		boxes = boxes[1:]
		start, end := max(box.Start, pos), min(box.End, to)
		if box.Page < page || start >= end {
			continue
		}
		if err := c.add(pos, start, page, nil); err != nil {
			return nil, err
		}
		if err := c.add(start, end, page, [][4]float32{{box.MinX, box.MinY, box.MaxX, box.MaxY}}); err != nil {
			return nil, err
		}
		pos = end
	}
	if err := c.add(pos, to, page, nil); err != nil {
		return nil, err
	}
	return boxes, nil
}

func (c *layerCut) add(from, to, page int, boxes [][4]float32) error {
	for from < to {
		for len(c.sections) > 0 && c.sections[0].Offset <= from {
			selector, err := pageSelector{Page: page + 1, Boxes: nil}.encode()
			// gomutants:disable-next-line BRANCH_IF reason="pageSelector consists of primitive int and float32 values that always marshal"
			if err != nil {
				return err
			}
			c.parts = append(c.parts, domain.TextSegment{Text: "", Heading: c.sections[0].Title, Selector: selector})
			c.sections = c.sections[1:]
		}
		end := to
		if len(c.sections) > 0 && c.sections[0].Offset < to {
			end = c.sections[0].Offset
		}
		selector, err := pageSelector{Page: page + 1, Boxes: boxes}.encode()
		if err != nil {
			return err
		}
		c.parts = append(c.parts, domain.TextSegment{Text: c.text[from:end], Heading: "", Selector: selector})
		from = end
	}
	return nil
}
