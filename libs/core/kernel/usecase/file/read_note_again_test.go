package file_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestReadNoteAgainReturnsAnErrorForAnUnreadableNote(t *testing.T) {
	read := file.NewReadNoteAgain(func(context.Context, domain.Vault, []string) ([]string, error) {
		return []string{"notes/a.md"}, nil
	})

	if err := read.Execute(t.Context(), domain.Vault{}, "notes/a.md"); !errors.Is(err, file.ErrNoteUnreadable) {
		t.Fatalf("error is %v, want ErrNoteUnreadable", err)
	}
}

func TestReadNoteAgainReturnsTheRefreshError(t *testing.T) {
	failure := errors.New("index closed")
	read := file.NewReadNoteAgain(func(context.Context, domain.Vault, []string) ([]string, error) {
		return nil, failure
	})

	if err := read.Execute(t.Context(), domain.Vault{}, "notes/a.md"); !errors.Is(err, failure) {
		t.Fatalf("error is %v, want the refresh error", err)
	}
}

func TestReadNoteAgainReturnsNilForAnIndexedNote(t *testing.T) {
	var requestedPaths []string
	read := file.NewReadNoteAgain(func(_ context.Context, _ domain.Vault, paths []string) ([]string, error) {
		requestedPaths = paths
		return nil, nil
	})

	if err := read.Execute(t.Context(), domain.Vault{}, "notes/a.md"); err != nil || len(requestedPaths) != 1 || requestedPaths[0] != "notes/a.md" {
		t.Fatalf("error is %v for %v", err, requestedPaths)
	}
}
