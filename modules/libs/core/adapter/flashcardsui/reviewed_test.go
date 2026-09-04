package flashcardsui

import (
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// counting is the day as the application counts one.
var counting = review.Day{Starts: review.DayStarts}

// What a vault was answered on comes back a day at a time, oldest first,
// because what draws it draws it along a line of time.
func TestWhatAVaultWasAnsweredOnComesBackInOrder(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]

	sitting := started(t, api, v)
	for _, card := range sitting.GetAsked() {
		if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
			Vault: string(v.ID), Run: sitting.GetRun(),
			Card: card.GetCard(), Face: card.GetFace(),
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
	said := out.Msg
	if said.GetAnswered() != int32(len(sitting.GetAsked())) {
		t.Errorf("the vault holds %d answers, want %d", said.GetAnswered(), len(sitting.GetAsked()))
	}
	if said.GetStreak() != 1 {
		t.Errorf("the streak is %d on the first day, want 1", said.GetStreak())
	}
	if len(said.GetDays()) != 1 {
		t.Fatalf("answered on %d days, want the one", len(said.GetDays()))
	}
	if got := said.GetDays()[0]; got.GetDay() != counting.Names(time.Now()) {
		t.Errorf("the day is %q, want %q", got.GetDay(), counting.Names(time.Now()))
	}
}

// A vault nobody answered has nothing to draw, which is an answer and not a
// failure.
func TestAVaultNobodyAnsweredHasNothingToDraw(t *testing.T) {
	api, held := windowed(t, deck)

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(held[0].ID)}))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Msg.GetDays()) != 0 || out.Msg.GetStreak() != 0 || out.Msg.GetAnswered() != 0 {
		t.Errorf("came back with %+v", out.Msg)
	}
}

// What is still to come is counted by the day it falls on, so the grid draws a
// week ahead as well as the weeks behind. A card answered easily is sent days
// away, and the day it is sent to is a day nobody has answered on.
func TestWhatIsComingIsCountedByTheDayItFallsOn(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]

	sitting := started(t, api, v)
	if len(sitting.GetAsked()) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := sitting.GetAsked()[0]
	if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
		Vault: string(v.ID), Run: sitting.GetRun(),
		Card: card.GetCard(), Face: card.GetFace(),
		Rating: v1.Rating_RATING_EASY,
	})); err != nil {
		t.Fatal(err)
	}

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(v.ID)}))
	if err != nil {
		t.Fatal(err)
	}
	said := out.Msg

	if len(said.GetDue()) != 1 {
		t.Fatalf("%d days ahead hold anything, want the one", len(said.GetDue()))
	}
	coming := said.GetDue()[0]
	if coming.GetAnswered() != 1 {
		t.Errorf("%d cards fall on %s, want the one", coming.GetAnswered(), coming.GetDay())
	}
	// A day is written so that it sorts as text the way it sorts in time, which
	// is what says this day is ahead of today and not behind it.
	if today := counting.Names(time.Now()); coming.GetDay() <= today {
		t.Errorf("what is coming falls on %s, which is not after %s", coming.GetDay(), today)
	}
	// What was answered is still answered: the two are counted apart.
	if len(said.GetDays()) != 1 || said.GetDays()[0].GetEasy() != 1 {
		t.Errorf("the day behind came back as %+v", said.GetDays())
	}
}

// A vault whose cards are all still to come says so and does not fail: a person
// who answered everything this morning has an empty grid behind them and a full
// one ahead.
func TestNothingIsComingWhereNothingWasAnswered(t *testing.T) {
	api, held := windowed(t, deck)

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(held[0].ID)}))
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
	said := inOrder([]*v1.Reviewing{
		{Day: "2026-08-30"},
		{Day: "2026-07-01"},
		{Day: "2027-01-02"},
		{Day: "2026-08-30"},
		{Day: "2026-08-09"},
	})

	want := []string{"2026-07-01", "2026-08-09", "2026-08-30", "2026-08-30", "2027-01-02"}
	for at, one := range said {
		if one.GetDay() != want[at] {
			t.Errorf("the day at %d is %s, want %s", at, one.GetDay(), want[at])
		}
	}
}

// A question about a vault the installation does not hold is refused.
func TestReviewedIsRefusedForAVaultNobodyHolds(t *testing.T) {
	api, _ := windowed(t)

	_, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: "nothing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
}
