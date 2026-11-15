package file_test

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const talkPath = "talks/Marsh Talk.mp3"

// placeRecorder is a window that records the places it receives.
type placeRecorder struct{ requested []domain.Place }

func (w *placeRecorder) FocusPlace(_ context.Context, pos domain.Place) error {
	w.requested = append(w.requested, pos)
	return nil
}
func (w *placeRecorder) ShowMove(context.Context, domain.Move) error { return nil }
func (w *placeRecorder) ShowEdit(context.Context, domain.Edit) error { return nil }
func (w *placeRecorder) ActivateTab(context.Context, string) error   { return nil }

func newFocusFile(t *testing.T) (*file.FocusFile, *placeRecorder, domain.Vault) {
	t.Helper()
	v := vaultfixture.NewVault(t, map[string]string{
		talkPath:         "ID3 a recording of the marsh",
		"notes/Heron.md": "# Gray Heron\nHerons fish at dusk.\n",
	})
	view := &placeRecorder{}
	return file.NewFocusFile(vaultreader.Readers{}, view), view, v
}

func TestFocusFileShowsFileWithSpansHighlightedInOrder(t *testing.T) {
	focus, view, v := newFocusFile(t)

	err := focus.Execute(t.Context(), v, file.FocusFileInput{
		Path: talkPath, Ranges: []domain.ByteRange{{Start: 40, End: 50}, {Start: 10, End: 15}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Place{{Path: talkPath, Ranges: []domain.ByteRange{{Start: 40, End: 50}, {Start: 10, End: 15}}}}
	if !reflect.DeepEqual(view.requested, want) {
		t.Errorf("the window received %+v, want %+v", view.requested, want)
	}
}

func TestFocusFileWithNoSpansOpensTheFileAtItsBeginning(t *testing.T) {
	focus, view, v := newFocusFile(t)

	err := focus.Execute(t.Context(), v, file.FocusFileInput{Path: talkPath})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.requested) != 1 || view.requested[0].Path != talkPath || len(view.requested[0].Ranges) != 0 {
		t.Errorf("the window received %+v", view.requested)
	}
}

func TestFocusFileOfANoteOpensTheNote(t *testing.T) {
	focus, view, v := newFocusFile(t)

	err := focus.Execute(t.Context(), v, file.FocusFileInput{
		Path: "notes/Heron.md", Ranges: []domain.ByteRange{{Start: 2, End: 12}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.requested) != 1 || view.requested[0].Path != "notes/Heron.md" {
		t.Errorf("the window received %+v", view.requested)
	}
}

func TestFocusFileHighlightsAtMostEightSpans(t *testing.T) {
	focus, view, v := newFocusFile(t)
	eight := make([]domain.ByteRange, 8)
	for i := range eight {
		eight[i] = domain.ByteRange{Start: i * 10, End: i*10 + 5}
	}

	if err := focus.Execute(t.Context(), v, file.FocusFileInput{Path: talkPath, Ranges: eight}); err != nil {
		t.Fatalf("eight passages were rejected: %v", err)
	}
	if len(view.requested) != 1 || len(view.requested[0].Ranges) != 8 {
		t.Errorf("the window received %+v", view.requested)
	}
}

func TestFocusFileRejectsInvalidPlaceAndKeepsTheWindow(t *testing.T) {
	focus, view, v := newFocusFile(t)
	nine := make([]domain.ByteRange, 9)
	for i := range nine {
		nine[i] = domain.ByteRange{Start: i, End: i + 1}
	}

	for name, tc := range map[string]struct {
		in   file.FocusFileInput
		want string
	}{
		"no path":        {file.FocusFileInput{}, "path is empty"},
		"negative start": {file.FocusFileInput{Path: talkPath, Ranges: []domain.ByteRange{{Start: -1, End: 4}}}, "passage start must be >= 0"},
		"empty range":    {file.FocusFileInput{Path: talkPath, Ranges: []domain.ByteRange{{Start: 3, End: 3}}}, "length > 0"},
		"later empty":    {file.FocusFileInput{Path: talkPath, Ranges: []domain.ByteRange{{Start: 0, End: 4}, {Start: 9, End: 2}}}, "length > 0"},
		"nine passages":  {file.FocusFileInput{Path: talkPath, Ranges: nine}, "too many passages: at most 8"},
	} {
		t.Run(name, func(t *testing.T) {
			err := focus.Execute(t.Context(), v, tc.in)
			if !errors.Is(err, file.ErrInvalidRange) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("rejected as %v, want ErrInvalidRange saying %q", err, tc.want)
			}
		})
	}
	if len(view.requested) != 0 {
		t.Errorf("the user was moved to %+v", view.requested)
	}
}

func TestFocusFileOfPathNotInVaultFailsAndKeepsTheWindow(t *testing.T) {
	focus, view, v := newFocusFile(t)

	err := focus.Execute(t.Context(), v, file.FocusFileInput{Path: "talks/Gone.mp3"})
	if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "no file at talks/Gone.mp3") {
		t.Errorf("a missing file returned %v, want fs.ErrNotExist saying the vault contains nothing there", err)
	}
	if len(view.requested) != 0 {
		t.Errorf("the user was moved to %+v", view.requested)
	}
}
