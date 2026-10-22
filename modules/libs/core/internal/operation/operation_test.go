package operation_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/operation"
)

func TestAnOperationRecallsWhatItRemembered(t *testing.T) {
	ctx, end := operation.Begin(t.Context())
	defer end()

	operation.Remember(ctx, "a", "one", nil)
	if got, found := operation.Recall(ctx, "a"); !found || got != "one" {
		t.Errorf("recalled %q, %v; want one", got, found)
	}
	if _, found := operation.Recall(ctx, "b"); found {
		t.Error("recalled a key nothing was remembered under")
	}
}

func TestEndingAnOperationReleasesWhatItRemembered(t *testing.T) {
	ctx, end := operation.Begin(t.Context())
	released := 0
	operation.Remember(ctx, "a", "one", func() { released++ })
	end()
	end()

	if released != 1 {
		t.Errorf("released %d times, want once", released)
	}
	if _, found := operation.Recall(ctx, "a"); found {
		t.Error("recalled after the operation ended")
	}
}

func TestAnOperationBegunInsideAnotherJoinsIt(t *testing.T) {
	outer, endOuter := operation.Begin(t.Context())
	released := 0
	inner, endInner := operation.Begin(outer)
	operation.Remember(inner, "a", "one", func() { released++ })
	endInner()

	if released != 0 {
		t.Fatal("the inner end released what the outer operation holds")
	}
	if got, _ := operation.Recall(outer, "a"); got != "one" {
		t.Errorf("recalled %q in the outer operation, want one", got)
	}
	endOuter()
	if released != 1 {
		t.Errorf("released %d times, want once", released)
	}
}

func TestNothingIsCachedOutsideAnOperation(t *testing.T) {
	released := 0
	operation.Remember(t.Context(), "a", "one", func() { released++ })

	if released != 1 {
		t.Errorf("released %d times, want at once", released)
	}
	if _, found := operation.Recall(t.Context(), "a"); found {
		t.Error("recalled outside an operation")
	}
}

func TestRememberingAgainReleasesThePreviousValue(t *testing.T) {
	ctx, end := operation.Begin(t.Context())
	defer end()
	released := 0
	operation.Remember(ctx, "a", "one", func() { released++ })
	operation.Remember(ctx, "a", "two", nil)

	if released != 1 {
		t.Errorf("released %d times, want once", released)
	}
	if got, _ := operation.Recall(ctx, "a"); got != "two" {
		t.Errorf("recalled %q, want two", got)
	}
}

func TestRememberingAgainOverAValueCachedWithNothingToReleaseIsCached(t *testing.T) {
	ctx, end := operation.Begin(t.Context())
	defer end()

	operation.Remember(ctx, "a", "one", nil)
	released := 0
	operation.Remember(ctx, "a", "two", func() { released++ })

	if got, _ := operation.Recall(ctx, "a"); got != "two" {
		t.Errorf("recalled %q, want two", got)
	}
	if released != 0 {
		t.Errorf("released %d times before the operation ended", released)
	}
}

func TestAnOperationIsActiveOnlyWithinIt(t *testing.T) {
	if operation.IsActive(t.Context()) {
		t.Error("a context under no operation is active")
	}
	ctx, end := operation.Begin(t.Context())
	defer end()
	if !operation.IsActive(ctx) {
		t.Error("a context under an operation is not active")
	}
}
