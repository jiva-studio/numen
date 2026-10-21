package pool

import (
	"errors"
	"image"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// A held document is one being drawn from, and it says when the drawing began
// and stops there until it is let go.
type held struct {
	mu      sync.Mutex
	drawing chan struct{}
	let     chan struct{}
	isShut  bool
}

func (h *held) GetPageCount() int { return 1 }
func (h *held) Label(int) string  { return "1" }
func (h *held) GetPageSize(int) (float64, float64, error) {
	return 612, 792, nil
}

func (h *held) RenderPage(int, int) (image.Image, error) {
	close(h.drawing)
	<-h.let
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.isShut {
		return nil, errors.New("the document was closed underneath the drawing")
	}
	return image.NewRGBA(image.Rect(0, 0, 8, 8)), nil
}

func (h *held) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.isShut = true
}

// A window that goes waits for what is being drawn.
//
// Closing gives the library's worker back. Given back while a page is being
// drawn on it, it is taken from under the drawing.
func TestGoingWaitsForWhatIsBeingGenerated(t *testing.T) {
	one := &held{drawing: make(chan struct{}), let: make(chan struct{})}
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	mark := document.Fingerprint{Path: "library/a.pdf", Size: 1, Mtime: 1}

	renderErrCh := make(chan error, 1)
	go func() {
		doc, give, err := docs.Acquire(t.Context(), mark, func() (port.OpenDocument, error) { return one, nil })
		if err != nil {
			renderErrCh <- err
			return
		}
		defer give()
		_, err = doc.Scan.RenderPage(0, 72)
		renderErrCh <- err
	}()

	<-one.drawing

	isShut := make(chan struct{})
	go func() {
		docs.Close()
		close(isShut)
	}()

	// The close is waiting on the drawing, and the drawing has not been cut
	// from under.
	select {
	case <-isShut:
		t.Fatal("the window went while a page was being drawn")
	case <-time.After(50 * time.Millisecond):
	}

	close(one.let)
	if err := <-renderErrCh; err != nil {
		t.Errorf("the drawing was answered %v", err)
	}
	select {
	case <-isShut:
	case <-time.After(2 * time.Second):
		t.Fatal("the window never went")
	}
}

// A window with nothing open goes at once.
func TestGoingWithNothingOpenIsNotAWait(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	isDone := make(chan struct{})
	go func() { docs.Close(); close(isDone) }()
	select {
	case <-isDone:
	case <-time.After(time.Second):
		t.Fatal("closing an empty window waited")
	}
}

// A window that has begun going opens nothing more.
//
// A document taken after the last one was given back would be a document
// nobody is left to close, and the going would wait on it for ever.
func TestAWindowGoingOpensNothingMore(t *testing.T) {
	docs := NewDocuments(MaxOpenDocuments, DocumentIdleTimeout)
	docs.Close()

	mark := document.Fingerprint{Path: "library/a.pdf", Size: 1, Mtime: 1}
	opens := 0
	_, _, err := docs.Acquire(t.Context(), mark, func() (port.OpenDocument, error) {
		opens++
		return &held{drawing: make(chan struct{}), let: make(chan struct{})}, nil
	})
	if !errors.Is(err, ErrBusy) {
		t.Errorf("a document taken while the window goes was answered %v", err)
	}
	if opens != 0 {
		t.Errorf("it opened %d documents", opens)
	}
}
