package source

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// glancing is a window that records what is put in front of the person.
type glancing struct{ requested []domain.Place }

func (w *glancing) FocusPlace(_ context.Context, pos domain.Place) error {
	w.requested = append(w.requested, pos)
	return nil
}
func (w *glancing) ShowMove(context.Context, domain.Move) error { return nil }
func (w *glancing) ShowEdit(context.Context, domain.Edit) error { return nil }

func newFocus(t *testing.T) (*FocusPassage, *glancing) {
	t.Helper()
	shelved := newLibrary()
	shelved.hold(recordingPath, domain.KindRecording, newRecording([]string{"one"}), 1)
	view := &glancing{}
	return NewFocusPassage(vaults{first.ID: shelved}, view), view
}

func TestAPassageIsInsertInFrontOfThePersonWithTheOthersLit(t *testing.T) {
	focus, view := newFocus(t)

	summary, err := focus.Execute(t.Context(), first, Passage{
		Path: recordingPath, Start: 10, Length: 5, Also: []domain.ByteSpan{{From: 40, To: 50}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Place{Path: recordingPath, Spans: []domain.ByteSpan{{From: 10, To: 15}, {From: 40, To: 50}}}
	if !strings.Contains(summary, "that passage") || len(view.requested) != 1 || !samePlace(view.requested[0], want) {
		t.Errorf("summary %q, window asked %+v, want %+v", summary, view.requested, want)
	}
}

func samePlace(a, b domain.Place) bool {
	if a.Path != b.Path || len(a.Spans) != len(b.Spans) {
		return false
	}
	for i := range a.Spans {
		if a.Spans[i] != b.Spans[i] {
			return false
		}
	}
	return true
}

func TestADocumentWithNoLengthOpensAtItsBeginning(t *testing.T) {
	focus, view := newFocus(t)

	if _, err := focus.Execute(t.Context(), first, Passage{Path: recordingPath}); err != nil {
		t.Fatal(err)
	}
	if len(view.requested) != 1 || len(view.requested[0].Spans) != 0 {
		t.Errorf("the window was asked %+v", view.requested)
	}
}

func TestAPassageThatCannotBeShownMovesNobody(t *testing.T) {
	focus, view := newFocus(t)
	many := make([]domain.ByteSpan, domain.MostHighlights)
	for i := range many {
		many[i] = domain.ByteSpan{From: i, To: i + 1}
	}

	for name, tc := range map[string]struct {
		in   Passage
		want string
	}{
		"no document":      {Passage{}, "name the document"},
		"negative start":   {Passage{Path: recordingPath, Start: -1}, "begins at or after the start"},
		"too many":         {Passage{Path: recordingPath, Also: many}, "light at most"},
		"empty also":       {Passage{Path: recordingPath, Also: []domain.ByteSpan{{From: 3, To: 3}}}, "longer than nothing"},
		"nothing in vault": {Passage{Path: "library/gone.mp3"}, "holds nothing at library/gone.mp3"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := focus.Execute(t.Context(), first, tc.in); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("rejected as %v, want it to say %q", err, tc.want)
			}
		})
	}
	if len(view.requested) != 0 {
		t.Errorf("the person was moved to %+v", view.requested)
	}
}
