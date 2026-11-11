package cards_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
)

type statter map[string]error

func (s statter) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	return domain.FileStat{Path: path}, s[path]
}

func TestGetRowFolderIsTheFolderANoteMadeFromARowGoesIn(t *testing.T) {
	files := statter{
		"Languages":        host.ErrNotANote,
		"Languages/Sa.md":  nil,
		"Top.md":           nil,
		"Languages/Gone.m": host.ErrNotExist,
	}
	for row, want := range map[string]string{
		"":                "",
		"Languages":       "Languages",
		"Languages/Sa.md": "Languages",
		"Top.md":          "",
	} {
		got, err := cards.GetRowFolder(t.Context(), files, row)
		if err != nil || got != want {
			t.Errorf("row %q gave %q, %v, want %q", row, got, err, want)
		}
	}
}

func TestGetRowFolderFailsForAMissingRow(t *testing.T) {
	_, err := cards.GetRowFolder(t.Context(), statter{"Gone.md": host.ErrNotExist}, "Gone.md")
	if !errors.Is(err, cards.ErrRowGone) {
		t.Errorf("expected ErrRowGone, got %v", err)
	}
	boom := errors.New("host boom")
	if _, err := cards.GetRowFolder(t.Context(), statter{"A.md": boom}, "A.md"); !errors.Is(err, boom) {
		t.Errorf("expected the host failure, got %v", err)
	}
}
