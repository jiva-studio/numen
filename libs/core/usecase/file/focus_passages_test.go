package file_test

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const talkPath = "talks/Marsh Talk.mp3"

// placeRecorder is a window that records the places it is asked to show.
type placeRecorder struct{ requested []domain.Place }

func (w *placeRecorder) FocusPlace(_ context.Context, pos domain.Place) error {
	w.requested = append(w.requested, pos)
	return nil
}
func (w *placeRecorder) ShowMove(context.Context, domain.Move) error { return nil }
func (w *placeRecorder) ShowEdit(context.Context, domain.Edit) error { return nil }

func newFocusPassages(t *testing.T) (*file.FocusPassages, *placeRecorder, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, map[string]string{
		talkPath:         "ID3 a recording of the marsh",
		"notes/Heron.md": "# Grey Heron\nHerons fish at dusk.\n",
	})
	view := &placeRecorder{}
	return file.NewFocusPassages(vaultreader.Readers{}, view), view, v
}

func TestFocusPassagesShowsTheFileWithThePassagesLitInTheirOrder(t *testing.T) {
	focus, view, v := newFocusPassages(t)

	err := focus.Execute(t.Context(), v, file.FocusPassagesInput{
		Path: talkPath, Passages: []domain.ByteRange{{Start: 40, End: 50}, {Start: 10, End: 15}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Place{{Path: talkPath, Ranges: []domain.ByteRange{{Start: 40, End: 50}, {Start: 10, End: 15}}}}
	if !reflect.DeepEqual(view.requested, want) {
		t.Errorf("the window was asked %+v, want %+v", view.requested, want)
	}
}

func TestFocusPassagesWithNoPassagesOpensTheFileAtItsBeginning(t *testing.T) {
	focus, view, v := newFocusPassages(t)

	err := focus.Execute(t.Context(), v, file.FocusPassagesInput{Path: talkPath})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.requested) != 1 || view.requested[0].Path != talkPath || len(view.requested[0].Ranges) != 0 {
		t.Errorf("the window was asked %+v", view.requested)
	}
}

func TestFocusPassagesOfANoteOpensTheNote(t *testing.T) {
	focus, view, v := newFocusPassages(t)

	err := focus.Execute(t.Context(), v, file.FocusPassagesInput{
		Path: "notes/Heron.md", Passages: []domain.ByteRange{{Start: 2, End: 12}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.requested) != 1 || view.requested[0].Path != "notes/Heron.md" {
		t.Errorf("the window was asked %+v", view.requested)
	}
}

func TestFocusPassagesLightsAtMostEightPassages(t *testing.T) {
	focus, view, v := newFocusPassages(t)
	eight := make([]domain.ByteRange, 8)
	for i := range eight {
		eight[i] = domain.ByteRange{Start: i * 10, End: i*10 + 5}
	}

	if err := focus.Execute(t.Context(), v, file.FocusPassagesInput{Path: talkPath, Passages: eight}); err != nil {
		t.Fatalf("eight passages were refused: %v", err)
	}
	if len(view.requested) != 1 || len(view.requested[0].Ranges) != 8 {
		t.Errorf("the window was asked %+v", view.requested)
	}
}

func TestFocusPassagesRefusesAPlaceItCannotShowAndDoesNotMoveTheWindow(t *testing.T) {
	focus, view, v := newFocusPassages(t)
	nine := make([]domain.ByteRange, 9)
	for i := range nine {
		nine[i] = domain.ByteRange{Start: i, End: i + 1}
	}

	for name, tc := range map[string]struct {
		in   file.FocusPassagesInput
		want string
	}{
		"no path":        {file.FocusPassagesInput{}, "name the document"},
		"negative start": {file.FocusPassagesInput{Path: talkPath, Passages: []domain.ByteRange{{Start: -1, End: 4}}}, "begins at or after the start"},
		"empty range":    {file.FocusPassagesInput{Path: talkPath, Passages: []domain.ByteRange{{Start: 3, End: 3}}}, "longer than nothing"},
		"later empty":    {file.FocusPassagesInput{Path: talkPath, Passages: []domain.ByteRange{{Start: 0, End: 4}, {Start: 9, End: 2}}}, "longer than nothing"},
		"nine passages":  {file.FocusPassagesInput{Path: talkPath, Passages: nine}, "highlight at most 8"},
	} {
		t.Run(name, func(t *testing.T) {
			err := focus.Execute(t.Context(), v, tc.in)
			if !errors.Is(err, file.ErrInvalidPassage) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("refused as %v, want ErrInvalidPassage saying %q", err, tc.want)
			}
		})
	}
	if len(view.requested) != 0 {
		t.Errorf("the user was moved to %+v", view.requested)
	}
}

func TestFocusPassagesOfAPathTheVaultDoesNotHoldIsNotFoundAndDoesNotMoveTheWindow(t *testing.T) {
	focus, view, v := newFocusPassages(t)

	err := focus.Execute(t.Context(), v, file.FocusPassagesInput{Path: "talks/Gone.mp3"})
	if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "holds nothing at talks/Gone.mp3") {
		t.Errorf("a missing file answered %v, want fs.ErrNotExist saying the vault holds nothing there", err)
	}
	if len(view.requested) != 0 {
		t.Errorf("the user was moved to %+v", view.requested)
	}
}
