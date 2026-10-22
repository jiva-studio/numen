package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// answerCard is one card of a session answered, and the line that stands for
// it.
func answerCard(t *testing.T, api *API, vault string, session *v1.StartSessionResponse) string {
	t.Helper()
	if len(session.GetCards()) == 0 {
		t.Fatal("the vault owes nothing to answer")
	}
	card := session.GetCards()[0]
	out, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
		Vault: vault, Run: session.GetRun(),
		Mark: card.GetMark(), Face: card.GetFace(),
		Rating: v1.Rating_RATING_GOOD,
	}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg.GetAnswer()
}

// A card that has been answered before comes back carrying at its current position: it
// has been seen, and it is due at an instant the page can read. A card nobody
// has answered carries neither.
func TestACardAlreadyAnsweredReachesBackWithWhereItExists(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]

	first := startSession(t, api, v)
	for _, card := range first.GetCards() {
		if !card.GetIsNew() {
			t.Errorf("a card nobody answered says it is not new: %+v", card)
		}
		if card.GetDue() != "" {
			t.Errorf("a card nobody answered is due at %q", card.GetDue())
		}
	}
	answerCard(t, api, string(v.ID), first)

	// Answered well, so the card is minutes away and asked again in this
	// session: what it carries is where the answer left it.
	next := startSession(t, api, v)
	if len(next.GetCards()) == 0 {
		t.Fatal("the card answered a moment ago is not asked again")
	}
	card := next.GetCards()[0]
	if card.GetIsNew() {
		t.Error("a card already answered says it was not seen")
	}
	if _, err := time.Parse(review.TimestampLayout, card.GetDue()); err != nil {
		t.Errorf("the card is due at %q, which the page cannot read: %v", card.GetDue(), err)
	}
}

// A person hits the wrong key and takes it back, and what they took back is
// not part of what they answered.
func TestAnAnswerTakenBackIsNotCounted(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]

	session := startSession(t, api, v)
	provided := answerCard(t, api, string(v.ID), session)

	if _, err := api.TakeBackAnswer(t.Context(), connect.NewRequest(&v1.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: session.GetRun(), Answer: provided,
	})); err != nil {
		t.Fatal(err)
	}

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(v.ID)}))
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

// one.
func TestTakingBackWithoutNamingAnAnswerIsRefused(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]
	session := startSession(t, api, v)

	_, err := api.TakeBackAnswer(t.Context(), connect.NewRequest(&v1.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: session.GetRun(),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
}

// An answer is taken back on the run it was given on. A run this window never
// opened is not one to write into.
func TestTakingBackOnARunNobodyOpenedIsRefused(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]
	session := startSession(t, api, v)
	provided := answerCard(t, api, string(v.ID), session)

	_, err := api.TakeBackAnswer(t.Context(), connect.NewRequest(&v1.TakeBackAnswerRequest{
		Vault: string(v.ID), Run: "nothing", Answer: provided,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
}

// An answer names the vault it belongs to. One the installation does not hold
// fails before a line is written into anybody's review.
func TestAnsweringAVaultNobodyContainsIsRefused(t *testing.T) {
	api, _ := newAPI(t, deck)

	_, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
		Vault: "nothing", Run: "nothing",
		Mark: "k7m2xq9fzp", Face: "Recognise", Rating: v1.Rating_RATING_GOOD,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
}

// unreadable is a host whose list of vaults cannot be read.
type unreadable struct{ host.Vault }

func (unreadable) GetVaults(context.Context) ([]domain.Vault, error) {
	return nil, errors.New("the list of vaults cannot be read")
}

// A window that cannot read the installation's vaults says so. An empty list
// would read as an installation holding none, which is a person told their
// vaults are gone.
func TestAWindowThatCannotReadTheVaultsReturnsSo(t *testing.T) {
	_, _, installation := newAPIOn(t)
	api := &reviewHandler{Backend: New(Config{Vault: unreadable{Vault: installation.Host}})}

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(t.Context(),
		connect.NewRequest(&v1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if stream.Receive() {
		t.Fatalf("a window that cannot read the vaults listed %+v", stream.Msg())
	}
	if connect.CodeOf(stream.Err()) != connect.CodeInternal {
		t.Errorf("failed with %v", connect.CodeOf(stream.Err()))
	}
}

// A vault the installation does not hold fails before anything is written.
func TestTakingBackOnAVaultNobodyContainsIsRefused(t *testing.T) {
	api, _ := newAPI(t, deck)

	_, err := api.TakeBackAnswer(t.Context(), connect.NewRequest(&v1.TakeBackAnswerRequest{
		Vault: "nothing", Run: "nothing", Answer: "nothing",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
}

// Something moving underneath the window reaches the page, so a deck written
// while a person is looking at the list is a deck they are shown.
func TestSomethingMovingReachesThePage(t *testing.T) {
	api, _, installation := newAPIOn(t)
	client := newFlashcardsClient(t, api)

	stream, err := client.WatchReloads(t.Context(), connect.NewRequest(&v1.WatchReloadsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	// The stream says it is listening before anything has moved.
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
		if !stream.Msg().GetReload() {
			t.Error("did not receive refresh signal")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("missing update notification for moved item")
	}
}

// A deck edited in place keeps its size and its stamp, and the window keeps no
// face of it past a report that something moved.
func TestAReportedChangeForgetsTheFacesAlreadyParsed(t *testing.T) {
	api, target, installation := newAPIOn(t, deck)
	v := target[0]
	faces := api.scenarios().flash.CardsDue.CardFaces()

	before, err := faces.Execute(t.Context(), v)
	if err != nil || len(before) != 1 {
		t.Fatalf("the vault stands as %d card faces and %v", len(before), err)
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

	stale, err := faces.Execute(t.Context(), v)
	if err != nil || !isFront(stale[0], "Leaf mould") {
		t.Fatalf("the faces were parsed again without a report: %+v %v", stale, err)
	}

	stream, err := newFlashcardsClient(t, api).WatchReloads(t.Context(), connect.NewRequest(&v1.WatchReloadsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("the stream said nothing: %v", stream.Err())
	}
	installation.Host.Move()
	if !stream.Receive() || !stream.Msg().GetReload() {
		t.Fatalf("did not receive refresh signal: %v", stream.Err())
	}
	fresh, err := faces.Execute(t.Context(), v)
	if err != nil || !isFront(fresh[0], "Mould leaf") {
		t.Errorf("the faces stand as they were after a report: %+v %v", fresh, err)
	}
}

func isFront(f flashcards.CardFace, word string) bool {
	front, _ := f.Lay()
	return strings.Contains(front, word)
}

// unopenable is stores that open no vault.
type unopenable struct{}

func (unopenable) Open(domain.Vault) (ports.DerivedStore, error) {
	return nil, errNothingRead
}

// A session whose run cannot be opened is an internal error, and the page is
// handed no run to answer on.
func TestASessionWhoseRunCannotBeOpenedIsRefused(t *testing.T) {
	api, target := newAPI(t, deck)
	api.scenarios().flash.Log = flashcards.Log{Stores: unopenable{}}

	out, err := api.StartSession(t.Context(),
		connect.NewRequest(&v1.StartSessionRequest{Vault: string(target[0].ID)}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
	if out != nil {
		t.Errorf("a session without a run came back with %+v", out.Msg)
	}
}

// A vault whose answers cannot be counted is an internal error and no list of
// days.
func TestAVaultWhoseAnswersCannotBeCountedIsRefused(t *testing.T) {
	api, target := newAPI(t, deck)
	api.scenarios().flash.GetReviewDays = flashcards.NewGetReviewDays(
		flashcards.NewCountReviews(unopenable{}, api.scenarios().flash.Schedules, reviewDay, time.Now))

	out, err := api.ListReviewDays(t.Context(),
		connect.NewRequest(&v1.ListReviewDaysRequest{Vault: string(target[0].ID)}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("failed with %v", connect.CodeOf(err))
	}
	if out != nil {
		t.Errorf("days came back with %+v", out.Msg)
	}
}

// An answer for a vault nobody holds fails as that, and not as a run nobody
// opened.
func TestAnAnswerAndATakingBackNameTheVaultNobodyContains(t *testing.T) {
	api, _ := newAPI(t, deck)

	_, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
		Vault: "nothing", Run: "nothing", Rating: v1.Rating_RATING_GOOD,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound || errors.Is(err, ErrNoRun) {
		t.Errorf("answering failed with %v, want a vault nobody holds", err)
	}

	_, err = api.TakeBackAnswer(t.Context(), connect.NewRequest(&v1.TakeBackAnswerRequest{
		Vault: "nothing", Run: "nothing", Answer: "nothing",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound || errors.Is(err, ErrNoRun) {
		t.Errorf("taking back failed with %v, want a vault nobody holds", err)
	}
}

// Taking an answer back answers with a response, which is empty.
func TestTakingAnAnswerBackIsAnsweredWithAResponse(t *testing.T) {
	api, target := newAPI(t, deck)
	v := target[0]
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
func TestAStretchOfALongHistoryContainsEveryDayInsideIt(t *testing.T) {
	api, target := newAPI(t, lived)
	v := target[0]
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
	pastDays, ahead := names(all.GetDays()), names(all.GetDue())
	if len(pastDays) < 6 || len(ahead) < 6 {
		t.Fatalf("the history holds %d days behind and %d ahead, want six of each", len(pastDays), len(ahead))
	}

	inside := list(pastDays[1], pastDays[len(pastDays)-2])
	if got, want := names(inside.GetDays()), pastDays[1:len(pastDays)-1]; !slices.Equal(got, want) {
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
	api, target := newAPI(t, deck)
	api.scenarios().flash.Log = flashcards.Log{Stores: refusingStores{}}
	v := target[0]
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
	api, target := newAPI(t, deck)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out, err := api.StartSession(ctx, connect.NewRequest(&v1.StartSessionRequest{Vault: string(target[0].ID)}))
	if connect.CodeOf(err) != connect.CodeInternal || out != nil {
		t.Errorf("a session that could not be worked out came back with %+v and %v", out, err)
	}
}
