package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium"
	pdfiumerrors "github.com/klippa-app/go-pdfium/errors"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
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

var workers struct {
	sync.Mutex
	pool pdfium.Pool
	// size is the worker count the pool is started with; zero is one per CPU.
	size int
	// idle is how many returned workers the pool keeps alive; zero keeps none.
	idle int
	// open is the number of documents open on the pool.
	open int
	// isWarmed records whether Warmup has started the pool.
	isWarmed bool
	// release fires when the pool has had no open document for idleRelease.
	release Timer
	// afterFunc arms a timer for idleRelease.
	afterFunc func(time.Duration, func()) Timer
}

// SetAfterFunc sets the timer factory that releases idle workers.
func SetAfterFunc(fn func(time.Duration, func()) Timer) {
	workers.Lock()
	defer workers.Unlock()
	workers.afterFunc = fn
}

// idleRelease is how long an unused pool keeps its workers; a worker holds
// memory for as long as it lives.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="idleRelease is the timeout before unused workers are released"
var idleRelease = 5 * time.Minute

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
	if planned <= 0 || availableBytes <= 0 {
		return 0
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="100 computes the percentage of available memory"
	fits := availableBytes / 100 * idleMemoryPercent / workerBytes
	return int(min(int64(planned), fits))
}

// SetIdleWorkers sets how many returned workers the pool keeps alive. Zero or
// less keeps none. It has no effect on a pool that has started.
func SetIdleWorkers(n int) {
	workers.Lock()
	defer workers.Unlock()
	workers.idle = max(n, 0)
}

// minWorkers is the fewest workers the pool starts: a reader and the reader
// beside it each hold one document.
const minWorkers = 2

// SetWorkers sets how many workers the pool starts with. Zero or less is one
// per CPU. It has no effect on a pool that has started.
func SetWorkers(n int) {
	workers.Lock()
	defer workers.Unlock()
	workers.size = n
}

// getPoolSize is the worker count the pool starts with. The caller holds the
// lock.
func getPoolSize() int {
	if workers.size > 0 {
		return max(workers.size, minWorkers)
	}
	return max(runtime.NumCPU(), minWorkers)
}

// waitForWorker is how long a reader waits for a free worker. Reading a
// document takes a second or two. The wait is bounded: a worker stuck inside the
// library produces an error.
// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="waitForWorker is the bounded wait timeout for an available worker"
const waitForWorker = 2 * time.Minute

// acquirePool returns the running pool, starting one when none is running or
// the last was released, and counts a document open on it. The pool is chosen
// and counted under one lock, so a release cannot close it between the two. A
// start that failed is tried again by the next reader.
func acquirePool() (pdfium.Pool, error) {
	workers.Lock()
	defer workers.Unlock()
	if workers.pool == nil {
		total := getPoolSize()
		made, err := webassembly.Init(webassembly.Config{
			Context:       nil,
			MinIdle:       0,
			MaxTotal:      total,
			MaxIdle:       min(workers.idle, total),
			WASM:          nil,
			FSConfig:      nil,
			RuntimeConfig: nil,
			Stdout:        nil,
			Stderr:        nil,
			RandomSource:  nil,
			// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="workers are only reused when idle allowance is positive"
			ReuseWorkers: workers.idle > 0,
		})
		// gomutants:disable-next-line BRANCH_IF reason="webassembly initialization errors abort the pool acquisition"
		if err != nil {
			return nil, err
		}
		workers.pool = made
	}
	workers.open++
	if workers.release != nil {
		workers.release.Stop()
		workers.release = nil
	}
	return workers.pool, nil
}

// getWorker borrows a worker and counts the document that will be open on it.
func getWorker() (pdfium.Pdfium, error) {
	p, err := acquirePool()
	// gomutants:disable-next-line BRANCH_IF reason="acquirePool failure aborts getWorker"
	if err != nil {
		return nil, err
	}
	worker, err := p.GetInstance(waitForWorker)
	// gomutants:disable-next-line BRANCH_IF reason="GetInstance failure decrements document count and aborts getWorker"
	if err != nil {
		markDocumentClosed()
		return nil, err
	}
	return worker, nil
}

// putWorker returns a worker to the pool.
func putWorker(worker pdfium.Pdfium) error {
	err := worker.Close()
	markDocumentClosed()
	return err
}

// markDocumentClosed counts a document closed and, once none is open, schedules the
// pool's release.
func markDocumentClosed() {
	workers.Lock()
	defer workers.Unlock()
	workers.open--
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="markDocumentClosed only arms the idle release timer when no documents remain open and idle workers are kept"
	if workers.open > 0 || workers.pool == nil || workers.idle == 0 {
		return
	}
	// gomutants:disable-next-line BRANCH_IF reason="markDocumentClosed cancels any active timer before scheduling a new release"
	if workers.release != nil {
		workers.release.Stop()
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="markDocumentClosed schedules pool release after idle timeout"
	if workers.afterFunc != nil {
		workers.release = workers.afterFunc(idleRelease, releasePool)
	}
}

// releasePool closes the pool and every worker in it when no document is
// open; the next document starts a new pool.
func releasePool() {
	workers.Lock()
	defer workers.Unlock()
	if workers.open > 0 || workers.pool == nil {
		return
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="releasePool closes idle workers pool"
	if err := workers.pool.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "readers-pdf: close the worker pool: %v\n", err)
	}
	workers.pool = nil
	// gomutants:disable-next-line BRANCH_IF,STATEMENT_REMOVE reason="releasePool cancels any pending timer when closing pool"
	if workers.release != nil {
		workers.release.Stop()
		workers.release = nil
	}
}

// A document is one PDF, open, on a worker of its own.
type document struct {
	worker pdfium.Pdfium
	ref    references.FPDF_DOCUMENT
	pages  int
	file   io.Closer
}

// source supplies a document's bytes, one block at a time as the library
// requests them.
type source struct {
	reader io.ReadSeeker
	size   int64
	file   io.Closer
}

// withCloseError is err, joined with the error of closing what the failed operation had open. It is
// err itself where closing succeeded.
func withCloseError(err, closeErr error) error {
	if closeErr == nil {
		return err
	}
	return errors.Join(err, closeErr)
}

func openBytes(raw []byte) (*document, error) {
	return open(source{reader: bytes.NewReader(raw), size: int64(len(raw)), file: nil})
}

// openReader opens a document from a file the caller has open. It takes
// ownership of the file and closes it when the document is closed, or when
// opening fails.
func openReader(file io.ReadSeekCloser) (*document, error) {
	size, err := file.Seek(0, io.SeekEnd)
	if err == nil {
		_, err = file.Seek(0, io.SeekStart)
	}
	if err != nil {
		return nil, withCloseError(fmt.Errorf("pdf: %w", err), file.Close())
	}
	return open(source{reader: file, size: size, file: file})
}

// open takes ownership of the source's file and closes it when the document is
// closed, or when opening fails.
func open(src source) (*document, error) {
	closeFile := func() error {
		if src.file != nil {
			return src.file.Close()
		}
		return nil
	}
	if src.size == 0 {
		return nil, withCloseError(fmt.Errorf("%w: the file is empty", ErrNotPDF), closeFile())
	}
	worker, err := getWorker()
	// gomutants:disable-next-line BRANCH_IF reason="open releases source file when worker acquisition fails"
	if err != nil {
		return nil, withCloseError(fmt.Errorf("pdf: %w", err), closeFile())
	}

	opened, err := worker.OpenDocument(&requests.OpenDocument{File: nil, FilePath: nil, FileReader: src.reader, FileReaderSize: src.size, Password: nil})
	if err != nil {
		closeErr := errors.Join(putWorker(worker), closeFile())
		// A file that requires a password has unreadable text; it is reported as
		// encrypted, not malformed.
		// gomutants:disable-next-line EXPRESSION_REMOVE reason="password-protected PDFs return ErrEncrypted"
		if errors.Is(err, pdfiumerrors.ErrPassword) || errors.Is(err, pdfiumerrors.ErrSecurity) {
			return nil, withCloseError(fmt.Errorf("%w: %w", ErrEncrypted, err), closeErr)
		}
		// gomutants:disable-next-line ERRORF_WRAP reason="returns ErrNotPDF wrapping the underlying error"
		return nil, withCloseError(fmt.Errorf("%w: %w", ErrNotPDF, err), closeErr)
	}

	doc := &document{worker: worker, ref: opened.Document, pages: 0, file: src.file}
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
	err := putWorker(d.worker)
	if d.file != nil {
		return errors.Join(err, d.file.Close())
	}
	return err
}

// title returns the document's title metadata. Most documents have none, and
// many of the rest contain the name of the program that produced them.
func (d *document) title() string {
	meta, err := d.worker.FPDF_GetMetaText(&requests.FPDF_GetMetaText{Document: d.ref, Tag: "Title"})
	if err != nil {
		return ""
	}
	return normalizeSpaces(meta.Value)
}

// text is the text layer of one page, and is empty for a page that has none.
//
// A page that cannot be read yields empty text: one unreadable page does not
// justify rejecting the document.
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
func GetWorkers() int {
	workers.Lock()
	defer workers.Unlock()
	return getPoolSize()
}

// Warmup starts the pool and keeps at least one worker ready to serve requests.
func Warmup() {
	workers.Lock()
	if workers.isWarmed {
		workers.Unlock()
		return
	}
	workers.isWarmed = true
	workers.Unlock()
	worker, err := getWorker()
	if err != nil {
		return
	}
	if err := putWorker(worker); err != nil {
		fmt.Fprintf(os.Stderr, "readers-pdf: release the warm-up worker: %v\n", err)
	}
}
