package flashcardsui

import (
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
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

// A card that has been answered before comes back carrying where it stands: it
// has been seen, and it is due at an instant the page can read. A card nobody
// has answered carries neither.
func TestACardAlreadyAnsweredComesBackWithWhereItStands(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]

	first := started(t, api, v)
	for _, card := range first.GetAsked() {
		if card.GetSeen() {
			t.Errorf("a card nobody answered says it was seen: %+v", card)
		}
		if card.GetDue() != "" {
			t.Errorf("a card nobody answered is due at %q", card.GetDue())
		}
	}
	answered(t, api, string(v.ID), first)

	// Answered well, so the card is minutes away and asked again in this
	// sitting: what it carries is where the answer left it.
	next := started(t, api, v)
	if len(next.GetAsked()) == 0 {
		t.Fatal("the card answered a moment ago is not asked again")
	}
	card := next.GetAsked()[0]
	if !card.GetSeen() {
		t.Error("a card already answered says it was not seen")
	}
	if _, err := time.Parse(history.Stamp, card.GetDue()); err != nil {
		t.Errorf("the card is due at %q, which the page cannot read: %v", card.GetDue(), err)
	}
}

// A person hits the wrong key and takes it back, and what they took back is
// not part of what they answered.
func TestAnAnswerTakenBackIsNotCounted(t *testing.T) {
	api, held := windowed(t, deck)
	v := held[0]

	sitting := started(t, api, v)
	given := answered(t, api, string(v.ID), sitting)

	if _, err := api.TakeBack(t.Context(), connect.NewRequest(&v1.TakeBackRequest{
		VaultId: string(v.ID), Run: sitting.GetRun(), Answer: given,
	})); err != nil {
		t.Fatal(err)
	}

	out, err := api.Reviewed(t.Context(), connect.NewRequest(&v1.ReviewedRequest{VaultId: string(v.ID)}))
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
		VaultId: string(v.ID), Run: sitting.GetRun(),
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
	given := answered(t, api, string(v.ID), sitting)

	_, err := api.TakeBack(t.Context(), connect.NewRequest(&v1.TakeBackRequest{
		VaultId: string(v.ID), Run: "nothing", Answer: given,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
}

// An answer names the vault it belongs to. One the installation does not hold
// is refused before a line is written into anybody's history.
func TestAnsweringAVaultNobodyHoldsIsRefused(t *testing.T) {
	api, _ := windowed(t, deck)

	_, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
		VaultId: "nothing", Run: "nothing",
		Card: "k7m2xq9fzp", Face: "Recognise", Rating: v1.Rating_RATING_GOOD,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
}

// unreadable is an installation whose list of vaults cannot be read.
type unreadable struct{ registry }

func (unreadable) All() ([]domain.Vault, error) {
	return nil, errors.New("the list of vaults cannot be read")
}

// A window that cannot read the installation's vaults says so. An empty list
// would read as an installation holding none, which is a person told their
// vaults are gone.
func TestAWindowThatCannotReadTheVaultsSaysSo(t *testing.T) {
	api := &API{Registry: unreadable{}, Now: time.Now}

	stream, err := serving(t, api).Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if stream.Receive() {
		t.Fatalf("a window that cannot read the vaults listed %+v", stream.Msg())
	}
	if connect.CodeOf(stream.Err()) != connect.CodeInternal {
		t.Errorf("refused with %v", connect.CodeOf(stream.Err()))
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
