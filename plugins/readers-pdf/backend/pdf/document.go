package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium"
	pdfiumerrors "github.com/klippa-app/go-pdfium/errors"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
)

// PDFium is reached as WebAssembly, so it is Go the way the rest of this is Go:
// one binary for every platform, built from any of them, with no C toolchain
// and nothing to install beside it. The library is compiled into this program.
//
// A worker holds one document at a time, so there are several of them and a
// reader borrows one. A returned worker stays alive while the pool is within
// its idle count, and the pool is released once no document has been open for
// idleRelease.

// Timer stops a scheduled callback.
type Timer interface {
	Stop() bool
}

// Workers is the pool of PDFium workers documents are read on. A worker holds one document at a
// time. The pool starts with the first document and is released once no document has been open
// for its idle release.
type Workers struct {
	mu          sync.Mutex
	pool        pdfium.Pool
	size        int
	idle        int
	open        int
	isWarmed    bool
	release     Timer
	afterFunc   func(time.Duration, func()) Timer
	idleRelease time.Duration
	compilation wazero.CompilationCache
	report      func(what string, err error)
}

// NewWorkers returns a pool that has not started, of one worker per CPU, keeping no idle worker.
func NewWorkers() *Workers {
	return &Workers{
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="NewWorkers initializes pool counters to zero"
		mu: sync.Mutex{}, pool: nil, size: 0, idle: 0, open: 0, isWarmed: false, release: nil,
		afterFunc: nil, idleRelease: defaultIdleRelease, compilation: nil, report: nil,
	}
}

// SetReport sets the callback that receives background failures of the worker pool.
func (w *Workers) SetReport(report func(what string, err error)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.report = report
}

func (w *Workers) reportError(what string, err error) {
	if w.report != nil && err != nil {
		w.report(what, err)
	}
}

// SetCompilationCache makes the pools started afterwards compile PDFium into cache, so pools that
// share a cache compile it once. It has no effect on a pool that has started.
func (w *Workers) SetCompilationCache(cache wazero.CompilationCache) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.compilation = cache
}

func (w *Workers) getRuntimeConfig() wazero.RuntimeConfig {
	if w.compilation == nil {
		return nil
	}
	// The bundled PDFium module needs the exception handling feature.
	return wazero.NewRuntimeConfig().
		WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesExceptionHandling).
		WithCompilationCache(w.compilation)
}

// SetAfterFunc sets the timer factory that releases idle workers.
func (w *Workers) SetAfterFunc(fn func(time.Duration, func()) Timer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.afterFunc = fn
}

// defaultIdleRelease is how long an unused pool keeps its workers; a worker holds
// memory for as long as it lives.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="defaultIdleRelease is the timeout before unused workers are released"
const defaultIdleRelease = 5 * time.Minute

// idleMemoryPercent is the share of the available memory the idle workers may
// hold together.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="idleMemoryPercent is the 5 percent memory cap for idle workers"
const idleMemoryPercent = 5

// workerBytes is the memory one worker holds after reading a document: the
// linear memory of a freshly instantiated pdfium module plus space for the
// document's working set.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="workerBytes is the 32 megabyte working set estimate per worker"
const workerBytes int64 = 32 << 20

// ComputeIdleWorkers is how many returned workers to keep alive: the planned
// worker count, capped to what fits in idleMemoryPercent of the available
// memory. Zero when the available memory is unknown.
func ComputeIdleWorkers(planned int, availableBytes int64) int {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT,INTEGER_INCREMENT,EXPRESSION_REMOVE reason="non-positive planned count or available memory yields zero idle workers"
	if planned <= 0 || availableBytes <= 0 {
		return 0
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="100 computes the percentage of available memory"
	fits := availableBytes / 100 * idleMemoryPercent / workerBytes
	return int(min(int64(planned), fits))
}

// SetIdleWorkers sets how many returned workers the pool keeps alive. Zero or
// less keeps none. It has no effect on a pool that has started.
func (w *Workers) SetIdleWorkers(n int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.idle = max(n, 0)
}

// minWorkers is the fewest workers the pool starts: a reader and the reader
// beside it each hold one document.
const minWorkers = 2

// SetWorkers sets how many workers the pool starts with. Zero or less is one
// per CPU. It has no effect on a pool that has started.
func (w *Workers) SetWorkers(n int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.size = n
}

func (w *Workers) getPoolSize() int {
	if w.size > 0 {
		return max(w.size, minWorkers)
	}
	return max(runtime.NumCPU(), minWorkers)
}

// waitForWorker is how long a reader waits for a free worker. Reading a
// document takes a second or two. The wait is bounded: a worker stuck inside the
// library produces an error.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="waitForWorker is the bounded wait timeout for an available worker"
const waitForWorker = 2 * time.Minute

func (w *Workers) acquirePool() (pdfium.Pool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pool == nil {
		total := w.getPoolSize()
		made, err := webassembly.Init(webassembly.Config{
			Context: nil,
			// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="MinIdle starts at zero workers"
			MinIdle:       0,
			MaxTotal:      total,
			MaxIdle:       min(w.idle, total),
			WASM:          nil,
			FSConfig:      nil,
			RuntimeConfig: w.getRuntimeConfig(),
			Stdout:        nil,
			Stderr:        nil,
			RandomSource:  nil,
			// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT,INTEGER_INCREMENT reason="workers are only reused when idle allowance is positive"
			ReuseWorkers: w.idle > 0,
		})
		// gomutants:disable-next-line BRANCH_IF reason="webassembly initialization errors abort the pool acquisition"
		if err != nil {
			return nil, err
		}
		w.pool = made
	}
	w.open++
	if w.release != nil {
		w.release.Stop()
		w.release = nil
	}
	return w.pool, nil
}

func (w *Workers) getWorker() (pdfium.Pdfium, error) {
	p, err := w.acquirePool()
	// gomutants:disable-next-line BRANCH_IF reason="acquirePool failure aborts getWorker"
	if err != nil {
		return nil, err
	}
	worker, err := p.GetInstance(waitForWorker)
	// gomutants:disable-next-line BRANCH_IF reason="GetInstance failure decrements document count and aborts getWorker"
	if err != nil {
		w.markDocumentClosed()
		return nil, err
	}
	return worker, nil
}

func (w *Workers) putWorker(worker pdfium.Pdfium) error {
	err := worker.Close()
	w.markDocumentClosed()
	return err
}

func (w *Workers) markDocumentClosed() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.open--
	// gomutants:disable-next-line EXPRESSION_REMOVE,INVERT_LOGICAL,INTEGER_DECREMENT,INTEGER_INCREMENT,BRANCH_IF reason="markDocumentClosed only arms the idle release timer when no documents remain open and idle workers are kept"
	if w.open > 0 || w.pool == nil || w.idle == 0 {
		return
	}
	// gomutants:disable-next-line BRANCH_IF reason="markDocumentClosed cancels any active timer before scheduling a new release"
	if w.release != nil {
		w.release.Stop()
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="markDocumentClosed schedules pool release after idle timeout"
	if w.afterFunc != nil {
		w.release = w.afterFunc(w.idleRelease, w.releasePool)
	}
}

func (w *Workers) releasePool() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.open > 0 || w.pool == nil {
		return
	}
	// gomutants:disable-next-line STATEMENT_REMOVE,CONDITIONALS_NEGATION,BRANCH_IF reason="releasePool closes idle workers pool and logs any failure"
	if err := w.pool.Close(); err != nil {
		w.reportError("close the worker pool", err)
	}
	w.pool = nil
	if w.release != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="releasePool cancels any pending timer when closing pool"
		w.release.Stop()
		w.release = nil
	}
}

// Close closes the pool and every worker in it, whether or not a document is open on it. A later
// document starts a new pool.
func (w *Workers) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.release != nil {
		w.release.Stop()
		w.release = nil
	}
	if w.pool == nil {
		return nil
	}
	err := w.pool.Close()
	w.pool = nil
	return err
}

// A document is one PDF, open, on a worker of its own.
type document struct {
	workers *Workers
	worker  pdfium.Pdfium
	ref     references.FPDF_DOCUMENT
	pages   int
	file    io.Closer
}

// source supplies a document's bytes, one block at a time as the library
// requests them.
type source struct {
	reader io.ReadSeeker
	size   int64
	file   io.Closer
}

func withCloseError(err, closeErr error) error {
	if closeErr == nil {
		return err
	}
	return errors.Join(err, closeErr)
}

func (w *Workers) openBytes(raw []byte) (*document, error) {
	return w.openSource(source{reader: bytes.NewReader(raw), size: int64(len(raw)), file: nil})
}

func (w *Workers) openReader(file io.ReadSeekCloser) (*document, error) {
	size, err := file.Seek(0, io.SeekEnd)
	if err == nil {
		_, err = file.Seek(0, io.SeekStart)
	}
	if err != nil {
		return nil, withCloseError(fmt.Errorf("pdf: %w", err), file.Close())
	}
	return w.openSource(source{reader: file, size: size, file: file})
}

func (w *Workers) openSource(src source) (*document, error) {
	closeFile := func() error {
		if src.file != nil {
			return src.file.Close()
		}
		return nil
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT,BRANCH_IF reason="empty source files fail immediately with ErrNotPDF"
	if src.size == 0 {
		return nil, withCloseError(fmt.Errorf("%w: the file is empty", ErrNotPDF), closeFile())
	}
	worker, err := w.getWorker()
	// gomutants:disable-next-line BRANCH_IF reason="open releases source file when worker acquisition fails"
	if err != nil {
		return nil, withCloseError(fmt.Errorf("pdf: %w", err), closeFile())
	}

	opened, err := worker.OpenDocument(&requests.OpenDocument{File: nil, FilePath: nil, FileReader: src.reader, FileReaderSize: src.size, Password: nil})
	if err != nil {
		closeErr := errors.Join(w.putWorker(worker), closeFile())
		// A file that requires a password has unreadable text; it is reported as
		// encrypted, not malformed.
		// gomutants:disable-next-line EXPRESSION_REMOVE,INVERT_LOGICAL,BRANCH_IF reason="password-protected PDFs return ErrEncrypted"
		if errors.Is(err, pdfiumerrors.ErrPassword) || errors.Is(err, pdfiumerrors.ErrSecurity) {
			return nil, withCloseError(fmt.Errorf("%w: %w", ErrEncrypted, err), closeErr)
		}
		// gomutants:disable-next-line ERRORF_WRAP reason="returns ErrNotPDF wrapping the underlying error"
		return nil, withCloseError(fmt.Errorf("%w: %w", ErrNotPDF, err), closeErr)
	}

	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="pages field is overwritten after FPDF_GetPageCount"
	doc := &document{workers: w, worker: worker, ref: opened.Document, pages: 0, file: src.file}
	count, err := worker.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.ref})
	// gomutants:disable-next-line BRANCH_IF reason="aborts when page count cannot be retrieved"
	if err != nil {
		return nil, withCloseError(fmt.Errorf("%w: %w", ErrNotPDF, err), doc.Close())
	}
	doc.pages = count.PageCount
	return doc, nil
}

// Close gives the worker back and releases the file. Closing the instance
// closes the document with it.
func (d *document) Close() error {
	err := d.workers.putWorker(d.worker)
	if d.file != nil {
		return errors.Join(err, d.file.Close())
	}
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="worker close error is propagated when no source file is attached"
	return err
}

func (d *document) title() string {
	meta, err := d.worker.FPDF_GetMetaText(&requests.FPDF_GetMetaText{Document: d.ref, Tag: "Title"})
	if err != nil {
		return ""
	}
	return normalizeSpaces(meta.Value)
}

func (d *document) text(index int) string {
	page, err := d.worker.GetPageText(&requests.GetPageText{
		Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}, ByReference: nil},
	})
	if err != nil {
		return ""
	}
	return page.Text
}

// GetWorkers is the worker count the pool starts with, or has started with.
func (w *Workers) GetWorkers() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.getPoolSize()
}

// Warmup starts the pool and keeps at least one worker ready to serve requests.
func (w *Workers) Warmup() {
	w.mu.Lock()
	if w.isWarmed {
		w.mu.Unlock()
		return
	}
	w.isWarmed = true
	w.mu.Unlock()
	worker, err := w.getWorker()
	if err != nil {
		return
	}
	if err := w.putWorker(worker); err != nil {
		w.mu.Lock()
		report := w.report
		w.mu.Unlock()
		if report != nil {
			report("release the warm-up worker", err)
		}
	}
}
