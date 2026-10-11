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
var workers struct {
	sync.Mutex
	pool pdfium.Pool
	// size is the worker count the pool is started with; zero is one per CPU.
	size int
	// idle is how many returned workers the pool keeps alive; zero keeps none.
	idle int
	// open is the number of documents open on the pool.
	open int
	// release fires when the pool has had no open document for idleRelease.
	release *time.Timer
}

// idleRelease is how long an unused pool keeps its workers; a worker holds
// memory for as long as it lives.
var idleRelease = 5 * time.Minute

// idleMemoryPercent is the share of the available memory the idle workers may
// hold together.
const idleMemoryPercent = 5

// workerBytes is the memory one worker holds once it has read a document: the
// linear memory of a freshly instantiated pdfium module plus room for the
// document's working set.
const workerBytes int64 = 32 << 20

// ComputeIdleWorkers is how many returned workers to keep alive: the planned
// worker count, capped to what fits in idleMemoryPercent of the available
// memory. Zero when the available memory is unknown.
func ComputeIdleWorkers(planned int, availableBytes int64) int {
	if planned <= 0 || availableBytes <= 0 {
		return 0
	}
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

// waitForWorker is how long a reader waits for one to come free. Reading a
// document takes a second or two, and the wait is bounded so that a worker
// wedged inside the library is a failure rather than a program that stops.
const waitForWorker = 2 * time.Minute

// pool is the workers this process reads documents on, started the first time
// one is wanted. A start that failed is tried again by the next reader.
func pool() (pdfium.Pool, error) {
	workers.Lock()
	defer workers.Unlock()
	if workers.pool != nil {
		return workers.pool, nil
	}
	total := getPoolSize()
	made, err := webassembly.Init(webassembly.Config{
		MaxTotal:     total,
		MaxIdle:      min(workers.idle, total),
		ReuseWorkers: workers.idle > 0,
	})
	if err != nil {
		return nil, err
	}
	workers.pool = made
	return made, nil
}

// getWorker borrows a worker and counts the document that will be open on it.
func getWorker() (pdfium.Pdfium, error) {
	p, err := pool()
	if err != nil {
		return nil, err
	}
	workers.Lock()
	workers.open++
	if workers.release != nil {
		workers.release.Stop()
		workers.release = nil
	}
	workers.Unlock()
	worker, err := p.GetInstance(waitForWorker)
	if err != nil {
		finishWorker()
		return nil, err
	}
	return worker, nil
}

// putWorker returns a worker to the pool.
func putWorker(worker pdfium.Pdfium) {
	worker.Close()
	finishWorker()
}

// finishWorker counts a document closed and, once none is open, schedules the
// pool's release.
func finishWorker() {
	workers.Lock()
	defer workers.Unlock()
	workers.open--
	if workers.open > 0 || workers.pool == nil || workers.idle == 0 {
		return
	}
	if workers.release != nil {
		workers.release.Stop()
	}
	workers.release = time.AfterFunc(idleRelease, releasePool)
}

// releasePool closes the pool and every worker in it when no document is
// open; the next document starts a new pool.
func releasePool() {
	workers.Lock()
	defer workers.Unlock()
	if workers.open > 0 || workers.pool == nil {
		return
	}
	workers.pool.Close()
	workers.pool = nil
	workers.release = nil
}

// A document is one PDF, open, on a worker of its own.
type document struct {
	worker pdfium.Pdfium
	ref    references.FPDF_DOCUMENT
	pages  int
	file   io.Closer
}

// source is where a document's bytes are read from, a block at a time as the
// library asks for them.
type source struct {
	reader io.ReadSeeker
	size   int64
	file   io.Closer
}

func openBytes(raw []byte) (*document, error) {
	return open(source{reader: bytes.NewReader(raw), size: int64(len(raw))})
}

func openPath(path string) (*document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	return openReader(file)
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
		file.Close()
		return nil, fmt.Errorf("pdf: %w", err)
	}
	return open(source{reader: file, size: size, file: file})
}

// open takes ownership of the source's file and closes it when the document is
// closed, or when opening fails.
func open(src source) (*document, error) {
	closeFile := func() {
		if src.file != nil {
			src.file.Close()
		}
	}
	if src.size == 0 {
		closeFile()
		return nil, fmt.Errorf("%w: the file is empty", ErrNotPDF)
	}
	worker, err := getWorker()
	if err != nil {
		closeFile()
		return nil, fmt.Errorf("pdf: %w", err)
	}

	opened, err := worker.OpenDocument(&requests.OpenDocument{FileReader: src.reader, FileReaderSize: src.size})
	if err != nil {
		putWorker(worker)
		closeFile()
		// A file that asks for a password is a file whose text nobody here can
		// read, and it is worth saying so rather than saying it is malformed.
		if errors.Is(err, pdfiumerrors.ErrPassword) || errors.Is(err, pdfiumerrors.ErrSecurity) {
			return nil, fmt.Errorf("%w: %w", ErrEncrypted, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrNotPDF, err)
	}

	doc := &document{worker: worker, ref: opened.Document, file: src.file}
	count, err := worker.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.ref})
	if err != nil {
		doc.close()
		return nil, fmt.Errorf("%w: %w", ErrNotPDF, err)
	}
	doc.pages = count.PageCount
	return doc, nil
}

// close gives the worker back and releases the file. Closing the instance
// closes the document with it.
func (d *document) close() {
	putWorker(d.worker)
	if d.file != nil {
		d.file.Close()
	}
}

// title is what the document says it is called. Most say nothing, and a good
// many of the rest say the name of the program that made them.
func (d *document) title() string {
	meta, err := d.worker.FPDF_GetMetaText(&requests.FPDF_GetMetaText{Document: d.ref, Tag: "Title"})
	if err != nil {
		return ""
	}
	return tidy(meta.Value)
}

// text is the text layer of one page, and is empty for a page that has none.
//
// A page that cannot be read is a page that says nothing: one page of a
// document is not worth refusing the document over.
func (d *document) text(index int) string {
	page, err := d.worker.GetPageText(&requests.GetPageText{
		Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}},
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
