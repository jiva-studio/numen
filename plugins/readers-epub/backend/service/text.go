package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

// textKey is the key the plugin keeps the text of a book under.
const textKey = "text"

// partBytes is the most text one segment holds.
const partBytes = 64 << 10

// textKind is the kind of the text versions the plugin makes.
const textKind domain.TextKind = "book-text"

// TextServer writes the text of the vault's EPUB books into the plugin's folder as text versions,
// each named section opened where it begins and each segment selected by the CFI of the block it is
// in. The text and its offsets are the ones the reader sets the book with. Each text is written as
// background work in the host's queue.
type TextServer struct {
	files host.Vault
	texts host.Texts
	store host.Store
	queue host.Queue

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	last    bookText
	running map[string]bool
}

var _ host.TextProvider = (*TextServer)(nil)

// NewTextServer is the texts of the books read through files, written into store and reported to
// texts, each written in a turn of queue.
func NewTextServer(files host.Vault, texts host.Texts, store host.Store, queue host.Queue) *TextServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &TextServer{
		files: files, texts: texts, store: store, queue: queue,
		ctx: ctx, cancel: cancel, running: map[string]bool{},
	}
}

// Wait waits for the texts it started to be written.
func (s *TextServer) Wait() { s.wg.Wait() }

// Close stops the texts it started and waits for them to end.
func (s *TextServer) Close() {
	s.cancel()
	s.Wait()
}

// bookText is one file's text as segments, and the hash of the bytes it was read from.
type bookText struct {
	hash     string
	text     string
	segments []domain.TextSegment
}

// isUnreadable reports whether a failure to read a book is a file that holds no text to give: no
// archive, no book in it, or a book whose text is locked.
func isUnreadable(err error) bool {
	return errors.Is(err, epub.ErrNotArchive) || errors.Is(err, epub.ErrNoContainer) ||
		errors.Is(err, epub.ErrNoPackage) || errors.Is(err, epub.ErrEncrypted)
}

// readBook reads the book at a path of the vault. The book last read is kept while the file is the
// same bytes.
func (s *TextServer) readBook(ctx context.Context, files host.Vault, path string) (bookText, error) {
	raw, _, err := files.Read(ctx, path)
	if err != nil {
		return bookText{}, err
	}
	hash := domain.HashContent(raw)
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	if last.hash == hash {
		return last, nil
	}
	book, err := epub.Read(raw)
	if err != nil {
		return bookText{}, err
	}
	read := bookText{hash: hash, text: book.Text, segments: getSegments(book.Text, book.Parts, book.Anchors())}
	s.mu.Lock()
	s.last = read
	s.mu.Unlock()
	return read, nil
}

// CreateText starts writing the text of a book as a version made from the file, and returns. The
// plugin reads files, so a request for text of a version is invalid. A book whose text is being
// written already is not started again.
func (s *TextServer) CreateText(_ context.Context, req host.TextCreation) error {
	if req.Source != "" {
		return fmt.Errorf("%w: the EPUB reader reads files, and %s is a version", host.ErrInvalidVersion, req.Source)
	}
	key := string(req.Vault) + "\x00" + req.Path
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[key] {
		return nil
	}
	s.running[key] = true
	s.wg.Go(func() {
		defer s.finish(key)
		s.writeText(s.ctx, req)
	})
	return nil
}

func (s *TextServer) finish(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, key)
}

// writeText writes the text of a book in a turn of the queue. A failure stays shown as a task until
// the text is written again.
func (s *TextServer) writeText(ctx context.Context, req host.TextCreation) {
	task := host.Task{
		ID: "text:" + req.Path, Title: "Reading the text of " + path.Base(req.Path), About: req.Path,
		IsUserStarted: req.IsUserStarted,
	}
	err := s.putText(ctx, req, task)
	if err != nil && ctx.Err() == nil {
		task.Error = err.Error()
		_ = s.queue.ReportTask(context.WithoutCancel(ctx), task)
		return
	}
	_ = s.queue.FinishTask(context.WithoutCancel(ctx), task.ID)
}

// putText reports the text of a book as a version made from the file. A file that is no book, a
// locked book and a book with no text get no version.
func (s *TextServer) putText(ctx context.Context, req host.TextCreation, task host.Task) error {
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
	read, err := s.readBook(ctx, files, req.Path)
	if isUnreadable(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", req.Path, err)
	}
	if strings.TrimSpace(read.text) == "" {
		return nil
	}
	file := read.hash + ".jsonl"
	if err := host.WriteText(ctx, store, file, read.segments); err != nil {
		return err
	}
	_, err = texts.PutVersion(ctx, host.TextPut{
		Path: req.Path, Key: textKey, Kind: textKind, SourceHash: read.hash, File: file,
	})
	return err
}

// getSegments is a book's text as segments: one from where each block element begins, cut again at
// partBytes on whole characters, with an empty segment opening each named section where it begins.
// The selector of each segment names the CFI of the block it is in. The sections and the anchors
// ascend by offset.
func getSegments(text string, sections []epub.Part, anchors []epub.Anchor) []domain.TextSegment {
	cutter := segmentCut{text: text, sections: sections, anchors: anchors}
	from := 0
	for {
		cutter.openAt(from)
		if from >= len(text) {
			return cutter.segments
		}
		to := cutter.getEnd(from)
		cutter.segments = append(cutter.segments, domain.TextSegment{Text: text[from:to], Selector: cutter.selector})
		from = to
	}
}

// segmentCut is the segments of a book cut so far, the sections and the anchors still ahead, and the
// selector of the block the cut is in.
type segmentCut struct {
	text     string
	sections []epub.Part
	anchors  []epub.Anchor
	selector domain.Selector
	segments []domain.TextSegment
}

// openAt takes the selector of the last block beginning at or before an offset, and opens each
// section beginning there.
func (c *segmentCut) openAt(from int) {
	for len(c.anchors) > 0 && c.anchors[0].Offset <= from {
		c.selector = host.Selector{CFI: c.anchors[0].CFI}.Encode()
		c.anchors = c.anchors[1:]
	}
	for len(c.sections) > 0 && c.sections[0].Offset <= from {
		c.segments = append(c.segments, domain.TextSegment{Heading: c.sections[0].Title, Selector: c.selector})
		c.sections = c.sections[1:]
	}
}

// getEnd is where the segment from an offset ends: at most partBytes on, and before the next block
// or section begins.
func (c *segmentCut) getEnd(from int) int {
	to := getCutEnd(c.text, from)
	if len(c.anchors) > 0 && c.anchors[0].Offset < to {
		to = c.anchors[0].Offset
	}
	if len(c.sections) > 0 && c.sections[0].Offset < to {
		to = c.sections[0].Offset
	}
	return to
}

// getCutEnd is where a segment from an offset ends: partBytes on, moved back to the start of a
// character.
func getCutEnd(text string, from int) int {
	end := min(from+partBytes, len(text))
	to := end
	for to > from && to < len(text) && !utf8.RuneStart(text[to]) {
		to--
	}
	if to == from {
		return end
	}
	return to
}
