package apierror_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/apierror"
)

func TestGetDeckErrorNamesWhyNoDeckWasRead(t *testing.T) {
	if code, ok := apierror.GetDeckError(note.TooLarge, format.TypeDeck); !ok || code != api.ErrorCodeDeckTooLarge {
		t.Fatalf("unexpected GetDeckError(TooLarge): %v, %v", code, ok)
	}
	if code, ok := apierror.GetDeckError(note.Ok, format.TypeDeck); ok || code != api.ErrorCodeUnspecified {
		t.Fatalf("unexpected GetDeckError(Ok, TypeDeck): %v, %v", code, ok)
	}
	if code, ok := apierror.GetDeckError(note.Ok, format.TypeCardModel); !ok || code != api.ErrorCodeNotADeck {
		t.Fatalf("unexpected GetDeckError(Ok, TypeCardModel): %v, %v", code, ok)
	}
	if code, ok := apierror.GetDeckError(note.Missing, format.TypeCardModel); !ok || code != api.ErrorCodeNotFound {
		t.Fatalf("unexpected GetDeckError(Missing, TypeCardModel): %v, %v", code, ok)
	}
}

func TestGetCardModelErrorNamesWhyNoCardModelWasRead(t *testing.T) {
	if code, ok := apierror.GetCardModelError(note.Ok, format.TypeCardModel); ok || code != api.ErrorCodeUnspecified {
		t.Fatalf("unexpected GetCardModelError(Ok, TypeCardModel): %v, %v", code, ok)
	}
	if code, ok := apierror.GetCardModelError(note.Ok, format.TypeDeck); !ok || code != api.ErrorCodeNotACardModel {
		t.Fatalf("unexpected GetCardModelError(Ok, TypeDeck): %v, %v", code, ok)
	}
	if code, ok := apierror.GetCardModelError(note.Missing, format.TypeDeck); !ok || code != api.ErrorCodeNotFound {
		t.Fatalf("unexpected GetCardModelError(Missing, TypeDeck): %v, %v", code, ok)
	}
}

func TestDescribeCreateReturnsTheReasonByCode(t *testing.T) {
	if got := apierror.DescribeCreate(api.ErrorCodeInvalidName); got != "a note cannot be called that" {
		t.Fatalf("unexpected invalid name error: %q", got)
	}
	if got := apierror.DescribeCreate(api.ErrorCodeNotFound); got != "the file could not be made: not_found" {
		t.Fatalf("unexpected missing error: %q", got)
	}
	if got := apierror.DescribeCreate(api.ErrorCode("CUSTOM")); got != "the file could not be made: CUSTOM" {
		t.Fatalf("unexpected custom error: %q", got)
	}
}

func TestConvertOutcomeIsSilentForAReadThatSucceeded(t *testing.T) {
	if code, ok := apierror.ConvertOutcome(edit.Ok); ok || code != api.ErrorCodeUnspecified {
		t.Fatalf("unexpected ConvertOutcome(Ok): %v, %v", code, ok)
	}
	if code, ok := apierror.ConvertOutcome(edit.ReadOutcome("unknown")); ok || code != api.ErrorCodeUnspecified {
		t.Fatalf("unexpected ConvertOutcome(unknown): %v, %v", code, ok)
	}
	if code, ok := apierror.ConvertOutcome(edit.Missing); !ok || code != api.ErrorCodeNotFound {
		t.Fatalf("unexpected ConvertOutcome(Missing): %v, %v", code, ok)
	}
}

func TestGetErrorCodeReturnsFalseForAnUnmappedFailure(t *testing.T) {
	if code, ok := apierror.GetErrorCode(errors.New("unmapped")); ok || code != api.ErrorCodeUnspecified {
		t.Fatalf("unexpected GetErrorCode(unmapped): %v, %v", code, ok)
	}
	if code, ok := apierror.GetErrorCode(ports.ErrStale); !ok || code != api.ErrorCodeConflict {
		t.Fatalf("unexpected GetErrorCode(stale): %v, %v", code, ok)
	}
}

func TestIsIndexOutdatedIsTrueOnlyForTheIndexFailure(t *testing.T) {
	if apierror.IsIndexOutdated(nil) || !apierror.IsIndexOutdated(edit.ErrIndexOutdated) {
		t.Fatal("unexpected IsIndexOutdated results")
	}
}

func TestClassifyCreateReadsEachEndOfTheScenario(t *testing.T) {
	for _, one := range []struct {
		what     string
		made     cards.CreateNoteResult
		err      error
		path     string
		code     api.ErrorCode
		outdated bool
		failure  server.ErrorCode
	}{
		{what: "made", made: cards.CreateNoteResult{Path: "Deck.md"}, path: "Deck.md"},
		{what: "index behind", made: cards.CreateNoteResult{Path: "Deck.md"}, err: note.ErrIndexOutdated, path: "Deck.md", outdated: true},
		{what: "no fields", err: cards.ErrNoFields, failure: server.CodeInvalidArgument},
		{what: "name taken", err: ports.ErrAlreadyExists, code: api.ErrorCodeAlreadyExists},
		{what: "disk", err: errors.New("disk boom"), failure: server.CodeInternal},
	} {
		made, code, outdated, err := apierror.ClassifyCreate(one.made, one.err)
		failure, isFailure := errors.AsType[*server.APIError](err)
		if made.Path != one.path || code != one.code || outdated != one.outdated {
			t.Errorf("%s: got %+v, %v, %v", one.what, made, code, outdated)
		}
		if isFailure != (one.failure != "") || (isFailure && failure.Code != one.failure) {
			t.Errorf("%s: failed with %v, want %q", one.what, err, one.failure)
		}
	}
}
