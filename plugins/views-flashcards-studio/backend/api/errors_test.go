package api_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/apitest"
)

func TestNoteNotFoundAndConflictErrorCodesAreSnakeCase(t *testing.T) {
	if api.ErrorCodeNotFound != "not_found" {
		t.Errorf("ErrorCodeNotFound is %q, want not_found", api.ErrorCodeNotFound)
	}
	if api.ErrorCodeConflict != "conflict" {
		t.Errorf("ErrorCodeConflict is %q, want conflict", api.ErrorCodeConflict)
	}
}

func TestNoErrorCodeIsSpelledMissingOrStale(t *testing.T) {
	for _, code := range apitest.GetErrorCodeValues() {
		if code == "missing" || code == "stale" {
			t.Errorf("the error codes contain %q", code)
		}
	}
	if !slices.Contains(apitest.GetErrorCodeValues(), api.ErrorCodeNotFound) || !slices.Contains(apitest.GetErrorCodeValues(), api.ErrorCodeConflict) {
		t.Errorf("the error codes %v leave out not_found or conflict", apitest.GetErrorCodeValues())
	}
}
