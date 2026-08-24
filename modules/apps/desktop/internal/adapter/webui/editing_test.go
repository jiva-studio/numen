package webui

import (
	"fmt"
	"io/fs"
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
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

// A refusal is about the note the request named. The filesystem says "no such
// file" about a vault on a drive that is not plugged in as readily as about a
// note nobody wrote, and only the core knows which of the two it met — so only
// what the core named is answered as a refusal.
func TestOnlyTheCoreSaysWhichNoteIsMissing(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want v1.Refusal
		is   bool
	}{
		"the vault holds no note there": {
			err:  fmt.Errorf("read Old.md: %w", note.ErrNoNote),
			want: v1.Refusal_REFUSAL_MISSING,
			is:   true,
		},
		"the vault itself is out of reach": {
			err: fmt.Errorf("stat /vault: %w", fs.ErrNotExist),
		},
		"a title no note can be given": {
			err:  fmt.Errorf("%w: nothing to name it", note.ErrUnnameable),
			want: v1.Refusal_REFUSAL_UNNAMEABLE,
			is:   true,
		},
		"a heading that says the title as something else": {
			err:  fmt.Errorf("Old.md: %w", note.ErrNotAHeading),
			want: v1.Refusal_REFUSAL_UNNAMEABLE,
			is:   true,
		},
		"a frontmatter written on one line": {
			err:  fmt.Errorf("Old.md: %w", note.ErrInline),
			want: v1.Refusal_REFUSAL_UNREADABLE,
			is:   true,
		},
		"a frontmatter block that is never closed": {
			err:  fmt.Errorf("Old.md: %w", note.ErrUnterminated),
			want: v1.Refusal_REFUSAL_UNREADABLE,
			is:   true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			refusal, refused := refusedBy(c.err)
			if refused != c.is {
				t.Fatalf("want refused=%v, got %v (%v)", c.is, refused, refusal)
			}
			if refusal != c.want {
				t.Errorf("want %v, got %v", c.want, refusal)
			}
		})
	}
}
