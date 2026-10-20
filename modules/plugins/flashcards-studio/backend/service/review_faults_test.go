package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/ports"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/review"
)

// unopenable is stores that open no vault.
type unopenable struct{}

func (unopenable) Open(domain.Vault) (ports.DerivedStore, error) {
	return nil, errNothingRead
}

// A session whose run cannot be opened is an internal error, and the page is
// handed no run to answer on.
func TestASessionWhoseRunCannotBeOpenedIsRefused(t *testing.T) {
	api, held := newAPI(t, deck)
	api.scenarios().flash.Log = flashcards.Log{Stores: unopenable{}}

	out, err := api.StartSession(t.Context(),
		connect.NewRequest(&v1.StartSessionRequest{Vault: string(held[0].ID)}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
	if out != nil {
		t.Errorf("a session without a run came back with %+v", out.Msg)
	}
}

// A vault whose answers cannot be counted is an internal error and no list of
// days.
func TestAVaultWhoseAnswersCannotBeCountedIsRefused(t *testing.T) {
	api, held := newAPI(t, deck)
	api.scenarios().flash.GetReviewDays = flashcards.NewGetReviewDays(
		flashcards.NewCountReviews(unopenable{}, api.scenarios().flash.Schedules, reviewDay, time.Now))

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(held[0].ID)}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("refused with %v", connect.CodeOf(err))
	}
	if out != nil {
		t.Errorf("days came back with %+v", out.Msg)
	}
}

// An answer for a vault nobody holds is refused as that, and not as a run nobody
// opened.
func TestAnAnswerAndATakingBackNameTheVaultNobodyHolds(t *testing.T) {
	api, _ := newAPI(t, deck)

	_, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
		Vault: "nothing", Run: "nothing", Rating: v1.Rating_RATING_GOOD,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound || errors.Is(err, ErrNoRun) {
		t.Errorf("answering was refused with %v, want a vault nobody holds", err)
	}

	_, err = api.TakeBackAnswer(t.Context(), connect.NewRequest(&v1.TakeBackAnswerRequest{
		Vault: "nothing", Run: "nothing", Answer: "nothing",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound || errors.Is(err, ErrNoRun) {
		t.Errorf("taking back was refused with %v, want a vault nobody holds", err)
	}
}

// Taking an answer back answers with a response, which is empty.
func TestTakingAnAnswerBackIsAnsweredWithAResponse(t *testing.T) {
	api, held := newAPI(t, deck)
	v := held[0]
	session := startSession(t, api, v)
	answer := answerCard(t, api, string(v.ID), session)

	out, err := api.TakeBackAnswer(t.Context(), connect.NewRequest(&v1.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: session.GetRun(), Answer: answer,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.Msg == nil {
		t.Errorf("taking an answer back came back with %+v", out)
	}
}

// A card with no projection of what each answer would do carries none.
func TestACardWithNoProjectionCarriesNone(t *testing.T) {
	if got := newWireIntervals(nil); got != nil {
		t.Errorf("a card with nothing ahead of it carries %+v", got)
	}

	got := newWireIntervals(map[review.Rating]time.Duration{
		review.Again: time.Minute, review.Hard: 2 * time.Minute,
		review.Good: 3 * time.Minute, review.Easy: 4 * time.Minute,
	})
	if got.GetAgain() != 60 || got.GetHard() != 120 || got.GetGood() != 180 || got.GetEasy() != 240 {
		t.Errorf("the four came to %+v, want a minute to four, in seconds", got)
	}
}

// A stretch reaches every day of a long history, answered and still to come:
// the days outside it do not end the walk through the days inside it.
func TestAStretchOfALongHistoryHoldsEveryDayInsideIt(t *testing.T) {
	api, held := newAPI(t, lived)
	v := held[0]
	runDays(t, api, v, 14)

	list := func(from, to string) *v1.ListReviewDaysResponse {
		out, err := api.ListReviewDays(t.Context(), connect.NewRequest(&v1.ListReviewDaysRequest{
			Vault: string(v.ID), From: from, To: to,
		}))
		if err != nil {
			t.Fatal(err)
		}
		return out.Msg
	}
	names := func(days []*v1.ReviewDay) []string {
		var out []string
		for _, one := range days {
			out = append(out, one.GetDay())
		}
		return out
	}

	all := list("", "")
	behind, ahead := names(all.GetDays()), names(all.GetDue())
	if len(behind) < 6 || len(ahead) < 6 {
		t.Fatalf("the history holds %d days behind and %d ahead, want six of each", len(behind), len(ahead))
	}

	inside := list(behind[1], behind[len(behind)-2])
	if got, want := names(inside.GetDays()), behind[1:len(behind)-1]; !slices.Equal(got, want) {
		t.Errorf("the days behind came back as %v, want %v", got, want)
	}

	coming := list(ahead[1], ahead[len(ahead)-2])
	if got, want := names(coming.GetDue()), ahead[1:len(ahead)-1]; !slices.Equal(got, want) {
		t.Errorf("the days ahead came back as %v, want %v", got, want)
	}
}

// appendRefusing is a store whose file takes no answer.
type appendRefusing struct{ ports.DerivedStore }

func (appendRefusing) AppendArtifact(context.Context, domain.ArtifactName, []byte) error {
	return errNothingRead
}

type refusingStores struct{}

func (refusingStores) Open(domain.Vault) (ports.DerivedStore, error) { return appendRefusing{}, nil }

// An answer the run's file would not take is an internal error, and so is the
// taking back of one.
func TestAnAnswerTheRunCannotWriteIsRefused(t *testing.T) {
	api, held := newAPI(t, deck)
	api.scenarios().flash.Log = flashcards.Log{Stores: refusingStores{}}
	v := held[0]
	session := startSession(t, api, v)
	card := session.GetCards()[0]

	out, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
		Vault: string(v.ID), Run: session.GetRun(),
		Mark: card.GetMark(), Face: card.GetFace(), Rating: v1.Rating_RATING_GOOD,
	}))
	if connect.CodeOf(err) != connect.CodeInternal || out != nil {
		t.Errorf("answering came back with %+v and %v, want an internal error", out, err)
	}

	back, err := api.TakeBackAnswer(t.Context(), connect.NewRequest(&v1.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: session.GetRun(), Answer: "01J8F3K2M9QRSTVWXYZ012",
	}))
	if connect.CodeOf(err) != connect.CodeInternal || back != nil {
		t.Errorf("taking back came back with %+v and %v, want an internal error", back, err)
	}
}

// A session that cannot be worked out is an internal error and opens no run.
func TestASessionThatCannotBeWorkedOutIsRefused(t *testing.T) {
	api, held := newAPI(t, deck)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out, err := api.StartSession(ctx, connect.NewRequest(&v1.StartSessionRequest{Vault: string(held[0].ID)}))
	if connect.CodeOf(err) != connect.CodeInternal || out != nil {
		t.Errorf("a session that could not be worked out came back with %+v and %v", out, err)
	}
}
