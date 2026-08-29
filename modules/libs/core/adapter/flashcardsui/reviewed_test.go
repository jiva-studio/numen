package flashcardsui

import (
	"testing"
	"time"

	"connectrpc.com/connect"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// counting is the day as the application counts one.
var counting = history.Day{Starts: history.DayStarts}

// What a vault was answered on comes back a day at a time, oldest first,
// because what draws it draws it along a line of time.
func TestWhatAVaultWasAnsweredOnComesBackInOrder(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]

	sitting := started(t, api, v)
	for _, card := range sitting.GetAsked() {
		if _, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
			VaultId: v.ID, Run: sitting.GetRun(),
			Card: card.GetCard(), Face: card.GetFace(),
			Rating: v1.Rating_RATING_GOOD,
		})); err != nil {
			t.Fatal(err)
		}
	}

	out, err := api.Reviewed(t.Context(), connect.NewRequest(&v1.ReviewedRequest{VaultId: v.ID}))
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

	out, err := api.Reviewed(t.Context(), connect.NewRequest(&v1.ReviewedRequest{VaultId: held[0].ID}))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Msg.GetDays()) != 0 || out.Msg.GetStreak() != 0 || out.Msg.GetAnswered() != 0 {
		t.Errorf("came back with %+v", out.Msg)
	}
}

// A question about a vault the installation does not hold is refused.
func TestReviewedIsRefusedForAVaultNobodyHolds(t *testing.T) {
	api, _ := windowed(t)

	_, err := api.Reviewed(t.Context(), connect.NewRequest(&v1.ReviewedRequest{VaultId: "nothing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
}
