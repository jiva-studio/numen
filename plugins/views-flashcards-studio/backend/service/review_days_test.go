package service

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// counting is the day as the application counts one.
var counting = review.Day{Starts: review.DayStartTime}

// What a vault was answered on comes back a day at a time, oldest first,
// because what draws it draws it along a line of time.
func TestListReviewDays_ReturnsDaysOldestFirst(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]

	session := startSession(t, handler, v)
	for _, card := range session.Cards {
		if _, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
			Vault: string(v.ID), Run: session.Run,
			CardID: card.CardID, Template: card.Template,
			Rating: api.RatingGood,
		}); err != nil {
			t.Fatal(err)
		}
	}

	out, err := handler.ListReviewDays(t.Context(),
		&api.ListReviewDaysRequest{Vault: string(v.ID)})
	if err != nil {
		t.Fatal(err)
	}
	response := out
	if response.Reviewed != int32(len(session.Cards)) {
		t.Errorf("the vault holds %d answers, want %d", response.Reviewed, len(session.Cards))
	}
	if response.Streak != 1 {
		t.Errorf("the streak is %d on the first day, want 1", response.Streak)
	}
	if len(response.Days) != 1 {
		t.Fatalf("answered on %d days, want the one", len(response.Days))
	}
	if got := response.Days[0]; got.Day != counting.GetName(time.Now()) {
		t.Errorf("the day is %q, want %q", got.Day, counting.GetName(time.Now()))
	}
}

// A vault with no answers has no days to draw; this is an empty result and not an
// error.
func TestListReviewDays_EmptyVaultHasNoDays(t *testing.T) {
	handler, target := newAPI(t, deck)

	out, err := handler.ListReviewDays(t.Context(),
		&api.ListReviewDaysRequest{Vault: string(target[0].ID)})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Days) != 0 || out.Streak != 0 || out.Reviewed != 0 {
		t.Errorf("came back with %+v", out)
	}
}

// What is still to come is counted by the day it falls on, so the grid draws a
// week ahead as well as the weeks behind. A card answered easily is sent days
func TestWhatIsComingIsCountedByTheDayItFallsOn(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]

	session := startSession(t, handler, v)
	if len(session.Cards) == 0 {
		t.Fatal("the vault has no cards due")
	}
	card := session.Cards[0]
	if _, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
		Vault: string(v.ID), Run: session.Run,
		CardID: card.CardID, Template: card.Template,
		Rating: api.RatingEasy,
	}); err != nil {
		t.Fatal(err)
	}

	out, err := handler.ListReviewDays(t.Context(),
		&api.ListReviewDaysRequest{Vault: string(v.ID)})
	if err != nil {
		t.Fatal(err)
	}
	response := out

	if len(response.Due) != 1 {
		t.Fatalf("%d days ahead hold anything, want the one", len(response.Due))
	}
	coming := response.Due[0]
	if coming.Reviewed != 1 {
		t.Errorf("%d cards fall on %s, want the one", coming.Reviewed, coming.Day)
	}
	// A day is formatted to sort as text in the same order as in time.
	if today := counting.GetName(time.Now()); coming.Day <= today {
		t.Errorf("what is coming falls on %s, which is not after %s", coming.Day, today)
	}
	// What was answered is still answered: the two are counted apart.
	if len(response.Days) != 1 || response.Days[0].Easy != 1 {
		t.Errorf("the day behind came back as %+v", response.Days)
	}
}

// A vault whose cards are all due in the future returns empty past days and does
// not fail: a user who answered everything this morning has an empty past grid
// and a full upcoming one.
func TestListReviewDays_EmptyVaultHasNoDueDays(t *testing.T) {
	handler, target := newAPI(t, deck)

	out, err := handler.ListReviewDays(t.Context(),
		&api.ListReviewDaysRequest{Vault: string(target[0].ID)})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Due) != 0 {
		t.Errorf("%d days ahead hold something, want none", len(out.Due))
	}
}

// The days are drawn along a line of time, so they are handed over along one,
// whatever order they were counted in.
func TestTheDaysComeBackOldestFirst(t *testing.T) {
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

// A question about a vault the installation does not hold fails.
func TestAVaultNobodyContainsIsRefused(t *testing.T) {
	handler, _ := newAPI(t)

	_, err := handler.ListReviewDays(t.Context(),
		&api.ListReviewDaysRequest{Vault: "nothing"})
	if codeOf(err) != server.CodeNotFound {
		t.Errorf("failed with %v", codeOf(err))
	}
}

// A stretch containing no answers returns empty lists: a day outside the
// requested range is in neither list.
func TestAStretchNothingFallsInReachesBackEmpty(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]

	session := startSession(t, handler, v)
	for _, card := range session.Cards {
		if _, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
			Vault: string(v.ID), Run: session.Run,
			CardID: card.CardID, Template: card.Template,
			Rating: api.RatingGood,
		}); err != nil {
			t.Fatal(err)
		}
	}

	out, err := handler.ListReviewDays(t.Context(), &api.ListReviewDaysRequest{
		Vault: string(v.ID), From: "1970-01-01", To: "1970-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Days) != 0 || len(out.Due) != 0 {
		t.Errorf("a stretch in 1970 came back with %d days and %d ahead",
			len(out.Days), len(out.Due))
	}
	// The streak and the count cover the whole vault regardless of the requested
	// stretch.
	if out.Reviewed != int32(len(session.Cards)) {
		t.Errorf("the vault holds %d answers, want %d",
			out.Reviewed, len(session.Cards))
	}
}
