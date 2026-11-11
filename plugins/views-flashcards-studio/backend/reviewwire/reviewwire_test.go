package reviewwire

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/enumtest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/reloads"
)

// The four ratings are carried across as themselves, and nothing else is a rating.
func TestTheFourRatingsAreCarriedAcross(t *testing.T) {
	for response, want := range map[api.Rating]review.Rating{
		api.RatingAgain: review.Again,
		api.RatingHard:  review.Hard,
		api.RatingGood:  review.Good,
		api.RatingEasy:  review.Easy,
	} {
		if got := ParseRating(response); got != want {
			t.Errorf("%v came across as %v", response, got)
		}
	}
	if got := ParseRating(api.RatingUnspecified); got.Valid() {
		t.Errorf("nothing said came across as %v", got)
	}
	if got := ParseRating(api.Rating("unknown")); got != 0 {
		t.Errorf("expected 0 for unknown rating, got %v", got)
	}
}

// A rating the parser does not recognize is parsed as zero, which the use case rejects; a button
// press would be reported as not an answer.
func TestEveryRatingIsRead(t *testing.T) {
	enumtest.CheckHandled(t, api.GetRatingValues(), func(r api.Rating) bool { return ParseRating(r) != 0 })
}

// A card with no projection of what each answer would do carries none.
func TestACardWithNoProjectionCarriesNone(t *testing.T) {
	if got := newIntervals(nil); got != nil {
		t.Errorf("a card with nothing ahead of it carries %+v", got)
	}

	got := newIntervals(map[review.Rating]time.Duration{
		review.Again: time.Minute, review.Hard: 2 * time.Minute,
		review.Good: 3 * time.Minute, review.Easy: 4 * time.Minute,
	})
	if got.Again != 60 || got.Hard != 120 || got.Good != 180 || got.Easy != 240 {
		t.Errorf("the four came to %+v, want a minute to four, in seconds", got)
	}
}

// The days are drawn along a line of time, so they are passed along one, whatever order they were
// counted in.
func TestTheDaysAreReturnedOldestFirst(t *testing.T) {
	response := sortReviewDays([]*api.ReviewDay{
		{Day: "2026-08-30"},
		{Day: "2026-07-01"},
		{Day: "2027-01-02"},
		{Day: "2026-08-30"},
		{Day: "2026-08-09"},
	})

	want := []string{"2026-07-01", "2026-08-09", "2026-08-30", "2026-08-30", "2027-01-02"}
	for item, one := range response {
		if one.Day != want[item] {
			t.Errorf("the day at %d is %s, want %s", item, one.Day, want[item])
		}
	}
}

func TestAWatchThatCannotFollowTheChangesIsUnavailable(t *testing.T) {
	err := NewWatchError(&reloads.UnavailableError{Err: errors.New("watch boom")})
	if failure, ok := errors.AsType[*server.APIError](err); !ok || failure.Code != server.CodeUnavailable {
		t.Fatalf("expected an unavailable API error, got %v", err)
	}
	failure := errors.New("write failed")
	if got := NewWatchError(failure); !errors.Is(got, failure) {
		t.Fatalf("a listener that has failure is returned as it came, got %v", got)
	}
	if got := NewWatchError(nil); got != nil {
		t.Fatalf("expected nil for a watch that ended cleanly, got %v", got)
	}
}
