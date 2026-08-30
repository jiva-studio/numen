package flashcardsui

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// answered is one card of a sitting answered, and the line that stands for it.
func answered(t *testing.T, api *API, vault string, sitting *v1.StartResponse) string {
	t.Helper()
	if len(sitting.GetAsked()) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := sitting.GetAsked()[0]
	out, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
		VaultId: vault, Run: sitting.GetRun(),
		Card: card.GetCard(), Face: card.GetFace(),
		Rating: v1.Rating_RATING_GOOD,
	}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg.GetAnswer()
}

// A person hits the wrong key and takes it back, and what they took back is
// not part of what they answered.
func TestAnAnswerTakenBackIsNotCounted(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]

	sitting := started(t, api, v)
	given := answered(t, api, v.ID, sitting)

	if _, err := api.TakeBack(t.Context(), connect.NewRequest(&v1.TakeBackRequest{
		VaultId: v.ID, Run: sitting.GetRun(), Answer: given,
	})); err != nil {
		t.Fatal(err)
	}

	out, err := api.Reviewed(t.Context(), connect.NewRequest(&v1.ReviewedRequest{VaultId: v.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetAnswered() != 0 {
		t.Errorf("the vault holds %d answers, want none", out.Msg.GetAnswered())
	}
	if len(out.Msg.GetDays()) != 0 {
		t.Errorf("answered on %d days, want none", len(out.Msg.GetDays()))
	}
}

// Taking back an answer nobody named is the caller's mistake, and is refused as
// one rather than as trouble with the file.
func TestTakingBackWithoutNamingAnAnswerIsRefused(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]
	sitting := started(t, api, v)

	_, err := api.TakeBack(t.Context(), connect.NewRequest(&v1.TakeBackRequest{
		VaultId: v.ID, Run: sitting.GetRun(),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
}

// An answer is taken back on the run it was given on. A run this window never
// opened is not one to write into.
func TestTakingBackOnARunNobodyOpenedIsRefused(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]
	sitting := started(t, api, v)
	given := answered(t, api, v.ID, sitting)

	_, err := api.TakeBack(t.Context(), connect.NewRequest(&v1.TakeBackRequest{
		VaultId: v.ID, Run: "nothing", Answer: given,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
}

// A vault the installation does not hold is refused before anything is written.
func TestTakingBackOnAVaultNobodyHoldsIsRefused(t *testing.T) {
	api, _ := windowed(t, deck)

	_, err := api.TakeBack(t.Context(), connect.NewRequest(&v1.TakeBackRequest{
		VaultId: "nothing", Run: "nothing", Answer: "nothing",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
}
