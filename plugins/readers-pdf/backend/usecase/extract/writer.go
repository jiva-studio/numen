// Package extract writes the text layer of a PDF file into the plugin's folder as a text version.
package extract

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// textKey is the key the plugin keeps the text layer of a file under.
const textKey = "layer"

// TextKind is the TextKind of the text versions the plugin creates.
const TextKind domain.TextKind = "page-text"

// Files is what the writer reads of a vault: its files and its name.
type Files interface {
	host.FileReader
	GetVault(ctx context.Context) (domain.Vault, error)
}

// Writer writes the text layer of the vault's PDF files into the plugin's folder as text
// versions: every word with its page and the rectangle it covers in its selector. Each layer is
// written as background work in the host's queue.
type Writer struct {
	vaults func(domain.VaultID) Files
	texts  func(domain.VaultID) host.TextVersions
	store  func(domain.VaultID) host.StoreWriter
	queue  host.Queue
	report func(what string, err error)

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	last     layer
	running  map[string]context.CancelFunc

	workers *pdf.Workers
}

// Request names the file whose text layer is written, in the vault Vault names; empty is the
// vault the host shows when the layer is asked for.
type Request struct {
	Vault domain.VaultID
	Path  string
}

// NewWriter returns the writer of the text layers of the files read through vaults, written into
// store and reported to texts, each written in a turn of queue. report receives the failures of the
// background work.
func NewWriter(
	vaults func(domain.VaultID) Files, texts func(domain.VaultID) host.TextVersions, store func(domain.VaultID) host.StoreWriter, queue host.Queue, workers *pdf.Workers,
	report func(what string, err error),
) *Writer {
	return &Writer{
		vaults: vaults, texts: texts, store: store, queue: queue, report: report,
		wg: sync.WaitGroup{}, mu: sync.Mutex{}, isClosed: false, last: layer{hash: "", raw: nil, book: nil},
		running: map[string]context.CancelFunc{}, workers: workers,
	}
}

func (s *Writer) reportError(what string, err error) {
	if err != nil {
		s.report(what, err)
	}
}

// Wait waits for the text layers it started to be written.
func (s *Writer) Wait() { s.wg.Wait() }

// Close stops the text layers it started and waits for them to end.
func (s *Writer) Close() {
	s.mu.Lock()
	s.isClosed = true
	for _, cancel := range s.running {
		cancel()
	}
	s.mu.Unlock()
	s.Wait()
}

func (s *Writer) readLayer(ctx context.Context, files host.FileReader, path string) (layer, error) {
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
		s.writeLayer(runCtx, req)
	})
	return nil
}

func (s *Writer) finish(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, key)
}

func (s *Writer) writeLayer(ctx context.Context, req Request) {
	task := host.Task{
		ID: "text:" + req.Path, Title: "Reading the text of " + path.Base(req.Path), Detail: req.Path,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="task counters start at zero"
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "", IsUserStarted: false,
	}
	err := s.putLayer(ctx, req, task)
	if err != nil && ctx.Err() == nil {
		task.FailureReason = err.Error()
		if reportErr := s.queue.ReportTask(context.WithoutCancel(ctx), task); reportErr != nil {
			s.reportError("show the task "+task.ID, reportErr)
		}
		return
	}
	if finishErr := s.queue.FinishTask(context.WithoutCancel(ctx), task.ID); finishErr != nil {
		s.reportError("finish the task "+task.ID, finishErr)
	}
}

func (s *Writer) putLayer(ctx context.Context, req Request, task host.Task) (err error) {
	turn, err := s.queue.EnqueueWork(ctx, host.Work{IsUserStarted: false, OnWaiting: nil})
	if err != nil {
		return fmt.Errorf("wait for a turn to read %s: %w", req.Path, err)
	}
	defer func() { err = errors.Join(err, turn.Release()) }()
	if err := s.queue.ReportTask(ctx, task); err != nil {
		return err
	}
	files, texts, store := s.vaults(req.Vault), s.texts(req.Vault), s.store(req.Vault)
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
		Path: req.Path, Key: textKey, Source: "", Kind: TextKind, InputHash: read.hash, File: file,
	})
	return err
}
