package pool

import (
	"errors"
	"image"
	"sync"
	"testing"
	"time"
)

// A held document is one being drawn from, and it says when the drawing began
// and stops there until it is let go.
type held struct {
	mu      sync.Mutex
	drawing chan struct{}
	let     chan struct{}
	isShut  bool
}

func (h *held) Pages() int       { return 1 }
func (h *held) Label(int) string { return "1" }
func (h *held) Size(int) (float64, float64, error) {
	return 612, 792, nil
}

func (h *held) Image(int, int) (image.Image, error) {
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
func TestGoingWaitsForWhatIsBeingDrawn(t *testing.T) {
	one := &held{drawing: make(chan struct{}), let: make(chan struct{})}
	docs := NewDocuments(MostOpen, OpenIdleFor)
	mark := Fingerprint{Path: "library/a.pdf", Size: 1, Mtime: 1}

	drawn := make(chan error, 1)
	go func() {
		doc, give, err := docs.Take(t.Context(), mark, func() (Scan, error) { return one, nil })
		if err != nil {
			drawn <- err
			return
		}
		defer give()
		_, err = doc.Scan.Image(0, 72)
		drawn <- err
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
	if err := <-drawn; err != nil {
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
	docs := NewDocuments(MostOpen, OpenIdleFor)
	done := make(chan struct{})
	go func() { docs.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("closing an empty window waited")
	}
}

// A window that has begun going opens nothing more.
//
// A document taken after the last one was given back would be a document
// nobody is left to close, and the going would wait on it for ever.
func TestAWindowGoingOpensNothingMore(t *testing.T) {
	docs := NewDocuments(MostOpen, OpenIdleFor)
	docs.Close()

	mark := Fingerprint{Path: "library/a.pdf", Size: 1, Mtime: 1}
	opens := 0
	_, _, err := docs.Take(t.Context(), mark, func() (Scan, error) {
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
