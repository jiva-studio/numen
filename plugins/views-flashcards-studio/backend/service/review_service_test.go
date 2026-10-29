package service

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// answerCard answers one card of a session and returns the line recorded for
// it.
func answerCard(t *testing.T, handler *API, vault string, session *api.StartSessionResponse) string {
	t.Helper()
	if len(session.Cards) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := session.Cards[0]
	out, err := handler.AnswerCard(t.Context(), &api.AnswerCardRequest{
		Vault: vault, Run: session.Run,
		CardID: card.CardID, Template: card.Template,
		Rating: api.RatingGood,
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.Answer
}

// A card that has been answered before is returned with its current schedule
// state: it has been seen, and it has a due instant the page can parse. A card
// never answered carries neither.
func TestStartSession_ReviewedCardHasScheduleState(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]

	first := startSession(t, handler, v)
	for _, card := range first.Cards {
		if !card.IsNew {
			t.Errorf("a card nobody answered says it is not new: %+v", card)
		}
		if card.Due != "" {
			t.Errorf("a card nobody answered is due at %q", card.Due)
		}
	}
	answerCard(t, handler, string(v.ID), first)

	// Answered well, so the card is due in minutes and is presented again in this
	// session: the card carries the state from that answer.
	next := startSession(t, handler, v)
	if len(next.Cards) == 0 {
		t.Fatal("the card answered a moment ago is not asked again")
	}
	card := next.Cards[0]
	if card.IsNew {
		t.Error("a card already answered says it was not seen")
	}
	if _, err := time.Parse(review.TimestampLayout, card.Due); err != nil {
		t.Errorf("the card is due at %q, which the page cannot read: %v", card.Due, err)
	}
}

// A mistaken answer is withdrawn, and a withdrawn answer is not part of the
// answered count.
func TestAnAnswerTakenBackIsNotCounted(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]

	session := startSession(t, handler, v)
	provided := answerCard(t, handler, string(v.ID), session)

	if _, err := handler.TakeBackAnswer(t.Context(), &api.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: session.Run, Answer: provided,
	}); err != nil {
		t.Fatal(err)
	}

	out, err := handler.ListReviewDays(t.Context(),
		&api.ListReviewDaysRequest{Vault: string(v.ID)})
	if err != nil {
		t.Fatal(err)
	}
	if out.Reviewed != 0 {
		t.Errorf("the vault holds %d answers, want none", out.Reviewed)
	}
	if len(out.Days) != 0 {
		t.Errorf("answered on %d days, want none", len(out.Days))
	}
}

// one.
func TestTakingBackWithoutNamingAnAnswerIsRefused(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]
	session := startSession(t, handler, v)

	_, err := handler.TakeBackAnswer(t.Context(), &api.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: session.Run,
	})
	if codeOf(err) != server.CodeInvalidArgument {
		t.Errorf("failed with %v", codeOf(err))
	}
}

// An answer is taken back on the run it was given on. A run this window never
// opened is not one to write into.
func TestTakingBackOnARunNobodyOpenedIsRefused(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]
	session := startSession(t, handler, v)
	provided := answerCard(t, handler, string(v.ID), session)

	_, err := handler.TakeBackAnswer(t.Context(), &api.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: "nothing", Answer: provided,
	})
	if codeOf(err) != server.CodeNotFound {
		t.Errorf("failed with %v", codeOf(err))
	}
}

// An answer names the vault it belongs to. One the installation does not hold
// fails before a line is written into anybody's review.
func TestAnsweringAVaultNobodyContainsIsRefused(t *testing.T) {
	handler, _ := newAPI(t, deck)

	_, err := handler.AnswerCard(t.Context(), &api.AnswerCardRequest{
		Vault: "nothing", Run: "nothing",
		CardID: "k7m2xq9fzp", Template: "Recognise", Rating: api.RatingGood,
	})
	if codeOf(err) != server.CodeNotFound {
		t.Errorf("failed with %v", codeOf(err))
	}
}

// unreadable is a host whose list of vaults cannot be read.
type unreadable struct{ host.Vault }

func (unreadable) GetVaults(context.Context) ([]domain.Vault, error) {
	return nil, errors.New("the list of vaults cannot be read")
}

// A window that cannot read the installation's vaults reports the error. An empty
// list would read as an installation holding no vaults, which informs the user
// their vaults are gone.
func TestAWindowThatCannotReadTheVaultsReturnsSo(t *testing.T) {
	_, _, installation := newAPIOn(t)
	handler := &reviewHandler{Backend: New(Config{Vault: unreadable{Vault: installation.Host}, Store: installation.Host.Store()})}

	stream, err := newFlashcardsClient(t, handler).WatchCardsDue(t.Context(),
		&api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if stream.Receive() {
		t.Fatalf("a window that cannot read the vaults listed %+v", stream.Msg())
	}
	if codeOf(stream.Err()) != server.CodeInternal {
		t.Errorf("failed with %v", codeOf(stream.Err()))
	}
}

// A vault the installation does not hold fails before anything is written.
func TestTakingBackOnAVaultNobodyContainsIsRefused(t *testing.T) {
	handler, _ := newAPI(t, deck)

	_, err := handler.TakeBackAnswer(t.Context(), &api.TakeBackAnswerRequest{
		Vault: "nothing", Run: "nothing", Answer: "nothing",
	})
	if codeOf(err) != server.CodeNotFound {
		t.Errorf("failed with %v", codeOf(err))
	}
}

// A change under the window is delivered to the page, so a deck written while the list
// is displayed appears in the list.
func TestSomethingMovingReachesThePage(t *testing.T) {
	handler, _, installation := newAPIOn(t)
	client := newFlashcardsClient(t, handler)

	stream, err := client.WatchReloads(t.Context(), &api.WatchReloadsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	// The stream sends an initial message before any change.
	if !stream.Receive() {
		t.Fatalf("the stream said nothing: %v", stream.Err())
	}

	installation.Host.Move()

	response := make(chan bool, 1)
	go func() { response <- stream.Receive() }()
	select {
	case got := <-response:
		if !got {
			t.Fatalf("the stream ended: %v", stream.Err())
		}
		if !stream.Msg().Reload {
			t.Error("did not receive refresh signal")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("missing update notification for moved item")
	}
}

// A deck edited in place keeps its size and its stamp, and the window keeps no
// review card of it past a report that something moved.
func TestAReportedChangeForgetsTheCardsAlreadyParsed(t *testing.T) {
	handler, target, installation := newAPIOn(t, deck)
	v := target[0]
	reviewCards := handler.scenarios().flash.CardsDue.ReviewCards()

	before, err := reviewCards.Execute(t.Context(), v)
	if err != nil || len(before) != 1 {
		t.Fatalf("the vault stands as %d review cards and %v", len(before), err)
	}

	pos := filepath.Join(v.Path, "decks", "Words.md")
	info, err := os.Stat(pos)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(deck["decks/Words.md"], "### Word\n\nLeaf mould", "### Word\n\nMould leaf", 1)
	if err := os.WriteFile(pos, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(pos, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	stale, err := reviewCards.Execute(t.Context(), v)
	if err != nil || !isFront(stale[0], "Leaf mould") {
		t.Fatalf("the review cards were parsed again without a report: %+v %v", stale, err)
	}

	stream, err := newFlashcardsClient(t, handler).WatchReloads(t.Context(), &api.WatchReloadsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("the stream said nothing: %v", stream.Err())
	}
	installation.Host.Move()
	if !stream.Receive() || !stream.Msg().Reload {
		t.Fatalf("did not receive refresh signal: %v", stream.Err())
	}
	fresh, err := reviewCards.Execute(t.Context(), v)
	if err != nil || !isFront(fresh[0], "Mould leaf") {
		t.Errorf("the review cards stand as they were after a report: %+v %v", fresh, err)
	}
}

func isFront(f flashcards.ReviewCard, word string) bool {
	front, _ := f.Lay()
	return strings.Contains(front, word)
}

// unopenable is stores that open no vault.
type unopenable struct{}

func (unopenable) Open(domain.Vault) (ports.LogStore, error) {
	return nil, errNothingRead
}

// A session whose run cannot be opened is an internal error, and the page is
// handed no run to answer on.
func TestASessionWhoseRunCannotBeOpenedIsRefused(t *testing.T) {
	handler, target := newAPI(t, deck)
	handler.scenarios().flash.Log = flashcards.Log{Stores: unopenable{}}

	out, err := handler.StartSession(t.Context(),
		&api.StartSessionRequest{Vault: string(target[0].ID)})
	if codeOf(err) != server.CodeInternal {
		t.Errorf("failed with %v", codeOf(err))
	}
	if out != nil {
		t.Errorf("a session without a run came back with %+v", out)
	}
}

// A vault whose answers cannot be counted is an internal error and no list of
// days.
func TestAVaultWhoseAnswersCannotBeCountedIsRefused(t *testing.T) {
	handler, target := newAPI(t, deck)
	handler.scenarios().flash.GetReviewDays = flashcards.NewGetReviewDays(
		flashcards.NewCountReviews(unopenable{}, handler.scenarios().flash.Schedules, reviewDay, time.Now))

	out, err := handler.ListReviewDays(t.Context(),
		&api.ListReviewDaysRequest{Vault: string(target[0].ID)})
	if codeOf(err) != server.CodeInternal {
		t.Errorf("failed with %v", codeOf(err))
	}
	if out != nil {
		t.Errorf("days came back with %+v", out)
	}
}

// An answer for a vault that does not exist fails as an unknown vault, and not
// as an unopened run.
func TestAnAnswerAndATakingBackNameTheVaultNobodyContains(t *testing.T) {
	handler, _ := newAPI(t, deck)

	_, err := handler.AnswerCard(t.Context(), &api.AnswerCardRequest{
		Vault: "nothing", Run: "nothing", Rating: api.RatingGood,
	})
	if codeOf(err) != server.CodeNotFound || hasCause(err, ErrNoRun) {
		t.Errorf("answering failed with %v, want a vault nobody holds", err)
	}

	_, err = handler.TakeBackAnswer(t.Context(), &api.TakeBackAnswerRequest{
		Vault: "nothing", Run: "nothing", Answer: "nothing",
	})
	if codeOf(err) != server.CodeNotFound || hasCause(err, ErrNoRun) {
		t.Errorf("taking back failed with %v, want a vault nobody holds", err)
	}
}

// Taking an answer back answers with a response, which is empty.
func TestTakeBackAnswer_ReturnsEmptyResponse(t *testing.T) {
	handler, target := newAPI(t, deck)
	v := target[0]
	session := startSession(t, handler, v)
	answer := answerCard(t, handler, string(v.ID), session)

	out, err := handler.TakeBackAnswer(t.Context(), &api.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: session.Run, Answer: answer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
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
	if got.Again != 60 || got.Hard != 120 || got.Good != 180 || got.Easy != 240 {
		t.Errorf("the four came to %+v, want a minute to four, in seconds", got)
	}
}

// A range covers every day of a long history, answered and upcoming: the days
// outside it do not end the walk through the days inside it.
func TestListReviewDays_RangeCoversLongHistory(t *testing.T) {
	handler, target := newAPI(t, lived)
	v := target[0]
	runDays(t, handler, v, 14)

	list := func(from, to string) *api.ListReviewDaysResponse {
		out, err := handler.ListReviewDays(t.Context(), &api.ListReviewDaysRequest{
			Vault: string(v.ID), From: from, To: to,
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	names := func(days []*api.ReviewDay) []string {
		var out []string
		for _, one := range days {
			out = append(out, one.Day)
		}
		return out
	}

	all := list("", "")
	pastDays, ahead := names(all.Days), names(all.Due)
	if len(pastDays) < 6 || len(ahead) < 6 {
		t.Fatalf("the history holds %d days behind and %d ahead, want six of each", len(pastDays), len(ahead))
	}

	inside := list(pastDays[1], pastDays[len(pastDays)-2])
	if got, want := names(inside.Days), pastDays[1:len(pastDays)-1]; !slices.Equal(got, want) {
		t.Errorf("the days behind came back as %v, want %v", got, want)
	}

	coming := list(ahead[1], ahead[len(ahead)-2])
	if got, want := names(coming.Due), ahead[1:len(ahead)-1]; !slices.Equal(got, want) {
		t.Errorf("the days ahead came back as %v, want %v", got, want)
	}
}

// appendRefusing is a store whose file takes no answer.
type appendRefusing struct{ ports.LogStore }

func (appendRefusing) AppendLog(context.Context, string, []byte) error {
	return errNothingRead
}

type refusingStores struct{}

func (refusingStores) Open(domain.Vault) (ports.LogStore, error) { return appendRefusing{}, nil }

// An answer the run's file would not take is an internal error, and so is the
// taking back of one.
func TestAnAnswerTheRunCannotWriteIsRefused(t *testing.T) {
	handler, target := newAPI(t, deck)
	handler.scenarios().flash.Log = flashcards.Log{Stores: refusingStores{}}
	v := target[0]
	session := startSession(t, handler, v)
	card := session.Cards[0]

	out, err := handler.AnswerCard(t.Context(), &api.AnswerCardRequest{
		Vault: string(v.ID), Run: session.Run,
		CardID: card.CardID, Template: card.Template, Rating: api.RatingGood,
	})
	if codeOf(err) != server.CodeInternal || out != nil {
		t.Errorf("answering came back with %+v and %v, want an internal error", out, err)
	}

	back, err := handler.TakeBackAnswer(t.Context(), &api.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: session.Run, Answer: "01J8F3K2M9QRSTVWXYZ012",
	})
	if codeOf(err) != server.CodeInternal || back != nil {
		t.Errorf("taking back came back with %+v and %v, want an internal error", back, err)
	}
}

// A session that cannot be worked out is an internal error and opens no run.
func TestASessionThatCannotBeWorkedOutIsRefused(t *testing.T) {
	handler, target := newAPI(t, deck)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out, err := handler.StartSession(ctx, &api.StartSessionRequest{Vault: string(target[0].ID)})
	if codeOf(err) != server.CodeInternal || out != nil {
		t.Errorf("a session that could not be worked out came back with %+v and %v", out, err)
	}
}
