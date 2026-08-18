package webui

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// A change reaches everyone drawing the vault the way a note put in front of
// the person does.
func TestAChangeReachesEveryoneDrawing(t *testing.T) {
	api := &API{Drawing: drawing()}
	line, done := api.Drawing.listen()
	t.Cleanup(done)

	said := domain.Editing{
		Change: "one", Path: "Aggressor.md", From: 2, To: 12, Text: "An axe",
	}
	if err := api.Viewing().Editing(t.Context(), said); err != nil {
		t.Fatal(err)
	}

	if got := <-line; got != said {
		t.Errorf("what arrived is %+v", got)
	}
}

// Every report carries the whole of what a change is doing, so a listener that
// has not read one is given the newest instead of the backlog.
func TestTheLastChangeDrawnIsTheOneWaiting(t *testing.T) {
	drawing := drawing()
	line, done := drawing.listen()
	t.Cleanup(done)

	drawing.tell(domain.Editing{Change: "one", Text: "first"})
	drawing.tell(domain.Editing{Change: "one", Text: "second"})

	if got := <-line; got.Text != "second" {
		t.Errorf("what waited is %q", got.Text)
	}
}

// The report that ends a change is not one a listener may be spared: a drawing
// that is never ended stays on the screen.
func TestTheReportThatEndsAChangeIsNotReplaced(t *testing.T) {
	drawing := drawing()
	line, done := drawing.listen()
	t.Cleanup(done)

	drawing.tell(domain.Editing{Change: "one", Done: true})
	drawing.tell(domain.Editing{Change: "one", Text: "more of it"})

	got := <-line
	if got.Change != "one" || !got.Done {
		t.Errorf("what waited is %+v, want the report that ended change one", got)
	}
	// What could not replace it waits behind it.
	if got := <-line; got.Text != "more of it" {
		t.Errorf("what waited behind it is %+v, want the report of change one", got)
	}
}

// Several changes are made at once, and each is drawn in a place of its own: a
// report about one note may not take the place of an unread report about
// another.
func TestAChangeDoesNotDisplaceOneToAnotherNote(t *testing.T) {
	drawing := drawing()
	line, done := drawing.listen()
	t.Cleanup(done)

	drawing.tell(domain.Editing{Change: "one", Path: "Aggressor.md", Text: "An axe"})
	drawing.tell(domain.Editing{Change: "two", Path: "Fugue.md", Text: "A theme"})
	drawing.tell(domain.Editing{Change: "two", Path: "Fugue.md", Text: "A theme answered"})

	if got := <-line; got.Change != "one" || got.Text != "An axe" {
		t.Errorf("the first report waiting is %+v, want change one saying %q", got, "An axe")
	}
	if got := <-line; got.Change != "two" || got.Text != "A theme answered" {
		t.Errorf("the next is %+v, want change two saying %q", got, "A theme answered")
	}
	if len(line) != 0 {
		t.Errorf("got %d reports queued behind them, want 0", len(line))
	}
}
