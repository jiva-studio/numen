package api_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
)

func TestTheErrorCodesOfANoteNotFoundAndAConflictAreSpelledInCamelCase(t *testing.T) {
	if api.ErrorCodeNotFound != "notFound" {
		t.Errorf("ErrorCodeNotFound is %q, want notFound", api.ErrorCodeNotFound)
	}
	if api.ErrorCodeConflict != "conflict" {
		t.Errorf("ErrorCodeConflict is %q, want conflict", api.ErrorCodeConflict)
	}
}

func TestNoErrorCodeIsSpelledMissingOrStale(t *testing.T) {
	for _, code := range api.GetErrorCodeValues() {
		if code == "missing" || code == "stale" {
			t.Errorf("the error codes hold %q", code)
		}
	}
	if !slices.Contains(api.GetErrorCodeValues(), api.ErrorCodeNotFound) || !slices.Contains(api.GetErrorCodeValues(), api.ErrorCodeConflict) {
		t.Errorf("the error codes %v leave out notFound or conflict", api.GetErrorCodeValues())
	}
}
