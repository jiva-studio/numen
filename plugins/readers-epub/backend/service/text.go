package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

// textKey is the key the plugin keeps the text of a book under.
const textKey = "text"

// partBytes is the most text one segment holds.
const partBytes = 64 << 10

// textKind is the TextKind of the text versions the plugin produces.
const textKind domain.TextKind = "book-text"

// TextServer writes the text of the vault's EPUB books into the plugin's folder as text versions,
// each named section opened where it begins and each segment selected by the CFI of the character
// it begins at. The text and its offsets are the ones the reader sets the book with. Each text is
// written as background work in the host's queue.
type TextServer struct {
	files host.Vault
	texts host.Texts
	store host.Store
	queue host.Queue

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	last     bookText
	running  map[string]context.CancelFunc
}

var _ host.TextProvider = (*TextServer)(nil)

// NewTextServer is the texts of the books read through files, written into store and reported to
// texts, each written in a turn of queue.
func NewTextServer(files host.Vault, texts host.Texts, store host.Store, queue host.Queue) *TextServer {
	return &TextServer{
		files: files, texts: texts, store: store, queue: queue,
		wg:       sync.WaitGroup{},
		mu:       sync.Mutex{},
		isClosed: false,
		last:     bookText{hash: "", text: "", segments: nil},
		running:  map[string]context.CancelFunc{},
	}
}

// Wait waits for the texts it started to be written.
func (s *TextServer) Wait() { s.wg.Wait() }

// Close stops the texts it started and waits for them to end.
func (s *TextServer) Close() {
	s.mu.Lock()
	s.isClosed = true
	for _, cancel := range s.running {
		cancel()
	}
	s.mu.Unlock()
	s.Wait()
}

// bookText is one file's text as segments, and the hash of the bytes it was read from.
type bookText struct {
	hash     string
	text     string
	segments []domain.TextSegment
}

func isUnreadable(err error) bool {
	return errors.Is(err, epub.ErrNotArchive) || errors.Is(err, epub.ErrNoContainer) ||
		errors.Is(err, epub.ErrNoPackage) || errors.Is(err, epub.ErrEncrypted)
}

func (s *TextServer) readBook(ctx context.Context, files host.Vault, path string) (bookText, error) {
	raw, _, err := files.Read(ctx, path)
	if err != nil {
		return bookText{}, err
	}
	hash := domain.HashContent(raw)
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	// gomutants:disable-next-line BRANCH_IF reason="single-book cache avoids re-parsing unchanged EPUB bytes"
	if last.hash == hash {
		return last, nil
	}
	book, err := epub.Read(raw)
	if err != nil {
		return bookText{}, err
	}
	segments, err := getSegments(book.Text, book.Parts, book.ReadPlaces())
	// gomutants:disable-next-line BRANCH_IF reason="cfiSelector consists of a string field that always marshals"
	if err != nil {
		return bookText{}, err
	}
	read := bookText{hash: hash, text: book.Text, segments: segments}
	s.mu.Lock()
	// gomutants:disable-next-line STATEMENT_REMOVE reason="caches extracted book text for subsequent calls"
	s.last = read
	s.mu.Unlock()
	return read, nil
}

// CreateText starts writing the text of a book as a version made from the file, and returns. The
// A book whose text is being written already is not started again.
func (s *TextServer) CreateText(ctx context.Context, req host.TextCreation) error {
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
		s.writeText(runCtx, req)
	})
	return nil
}

func (s *TextServer) finish(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, key)
}

func (s *TextServer) writeText(ctx context.Context, req host.TextCreation) {
	task := host.Task{
		ID: "text:" + req.Path, Title: "Reading the text of " + path.Base(req.Path), Detail: req.Path,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="task counters and unit start at zero"
		CompletedCount: 0, TotalCount: 0, Unit: 0, FailureReason: "", IsUserStarted: false,
	}
	err := s.putText(ctx, req, task)
	if err != nil && ctx.Err() == nil {
		task.FailureReason = err.Error()
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="task report failure is logged to stderr"
		if reportErr := s.queue.ReportTask(context.WithoutCancel(ctx), task); reportErr != nil {
			fmt.Fprintf(os.Stderr, "readers-epub: show the task %s: %v\n", task.ID, reportErr)
		}
		return
	}
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="task finish failure is logged to stderr"
	if finishErr := s.queue.FinishTask(context.WithoutCancel(ctx), task.ID); finishErr != nil {
		fmt.Fprintf(os.Stderr, "readers-epub: finish the task %s: %v\n", task.ID, finishErr)
	}
}

func (s *TextServer) putText(ctx context.Context, req host.TextCreation, task host.Task) (err error) {
	turn, err := s.queue.EnqueueWork(ctx, host.Work{IsUserStarted: false, OnWaiting: nil})
	if err != nil {
		return fmt.Errorf("wait for a turn to read %s: %w", req.Path, err)
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="releases queue turn when putText finishes"
	defer func() { err = errors.Join(err, turn.Release()) }()
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
		Path: req.Path, Key: textKey, Source: "", Kind: textKind, InputHash: read.hash, File: file,
	})
	return err
}

func getSegments(text string, sections []epub.Part, places epub.Places) ([]domain.TextSegment, error) {
	cutter := segmentCut{text: text, sections: sections, blocks: places.Blocks}
	var segments []domain.TextSegment
	from := 0
	for {
		selector, err := getSelector(places, from)
		// gomutants:disable-next-line BRANCH_IF reason="cfiSelector consists of a string field that always marshals"
		if err != nil {
			return nil, err
		}
		for _, title := range cutter.openAt(from) {
			segments = append(segments, domain.TextSegment{Text: "", Heading: title, Selector: selector})
		}
		if from >= len(text) {
			return segments, nil
		}
		to := cutter.getEnd(from)
		segments = append(segments, domain.TextSegment{Text: text[from:to], Heading: "", Selector: selector})
		from = to
	}
}

func getSelector(places epub.Places, offset int) (domain.Selector, error) {
	place, ok := places.GetCFI(offset)
	if !ok {
		return "", nil
	}
	return cfiSelector{CFI: place.String()}.encode()
}

// segmentCut is the text of a book, and the sections and block starts still ahead of the cut.
type segmentCut struct {
	text     string
	sections []epub.Part
	blocks   []int
}

func (c *segmentCut) openAt(from int) []string {
	for len(c.blocks) > 0 && c.blocks[0] <= from {
		c.blocks = c.blocks[1:]
	}
	var titles []string
	for len(c.sections) > 0 && c.sections[0].Offset <= from {
		titles = append(titles, c.sections[0].Title)
		c.sections = c.sections[1:]
	}
	return titles
}

func (c *segmentCut) getEnd(from int) int {
	to := getCutEnd(c.text, from)
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="setting to = c.blocks[0] when equal is a no-op"
	if len(c.blocks) > 0 && c.blocks[0] < to {
		to = c.blocks[0]
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="setting to = c.sections[0].Offset when equal is a no-op"
	if len(c.sections) > 0 && c.sections[0].Offset < to {
		to = c.sections[0].Offset
	}
	return to
}

func getCutEnd(text string, from int) int {
	end := min(from+partBytes, len(text))
	to := end
	// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_BOUNDARY,CONDITIONALS_NEGATION,INVERT_LOGICAL reason="text[from] is always a rune start and partBytes exceeds any UTF-8 rune length"
	for to > from && to < len(text) && !utf8.RuneStart(text[to]) {
		to--
	}
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="partBytes is 65536 so backing up at most 3 bytes never reaches from"
	if to == from {
		return end
	}
	return to
}
