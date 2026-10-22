package service

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/service/presetwire"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// bothWays is a stencil of two faces, so one card of it is two card faces.
const bothWays = "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
	"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n" +
	"\n## Read it\n\n### Front\n\n{{Meaning}}\n\n### Back\n\n{{Word}}\n"

// oneWay is a stencil of one face.
const oneWay = "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
	"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n"

// newStencilDeck is a deck of as many cards cut by one stencil, pointing at the
// preset named. A deck naming none is written with an empty name.
//
// Each deck is written from a mark of its own: a mark names a card, and two
// decks writing one mark are writing one card.
func newStencilDeck(stencil, at string, cards, from int) string {
	out := "---\ntype: deck\n"
	if at != "" {
		out += "links:\n  - to: " + at + "\n    role: ref\n    type: preset\n"
	}
	out += "---\n"
	for i := from; i < from+cards; i++ {
		out += fmt.Sprintf(
			"\n## Card %d ^face%06d\n\n[[%s]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n",
			i, i, stencil, i, i)
	}
	return out
}

// lived is a vault at the size a person's is: two presets and the defaults, four
// decks between them, and card faces in the hundreds.
//
// Verbs is cut by the two-faced stencil, so its cards are twice its faces.
var lived = map[string]string{
	"Both.md": bothWays,
	"One.md":  oneWay,
	"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 12\n" +
		"new_a_day: 15\nreviews_a_day: 120\nretention: 0.9\ncounts: cards\n---\n\n# Sanskrit\n",
	"Grammar.md": "---\ntype: preset\ngoal: retention\nretention: 0.9\n" +
		"new_a_day: 10\nreviews_a_day: 40\nminutes_a_day: 15\ncounts: cards\n---\n\n# Grammar\n",
	"decks/Verbs.md": newStencilDeck("Both", "Sanskrit", 150, 0),
	"decks/Nouns.md": newStencilDeck("One", "Sanskrit", 120, 1000),
	"decks/Roots.md": newStencilDeck("One", "Grammar", 150, 2000),
	"decks/Loose.md": newStencilDeck("One", "", 60, 3000),
}

// The card faces of the lived vault, by the deck they stand in.
var facesIn = map[string]int{
	"decks/Verbs.md": 300,
	"decks/Nouns.md": 120,
	"decks/Roots.md": 150,
	"decks/Loose.md": 60,
}

// underSanskrit and underGrammar are the decks each preset schedules.
var (
	underSanskrit = []string{"decks/Nouns.md", "decks/Verbs.md"}
	underGrammar  = []string{"decks/Roots.md"}
)

// reviewDay is the day the counts of this file stand in: a boundary four hours
// past midnight, counted in one zone whatever machine runs the test.
var reviewDay = review.Day{Starts: 4 * time.Hour, In: time.UTC}

// firstMorning is the instant the history begins at, well inside its review day.
var firstMorning = time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)

// setNow synchronizes window time, review day calculation, and scenario clocks to now.
func setNow(api *API, now time.Time) {
	at := func() time.Time { return now }
	api.now = at
	api.scenarios().day = reviewDay
	api.scenarios().flash.Record = flashcards.NewRecord(at)
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithDay(reviewDay).WithNow(at)
	api.scenarios().flash.Session = api.scenarios().flash.Session.WithDay(reviewDay).WithNow(at)
	api.scenarios().flash.GetReviewDays = api.scenarios().flash.GetReviewDays.WithDay(reviewDay).WithNow(at)
}

// runDays simulates consecutive daily review sessions across a vault.
func runDays(t *testing.T, api *API, v domain.Vault, days int) {
	t.Helper()
	for day := range days {
		morning := firstMorning.AddDate(0, 0, day)
		setNow(api, morning)
		if day%5 == 4 {
			continue
		}

		session := startSession(t, api, v)
		// How far a person got before they stopped, which is not always the end.
		cardCount := len(session.GetCards())
		if day%3 == 1 {
			cardCount = cardCount * 2 / 3
		}
		for i, one := range session.GetCards()[:cardCount] {
			// The times run over a spread a person's would, and every one of
			// them is counted whole.
			took := time.Duration(4+(i+day)%9) * time.Second
			rating := v1.Rating_RATING_GOOD
			if (i+day)%7 == 0 {
				rating = v1.Rating_RATING_AGAIN
			}
			_, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
				Vault: string(v.ID), Run: session.GetRun(),
				Mark: one.GetMark(), Face: one.GetFace(),
				Rating: rating, TookMs: took.Milliseconds(),
			}))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

// getVaultCount returns the cards due summary for one vault.
func getVaultCount(t *testing.T, api *API, v domain.Vault) *v1.VaultCardsDue {
	t.Helper()
	for _, one := range front(t, api).GetVaults() {
		if one.GetId() == string(v.ID) {
			if one.GetUnread() != "" {
				t.Fatalf("the vault could not be counted: %s", one.GetUnread())
			}
			return one
		}
	}
	t.Fatalf("summary does not contain vault %s", string(v.ID))
	return nil
}

// countOfferedCards returns how many card faces the deck screen puts in front of a person under
// one preset: the cards of every deck that preset schedules, owed and new.
func countOfferedCards(due *v1.VaultCardsDue, decks []string) int {
	out := 0
	for _, one := range due.GetDecks() {
		for _, deck := range decks {
			if one.GetDeck() == deck {
				out += int(one.GetDue()) + int(one.GetNew())
			}
		}
	}
	return out
}

// asWritten reads the preset configuration stored at path.
func asWritten(t *testing.T, api *API, v domain.Vault, path string) review.Preset {
	t.Helper()
	found, err := api.scenarios().flash.Presets.Read(t.Context(), v, path)
	if err != nil {
		t.Fatal(err)
	}
	return found.Settings
}

// writePreset writes settings into the preset at path, which is what the window
// does when a person lets go of the control.
func writePreset(t *testing.T, api *API, v domain.Vault, path string, p review.Preset) {
	t.Helper()
	if _, err := api.scenarios().flash.Presets.Save(t.Context(), v, path, p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}
}

// getCurve is what these settings come to over the whole range of their goal,
// drawn by the simulator this window's own pieces make.
func getCurve(t *testing.T, api *API, v domain.Vault, path string, p review.Preset) *v1.Curve {
	t.Helper()
	target, err := curves(api).Execute(t.Context(), v, path, p)
	if err != nil {
		t.Fatal(err)
	}
	return presetwire.ConvertCurveToWire(target)
}

// curves is the simulator behind the one control of a preset, standing on the
// day the window relies on.
func curves(api *API) flashcards.ProjectCurve {
	c := flashcards.NewProjectCurve(
		api.scenarios().flash.Schedules.CardFaces(),
		api.scenarios().flash.Schedules,
		api.scenarios().flash.Presets,
		api.scenarios().day,
		api.now,
	)
	c.Cores = runtime.GOMAXPROCS(0)
	return c
}

// The picture over a preset's range and the day its decks offer are one
// arithmetic.
//
// A person reads a count of cards off the curve where their control stands, and
// then reads a count of cards off the deck screen. The two are the same day of
// the same preset, so they are the same number.
//
// It is a goal of minutes that this holds for, because there the control moves
// today. A target of retention moves nothing until tomorrow, and its curve is
// the load over the days projected.
func TestTheCurveAndTheDeckScreenOfferTheSameDay(t *testing.T) {
	for _, one := range []struct {
		what   string
		goal   review.Goal
		places []int
		// steered puts the value at a place of the grid into the settings.
		steered func(p *review.Preset, value float64)
	}{
		{
			// A curve of minutes runs to twice what carrying the whole load
			// costs, so a day in the lower half of the grid relies on the grid
			// it was read from and the picture is drawn for the day it is left
			// at.
			what: "steered by its minutes", goal: review.GoalMinutes,
			places:  []int{1, 4, 9},
			steered: func(p *review.Preset, value float64) { p.MinutesADay = int(value) },
		},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, target := newAPI(t, lived)
			v := target[0]
			runDays(t, api, v, 14)
			setNow(api, firstMorning.AddDate(0, 0, 14))

			p := asWritten(t, api, v, "Sanskrit.md")
			p.Goal = one.goal
			for _, at := range one.places {
				// Apply curve configuration to preset.
				curveResp := getCurve(t, api, v, "Sanskrit.md", p)
				if len(curveResp.GetGrid()) <= at {
					t.Fatalf("the curve has %d places", len(curveResp.GetGrid()))
				}
				value := curveResp.GetGrid()[at]
				one.steered(&p, value)
				writePreset(t, api, v, "Sanskrit.md", p)

				curveResp = getCurve(t, api, v, "Sanskrit.md", p)
				if curveResp.GetGrid()[at] != value {
					t.Fatalf("the control was left at %v and the grid holds %v there",
						value, curveResp.GetGrid()[at])
				}
				draws := int(curveResp.GetAt()[at].GetReviews())
				if got := countOfferedCards(getVaultCount(t, api, v), underSanskrit); got != draws {
					t.Errorf("at %v the curve draws %d cards for the day and the deck "+
						"screen offers %d", value, draws, got)
				}
			}
		})
	}
}

// A curve carries the decks and the card faces the preset schedules, which is
// what tells a preset with nothing under it from one whose day is simply spent.
func TestTheCurveCarriesWhatThePresetSchedules(t *testing.T) {
	api, target := newAPI(t, lived)
	v := target[0]
	setNow(api, firstMorning)

	curveResp := getCurve(t, api, v, "Sanskrit.md", asWritten(t, api, v, "Sanskrit.md"))
	want := facesIn["decks/Verbs.md"] + facesIn["decks/Nouns.md"]
	if int(curveResp.GetDecks()) != len(underSanskrit) || int(curveResp.GetCards()) != want {
		t.Errorf("curve deck count = %d, face count = %d, want %d and %d",
			curveResp.GetDecks(), curveResp.GetCards(), len(underSanskrit), want)
	}
}

// sole are the decks that are the only deck of their preset, so the daily budget
// allocated to that preset went to this deck.
var sole = map[string]bool{"decks/Roots.md": true, "decks/Loose.md": true}

// What each deck offers adds up to what the vault offers, and a session on one
// deck asks for that deck's share and nothing else.
func TestEachDecksShareOfTheDayAddsUpToTheVaults(t *testing.T) {
	api, target := newAPI(t, lived)
	v := target[0]
	runDays(t, api, v, 14)
	setNow(api, firstMorning.AddDate(0, 0, 14))

	response := getVaultCount(t, api, v)
	if int(response.GetFaces()) != 630 {
		t.Errorf("the vault holds %d card faces, want 630", response.GetFaces())
	}
	due, newCards := 0, 0
	for _, one := range response.GetDecks() {
		if got := int(one.GetFaces()); got != facesIn[one.GetDeck()] {
			t.Errorf("%s holds %d card faces, want %d",
				one.GetDeck(), got, facesIn[one.GetDeck()])
		}
		due += int(one.GetDue())
		newCards += int(one.GetNew())

		sat, err := api.StartSession(t.Context(), connect.NewRequest(&v1.StartSessionRequest{
			Vault: string(v.ID), Deck: one.GetDeck(),
		}))
		if err != nil {
			t.Fatal(err)
		}
		want := int(one.GetDue()) + int(one.GetNew())
		got := len(sat.Msg.GetCards())
		if sole[one.GetDeck()] && got != want {
			t.Errorf("%s offers %d card faces and its session asks %d",
				one.GetDeck(), want, got)
		}
		if got < want {
			t.Errorf("%s offers %d card faces and its session asks %d",
				one.GetDeck(), want, got)
		}
		for _, card := range sat.Msg.GetCards() {
			if card.GetDeck() != one.GetDeck() {
				t.Fatalf("a session over %s asked a card of %s",
					one.GetDeck(), card.GetDeck())
			}
		}
	}
	if due != int(response.GetDue()) || newCards != int(response.GetNew()) {
		t.Errorf("the decks come to %d owed and %d new, the vault to %d and %d",
			due, newCards, response.GetDue(), response.GetNew())
	}
}

// TestADeckReturnsHowMuchOfItWasAnsweredToday verifies answered count metrics per deck.
func TestADeckReturnsHowMuchOfItWasAnsweredToday(t *testing.T) {
	api, target := newAPI(t, lived)
	v := target[0]
	runDays(t, api, v, 14)

	// A new day, with nothing answered in it yet.
	setNow(api, firstMorning.AddDate(0, 0, 14))
	for _, one := range getVaultCount(t, api, v).GetDecks() {
		if one.GetAnswered() != 0 {
			t.Errorf("nothing was answered today and %s counts %d",
				one.GetDeck(), one.GetAnswered())
		}
	}

	// One deck answered, and only that deck counts it.
	sat, err := api.StartSession(t.Context(), connect.NewRequest(&v1.StartSessionRequest{
		Vault: string(v.ID), Deck: "decks/Roots.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(sat.Msg.GetCards()) < 3 {
		t.Fatalf("the deck offered %d card faces to answer", len(sat.Msg.GetCards()))
	}
	for _, card := range sat.Msg.GetCards()[:3] {
		if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
			Vault: string(v.ID), Run: sat.Msg.GetRun(),
			Mark: card.GetMark(), Face: card.GetFace(),
			Rating: v1.Rating_RATING_GOOD, TookMs: 6000,
		})); err != nil {
			t.Fatal(err)
		}
	}

	for _, one := range getVaultCount(t, api, v).GetDecks() {
		want := 0
		if one.GetDeck() == "decks/Roots.md" {
			want = 3
		}
		if int(one.GetAnswered()) != want {
			t.Errorf("%s counts %d answered today, want %d",
				one.GetDeck(), one.GetAnswered(), want)
		}
	}
}

// CardsDue summary includes each vault once.
func TestCardsDueSummaryContainsEachVaultOnce(t *testing.T) {
	api, target := newAPI(t, lived, deck)
	setNow(api, firstMorning)

	rows := front(t, api).GetVaults()
	if len(rows) != len(target) {
		t.Fatalf("summary contains %d rows for %d vaults", len(rows), len(target))
	}
	observed := make(map[string]int, len(rows))
	for _, one := range rows {
		observed[one.GetId()]++
	}
	for _, v := range target {
		if observed[string(v.ID)] != 1 {
			t.Errorf("vault %s count = %d", string(v.ID), observed[string(v.ID)])
		}
	}

	// The counts are each vault's own, and the deck rows of one add up to it.
	for _, row := range rows {
		faces := 0
		for _, one := range row.GetDecks() {
			faces += int(one.GetFaces())
		}
		if faces != int(row.GetFaces()) {
			t.Errorf("the decks of %s hold %d card faces and the vault holds %d",
				row.GetName(), faces, row.GetFaces())
		}
	}
	if got := getVaultCount(t, api, target[0]).GetFaces(); got != 630 {
		t.Errorf("the lived vault counts %d card faces, want 630", got)
	}
	if got := getVaultCount(t, api, target[1]).GetFaces(); got != 1 {
		t.Errorf("the vault of one card counts %d card faces", got)
	}
}

// A preset's target reaches the cards it schedules, and a working out kept
// under one target is not answered with under another.
//
// Asking for more of the cards back is shorter intervals, so the same history
// leaves more of them owed today.
func TestATargetMovedIsNotAnsweredFromTheWorkingOutBelowTheOldOne(t *testing.T) {
	api, target := newAPI(t, lived)
	v := target[0]
	runDays(t, api, v, 14)
	setNow(api, firstMorning.AddDate(0, 0, 14))

	p := asWritten(t, api, v, "Grammar.md")
	p.Retention = review.RetentionBounds.Least
	writePreset(t, api, v, "Grammar.md", p)
	// The first count is what writes the working out down, so the second is the
	// one that could be answered from it.
	least := countOfferedCards(getVaultCount(t, api, v), underGrammar)

	p.Retention = review.RetentionBounds.Most
	writePreset(t, api, v, "Grammar.md", p)
	most := countOfferedCards(getVaultCount(t, api, v), underGrammar)

	if most <= least {
		t.Errorf("asking for %v of the cards back offers %d, and %v offers %d",
			review.RetentionBounds.Most, most, review.RetentionBounds.Least, least)
	}

	// And the preset beside it, whose target nobody moved, is where it was.
	if got := countOfferedCards(getVaultCount(t, api, v), underSanskrit); got == 0 {
		t.Error("the preset nobody touched offers nothing")
	}
}
