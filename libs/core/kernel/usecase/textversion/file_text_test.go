package textversion_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

func TestFileTextReadsTheLiveTextAndRecordsAnEdit(t *testing.T) {
	s := newScene(t)
	text := textversion.NewFileText(s.versions.Texts, s.versions.Versions)
	if _, ok, err := text.ReadLiveText(t.Context(), s.vault, scanPath); err != nil || ok {
		t.Fatalf("a file with no version reads as live (%v, %v)", ok, err)
	}
	s.requestText(t)
	live, ok, err := text.ReadLiveText(t.Context(), s.vault, scanPath)
	if err != nil || !ok || live.Text != "layer thw " {
		t.Fatalf("live text %q (%v, %v)", live.Text, ok, err)
	}

	edit, err := text.CreateEdit(t.Context(), s.vault, textdomain.Edit{
		PluginID: player, Path: scanPath, Source: live.Version().GetID(), Segments: []domain.TextSegment{{Text: "layer the "}},
	})
	if err != nil {
		t.Fatal(err)
	}
	live, _, err = text.ReadLiveText(t.Context(), s.vault, scanPath)
	if err != nil || live.Version().GetID() != edit.GetID() || live.Text != "layer the " {
		t.Fatalf("after the edit the live text is %q of %s (%v)", live.Text, live.Version().GetID(), err)
	}
}
