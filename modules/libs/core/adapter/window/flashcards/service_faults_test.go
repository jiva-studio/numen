package flashcards

import (
	"errors"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// unopenable is stores that open no vault.
type unopenable struct{}

func (unopenable) Open(domain.Vault) (port.DerivedStore, error) {
	return nil, errNothingRead
}

// A session whose run cannot be opened is an internal error, and the page is
// handed no run to answer on.
func TestASessionWhoseRunCannotBeOpenedIsRefused(t *testing.T) {
	api, held := newAPI(t, deck)
	api.Log = flashcards.Log{Stores: unopenable{}}

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
	api.Counted = flashcards.NewCountReviews(unopenable{}, api.Schedules, reviewDay, time.Now)

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
	if got := newAhead(nil); got != nil {
		t.Errorf("a card with nothing ahead of it carries %+v", got)
	}

	got := newAhead(map[review.Rating]time.Duration{
		review.Again: time.Minute, review.Hard: 2 * time.Minute,
		review.Good: 3 * time.Minute, review.Easy: 4 * time.Minute,
	})
	if got.GetAgain() != 60 || got.GetHard() != 120 || got.GetGood() != 180 || got.GetEasy() != 240 {
		t.Errorf("the four came to %+v, want a minute to four, in seconds", got)
	}
}

// Two days are ordered as text is, and the same day is neither before nor
// after itself.
func TestTwoDaysAreOrderedAsTheyFallInTime(t *testing.T) {
	for name, one := range map[string]struct {
		a, b string
		want int
	}{
		"before": {"2026-08-09", "2026-08-30", -1},
		"after":  {"2026-08-30", "2026-08-09", 1},
		"same":   {"2026-08-09", "2026-08-09", 0},
		"empty":  {"", "2026-08-09", -1},
	} {
		t.Run(name, func(t *testing.T) {
			if got := cmpDay(one.a, one.b); got != one.want {
				t.Errorf("cmpDay(%q, %q) = %d, want %d", one.a, one.b, got, one.want)
			}
		})
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
