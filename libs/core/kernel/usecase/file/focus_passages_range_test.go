package file_test

import (
	"math"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// The 32-bit range is a limit of the wire to the window, which its converters enforce; the use case
// shows any passage of the file.
func TestFocusPassagesShowsAPassageThatEndsPastThe32BitRange(t *testing.T) {
	focus, view, v := newFocusPassages(t)

	err := focus.Execute(t.Context(), v, file.FocusPassagesInput{
		Path: talkPath, Passages: []domain.ByteRange{{Start: math.MaxInt32 - 2, End: math.MaxInt32 + 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.requested) != 1 {
		t.Errorf("the window received %+v", view.requested)
	}
}

func TestFocusPassagesAcceptsAPassageThatEndsAtThe32BitLimit(t *testing.T) {
	focus, view, v := newFocusPassages(t)

	err := focus.Execute(t.Context(), v, file.FocusPassagesInput{
		Path: talkPath, Passages: []domain.ByteRange{{Start: math.MaxInt32 - 2, End: math.MaxInt32}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.requested) != 1 {
		t.Errorf("the window received %+v", view.requested)
	}
}
