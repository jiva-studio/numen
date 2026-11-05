package file_test

import (
	"errors"
	"math"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestFocusPassagesRefusesAPassageThatEndsPastThe32BitRange(t *testing.T) {
	focus, view, v := newFocusPassages(t)

	_, err := focus.Execute(t.Context(), v, file.FocusPassagesInput{
		Path: talkPath, Passages: []domain.ByteRange{{Start: math.MaxInt32 - 2, End: math.MaxInt32 + 3}},
	})
	if !errors.Is(err, file.ErrInvalidPassage) {
		t.Errorf("a passage past the 32-bit range answered %v, want ErrInvalidPassage", err)
	}
	if len(view.requested) != 0 {
		t.Errorf("the window was asked %+v", view.requested)
	}
}

func TestFocusPassagesAcceptsAPassageThatEndsAtThe32BitLimit(t *testing.T) {
	focus, view, v := newFocusPassages(t)

	_, err := focus.Execute(t.Context(), v, file.FocusPassagesInput{
		Path: talkPath, Passages: []domain.ByteRange{{Start: math.MaxInt32 - 2, End: math.MaxInt32}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.requested) != 1 {
		t.Errorf("the window was asked %+v", view.requested)
	}
}
