package container

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestNoteReaderReturnsAnErrorForAnUnreadableNote(t *testing.T) {
	read := newNoteReader(func(context.Context, domain.Vault, []string) (vault.RefreshResult, error) {
		return vault.RefreshResult{Unreadable: []string{"notes/a.md"}}, nil
	})

	if err := read(t.Context(), domain.Vault{}, "notes/a.md"); !errors.Is(err, errNoteUnreadable) {
		t.Fatalf("error is %v, want errNoteUnreadable", err)
	}
}

func TestNoteReaderReturnsTheRefreshError(t *testing.T) {
	failure := errors.New("index closed")
	read := newNoteReader(func(context.Context, domain.Vault, []string) (vault.RefreshResult, error) {
		return vault.RefreshResult{}, failure
	})

	if err := read(t.Context(), domain.Vault{}, "notes/a.md"); !errors.Is(err, failure) {
		t.Fatalf("error is %v, want the refresh error", err)
	}
}

func TestNoteReaderReturnsNilForAnIndexedNote(t *testing.T) {
	var requestedPaths []string
	read := newNoteReader(func(_ context.Context, _ domain.Vault, paths []string) (vault.RefreshResult, error) {
		requestedPaths = paths
		return vault.RefreshResult{Indexed: paths}, nil
	})

	if err := read(t.Context(), domain.Vault{}, "notes/a.md"); err != nil || len(requestedPaths) != 1 || requestedPaths[0] != "notes/a.md" {
		t.Fatalf("error is %v for %v", err, requestedPaths)
	}
}
