package pdf

import (
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The library is compiled from C and is fed whatever the vault holds. Every
// door into it defers this, so one document or one page it cannot survive is an
// error and not the end of the process the person's window runs in.
func TestAPanicInsideTheLibraryIsAnErrorAboutOnePage(t *testing.T) {
	drawn, err := func() (drawing image.Image, err error) {
		defer survived("drawing a page", &drawing, &err)
		panic("page 3 of 2")
	}()

	if drawn != nil {
		t.Errorf("a page that could not be drawn came back as %v", drawn)
	}
	if !errors.Is(err, port.ErrNotADocument) {
		t.Fatalf("it answered %v, want %v", err, port.ErrNotADocument)
	}
	// What was raised is in the error: a boundary that says nothing hides the
	// fault it caught.
	if !strings.Contains(err.Error(), "page 3 of 2") {
		t.Errorf("the error says %q, and not what was raised", err)
	}
	if !strings.Contains(err.Error(), "drawing a page") {
		t.Errorf("the error says %q, and not what was being done", err)
	}
}

// Nothing is caught where nothing was raised: the answer and the error are the
// ones the call made.
func TestWhatDidNotPanicIsLeftAlone(t *testing.T) {
	wanted := errors.New("the page is not there")
	drawn, err := func() (drawing image.Image, err error) {
		defer survived("drawing a page", &drawing, &err)
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), wanted
	}()

	if drawn == nil {
		t.Error("the drawing was taken away")
	}
	if !errors.Is(err, wanted) {
		t.Errorf("it answered %v, want %v", err, wanted)
	}
}
