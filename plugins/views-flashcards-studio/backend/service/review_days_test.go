package service

import (
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// counting is the day as the application counts one.
var counting = review.Day{Starts: review.DayStartTime}

// What a vault was answered on comes back a day at a time, oldest first,
// because what draws it draws it along a line of time.
func TestWhatAVaultWasAnsweredOnReachesBackInOrder(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]

	session := startSession(t, api, v)
	for _, card := range session.GetCards() {
		if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
			Vault: string(v.ID), Run: session.GetRun(),
			Mark: card.GetMark(), Face: card.GetFace(),
			Rating: v1.Rating_RATING_GOOD,
		})); err != nil {
			t.Fatal(err)
		}
	}

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(v.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	response := out.Msg
	if response.GetAnswered() != int32(len(session.GetCards())) {
		t.Errorf("the vault holds %d answers, want %d", response.GetAnswered(), len(session.GetCards()))
	}
	if response.GetStreak() != 1 {
		t.Errorf("the streak is %d on the first day, want 1", response.GetStreak())
	}
	if len(response.GetDays()) != 1 {
		t.Fatalf("answered on %d days, want the one", len(response.GetDays()))
	}
	if got := response.GetDays()[0]; got.GetDay() != counting.GetName(time.Now()) {
		t.Errorf("the day is %q, want %q", got.GetDay(), counting.GetName(time.Now()))
	}
}

// A vault nobody answered has nothing to draw, which is an answer and not a
// failure.
func TestAVaultNobodyAnsweredHasNothingToDraw(t *testing.T) {
	api, target := newAPI(t, deck)

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(target[0].ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Msg.GetDays()) != 0 || out.Msg.GetStreak() != 0 || out.Msg.GetAnswered() != 0 {
		t.Errorf("came back with %+v", out.Msg)
	}
}

// What is still to come is counted by the day it falls on, so the grid draws a
// week ahead as well as the weeks behind. A card answered easily is sent days
func TestWhatIsComingIsCountedByTheDayItFallsOn(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]

	session := startSession(t, api, v)
	if len(session.GetCards()) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := session.GetCards()[0]
	if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
		Vault: string(v.ID), Run: session.GetRun(),
		Mark: card.GetMark(), Face: card.GetFace(),
		Rating: v1.Rating_RATING_EASY,
	})); err != nil {
		t.Fatal(err)
	}

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(v.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	response := out.Msg

	if len(response.GetDue()) != 1 {
		t.Fatalf("%d days ahead hold anything, want the one", len(response.GetDue()))
	}
	coming := response.GetDue()[0]
	if coming.GetAnswered() != 1 {
		t.Errorf("%d cards fall on %s, want the one", coming.GetAnswered(), coming.GetDay())
	}
	// A day is written so that it sorts as text the way it sorts in time.
	if today := counting.GetName(time.Now()); coming.GetDay() <= today {
		t.Errorf("what is coming falls on %s, which is not after %s", coming.GetDay(), today)
	}
	// What was answered is still answered: the two are counted apart.
	if len(response.GetDays()) != 1 || response.GetDays()[0].GetEasy() != 1 {
		t.Errorf("the day behind came back as %+v", response.GetDays())
	}
}

// A vault whose cards are all still to come says so and does not fail: a person
// who answered everything this morning has an empty grid behind them and a full
// one ahead.
func TestNothingIsComingWhereNothingWasAnswered(t *testing.T) {
	api, target := newAPI(t, deck)

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(target[0].ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Msg.GetDue()) != 0 {
		t.Errorf("%d days ahead hold something, want none", len(out.Msg.GetDue()))
	}
}

// The days are drawn along a line of time, so they are handed over along one,
// whatever order they were counted in.
func TestTheDaysComeBackOldestFirst(t *testing.T) {
	response := sortReviewDays([]*v1.ReviewDay{
		{Day: "2026-08-30"},
		{Day: "2026-07-01"},
		{Day: "2027-01-02"},
		{Day: "2026-08-30"},
		{Day: "2026-08-09"},
	})

	want := []string{"2026-07-01", "2026-08-09", "2026-08-30", "2026-08-30", "2027-01-02"}
	for item, one := range response {
		if one.GetDay() != want[item] {
			t.Errorf("the day at %d is %s, want %s", item, one.GetDay(), want[item])
		}
	}
}

// A question about a vault the installation does not hold fails.
func TestAVaultNobodyContainsIsRefused(t *testing.T) {
	api, _ := newAPI(t)

	_, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: "nothing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
}

// The stretch reaches the answer itself: a day outside
// what was asked for is in neither list.
func TestAStretchNothingFallsInReachesBackEmpty(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]

	session := startSession(t, api, v)
	for _, card := range session.GetCards() {
		if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
			Vault: string(v.ID), Run: session.GetRun(),
			Mark: card.GetMark(), Face: card.GetFace(),
			Rating: v1.Rating_RATING_GOOD,
		})); err != nil {
			t.Fatal(err)
		}
	}

	out, err := api.ListReviewDays(t.Context(), connect.NewRequest(&v1.ListReviewDaysRequest{
		Vault: string(v.ID), From: "1970-01-01", To: "1970-12-31",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Msg.GetDays()) != 0 || len(out.Msg.GetDue()) != 0 {
		t.Errorf("a stretch in 1970 came back with %d days and %d ahead",
			len(out.Msg.GetDays()), len(out.Msg.GetDue()))
	}
	// The streak and the count are over the whole of the vault whatever stretch
	// was asked for.
	if out.Msg.GetAnswered() != int32(len(session.GetCards())) {
		t.Errorf("the vault holds %d answers, want %d",
			out.Msg.GetAnswered(), len(session.GetCards()))
	}
}
