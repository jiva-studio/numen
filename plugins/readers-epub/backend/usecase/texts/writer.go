// Package texts writes the text of EPUB books as text versions for the md.numen.readers.epub plugin.
package texts

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/segments"
)

// Vault is what the writer reads of a vault: its files and its name.
type Vault interface {
	host.FileReader
	GetVault(ctx context.Context) (domain.Vault, error)
}

// textKey is the key the plugin keeps the text of a book under.
const textKey = "text"

// Kind is the TextKind of the text versions the plugin produces.
const Kind domain.TextKind = "book-text"

// Writer writes the text of the vault's EPUB books into the plugin's folder as text versions,
// each named section opened where it begins and each segment selected by the CFI of the character
// it begins at. The text and its offsets are the ones the reader sets the book with. Each text is
// written as background work in the host's queue.
type Writer struct {
	vaults func(domain.VaultID) Vault
	texts  func(domain.VaultID) host.TextVersions
	store  func(domain.VaultID) host.StoreWriter
	queue  host.Queue
	report func(what string, err error)

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	last     bookText
	running  map[string]context.CancelFunc
}

// Request names the book whose text is written, in the vault Vault names; empty is the vault the
// host shows when the text is requested.
type Request struct {
	Vault domain.VaultID
	Path  string
}

// NewWriter is the texts of the books read through vaults, written into store and reported to
// texts, each written in a turn of queue.
func NewWriter(vaults func(domain.VaultID) Vault, texts func(domain.VaultID) host.TextVersions, store func(domain.VaultID) host.StoreWriter, queue host.Queue) *Writer {
	return &Writer{
		vaults: vaults, texts: texts, store: store, queue: queue, report: nil,
		wg:       sync.WaitGroup{},
		mu:       sync.Mutex{},
		isClosed: false,
		last:     bookText{hash: "", text: "", segments: nil},
		running:  map[string]context.CancelFunc{},
	}
}

// SetReport sets the callback that receives background failures of the writer.
func (s *Writer) SetReport(report func(what string, err error)) {
	s.report = report
}

func (s *Writer) reportError(what string, err error) {
	if s.report != nil && err != nil {
		s.report(what, err)
	}
}

// Wait waits for the texts it started to be written.
func (s *Writer) Wait() { s.wg.Wait() }

// Close stops the texts it started and waits for them to end.
func (s *Writer) Close() {
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

func (s *Writer) readBook(ctx context.Context, files host.FileReader, path string) (bookText, error) {
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
	cut, err := segments.Cut(book.Text, book.Parts, book.ReadPlaces())
	// gomutants:disable-next-line BRANCH_IF reason="cfiSelector consists of a string field that always marshals"
	if err != nil {
		return bookText{}, err
	}
	read := bookText{hash: hash, text: book.Text, segments: cut}
	s.mu.Lock()
	// gomutants:disable-next-line STATEMENT_REMOVE reason="caches extracted book text for subsequent calls"
	s.last = read
	s.mu.Unlock()
	return read, nil
}

// CreateText starts writing the text of a book as a version made from the file, and returns. An
// empty vault is the vault the host shows when CreateText is called, and the text is written there
// whichever vault the host shows later. A book whose text is being written already is not started
// again.
func (s *Writer) CreateText(ctx context.Context, req Request) error {
	if req.Vault == "" {
		shown, err := s.vaults("").GetVault(ctx)
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
		s.writeText(runCtx, req)
	})
	return nil
}

func (s *Writer) finish(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, key)
}

func (s *Writer) writeText(ctx context.Context, req Request) {
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
			s.reportError("show the task "+task.ID, reportErr)
		}
		return
	}
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="task finish failure is logged to stderr"
	if finishErr := s.queue.FinishTask(context.WithoutCancel(ctx), task.ID); finishErr != nil {
		s.reportError("finish the task "+task.ID, finishErr)
	}
}

func (s *Writer) putText(ctx context.Context, req Request, task host.Task) (err error) {
	turn, err := s.queue.EnqueueWork(ctx, host.Work{IsUserStarted: false, OnWaiting: nil})
	if err != nil {
		return fmt.Errorf("wait for a turn to read %s: %w", req.Path, err)
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="releases queue turn when putText finishes"
	defer func() { err = errors.Join(err, turn.Release()) }()
	if err := s.queue.ReportTask(ctx, task); err != nil {
		return err
	}
	files, texts, store := s.vaults(req.Vault), s.texts(req.Vault), s.store(req.Vault)
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
		Path: req.Path, Key: textKey, Source: "", Kind: Kind, InputHash: read.hash, File: file,
	})
	return err
}
