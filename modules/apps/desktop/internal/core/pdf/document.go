package pdf

import (
	"errors"
	"fmt"
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
// reader borrows one. Starting them costs a compile of the module, which is why
// they are started once and kept.
var (
	workers     pdfium.Pool
	workersOnce sync.Once
	workersWhy  error
)

// waitForWorker is how long a reader waits for one to come free. Reading a
// document takes a second or two, and the wait is bounded so that a worker
// wedged inside the library is a failure rather than a program that stops.
const waitForWorker = 2 * time.Minute

func pool() (pdfium.Pool, error) {
	workersOnce.Do(func() {
		// One worker a core, and at least two: a vault is read by one
		// goroutine today and by more as soon as anything asks it to be.
		n := max(runtime.NumCPU(), 2)
		workers, workersWhy = webassembly.Init(webassembly.Config{MaxTotal: n})
	})
	return workers, workersWhy
}

// A document is one PDF, open, on a worker of its own.
type document struct {
	worker pdfium.Pdfium
	ref    references.FPDF_DOCUMENT
	pages  int
}

func open(raw []byte) (*document, error) {
	p, err := pool()
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	worker, err := p.GetInstance(waitForWorker)
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}

	opened, err := worker.OpenDocument(&requests.OpenDocument{File: &raw})
	if err != nil {
		worker.Close()
		// A file that asks for a password is a file whose text nobody here can
		// read, and it is worth saying so rather than saying it is malformed.
		if errors.Is(err, pdfiumerrors.ErrPassword) || errors.Is(err, pdfiumerrors.ErrSecurity) {
			return nil, fmt.Errorf("%w: %w", ErrEncrypted, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrNotPDF, err)
	}

	doc := &document{worker: worker, ref: opened.Document}
	count, err := worker.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.ref})
	if err != nil {
		doc.close()
		return nil, fmt.Errorf("%w: %w", ErrNotPDF, err)
	}
	doc.pages = count.PageCount
	return doc, nil
}

// close gives the worker back. Closing the instance closes the document with
// it, so there is one thing to get right rather than two.
func (d *document) close() { d.worker.Close() }

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

// label is what the document calls a page. A book's front matter is numbered
// apart from its body, and the label is what a person reading it would say.
// Where a document names none, the page's own number is what it is called.
func (d *document) label(index int) string {
	if named, err := d.worker.FPDF_GetPageLabel(&requests.FPDF_GetPageLabel{
		Document: d.ref, Page: index,
	}); err == nil {
		if label := tidy(named.Label); label != "" {
			return label
		}
	}
	return fmt.Sprint(index + 1)
}
