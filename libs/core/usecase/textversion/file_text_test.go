package textversion_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

func TestFileTextReadsTheLiveTextAndRecordsAnEdit(t *testing.T) {
	s := newScene(t)
	text, err := textversion.NewFileText(s.versions, fileReaders(s.files))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := text.ReadLiveText(t.Context(), s.vault, scanPath); err != nil || ok {
		t.Fatalf("a file with no version reads as live (%v, %v)", ok, err)
	}
	s.requestText(t)
	live, ok, err := text.ReadLiveText(t.Context(), s.vault, scanPath)
	if err != nil || !ok || live.Text != "layer teh " {
		t.Fatalf("live text %q (%v, %v)", live.Text, ok, err)
	}

	edit, err := text.CreateEdit(t.Context(), s.vault, textversion.TextEdit{
		PluginID: player, Path: scanPath, Source: live.Version.GetID(), Segments: []domain.TextSegment{{Text: "layer the "}},
	})
	if err != nil {
		t.Fatal(err)
	}
	live, _, err = text.ReadLiveText(t.Context(), s.vault, scanPath)
	if err != nil || live.Version.GetID() != edit.GetID() || live.Text != "layer the " {
		t.Fatalf("after the edit the live text is %q of %s (%v)", live.Text, live.Version.GetID(), err)
	}
}

func TestNewFileTextRequiresVersionsAndReaders(t *testing.T) {
	if _, err := textversion.NewFileText(nil, fileReaders{}); err == nil {
		t.Error("a file text was made without versions")
	}
	if _, err := textversion.NewFileText(newScene(t).versions, nil); err == nil {
		t.Error("a file text was made without readers")
	}
}
