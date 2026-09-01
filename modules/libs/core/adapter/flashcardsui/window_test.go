package flashcardsui

import (
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// bothWays is a stencil of two faces, so one card of it is two card faces.
const bothWays = "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
	"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n" +
	"\n## Read it\n\n### Front\n\n{{Meaning}}\n\n### Back\n\n{{Word}}\n"

// oneWay is a stencil of one face.
const oneWay = "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
	"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n"

// written is a deck of as many cards cut by one stencil, pointing at the preset
// named. A deck naming none is written with an empty name.
//
// Each deck is written from a mark of its own: a mark names a card, and two
// decks writing one mark are writing one card.
func written(stencil, at string, cards, from int) string {
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
	"decks/Verbs.md": written("Both", "Sanskrit", 150, 0),
	"decks/Nouns.md": written("One", "Sanskrit", 120, 1000),
	"decks/Roots.md": written("One", "Grammar", 150, 2000),
	"decks/Loose.md": written("One", "", 60, 3000),
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
var reviewDay = history.Day{Starts: 4 * time.Hour, In: time.UTC}

// firstMorning is the instant the history begins at, well inside its review day.
var firstMorning = time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)

// standing puts the whole window on one instant: the day the counts stand in,
// the day a sitting is held to, and the day a curve is drawn for.
func standing(api *API, now time.Time) {
	at := func() time.Time { return now }
	api.Now = at
	api.Day = reviewDay
	api.Owed.Day, api.Owed.Now = reviewDay, at
	api.Session.Day, api.Session.Now = reviewDay, at
	api.Curves.Day, api.Curves.Now = reviewDay, at
	api.Counted.Day, api.Counted.Now = reviewDay, at
}

// lives sits down to the vault on each of as many days running.
//
// It is the history a person leaves behind: days finished and days given up on
// part way, days missed altogether, card faces at every stage of being learned,
// and answer times the projections are costed from. A day nobody sat to leaves
// its cards owed, which is the backlog the budgets then have to carry.
func lives(t *testing.T, api *API, v domain.Vault, days int) {
	t.Helper()
	for day := range days {
		morning := firstMorning.AddDate(0, 0, day)
		standing(api, morning)
		if day%5 == 4 {
			continue
		}

		sitting := started(t, api, v)
		// How far a person got before they stopped, which is not always the end.
		through := len(sitting.GetAsked())
		if day%3 == 1 {
			through = through * 2 / 3
		}
		for i, one := range sitting.GetAsked()[:through] {
			// The times run over a spread a person's would, and every one of
			// them is counted whole.
			took := time.Duration(4+(i+day)%9) * time.Second
			rating := v1.Rating_RATING_GOOD
			if (i+day)%7 == 0 {
				rating = v1.Rating_RATING_AGAIN
			}
			_, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
				VaultId: v.ID, Run: sitting.GetRun(),
				Card: one.GetCard(), Face: one.GetFace(),
				Rating: rating, TookMs: took.Milliseconds(),
			}))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

// owing is what the front door says about one vault.
func owing(t *testing.T, api *API, v domain.Vault) *v1.VaultOwing {
	t.Helper()
	for _, one := range front(t, api).GetVaults() {
		if one.GetVaultId() == v.ID {
			if one.GetUnread() != "" {
				t.Fatalf("the vault could not be counted: %s", one.GetUnread())
			}
			return one
		}
	}
	t.Fatalf("the front door does not hold the vault %s", v.ID)
	return nil
}

// offers is how many card faces the deck screen puts in front of a person under
// one preset: the cards of every deck that preset schedules, owed and new.
func offers(said *v1.VaultOwing, decks []string) int {
	out := 0
	for _, one := range said.GetDecks() {
		for _, deck := range decks {
			if one.GetDeck() == deck {
				out += int(one.GetDue()) + int(one.GetNew())
			}
		}
	}
	return out
}

// asWritten is the preset at path as its file now stands.
func asWritten(t *testing.T, api *API, v domain.Vault, path string) history.Preset {
	t.Helper()
	found, err := api.Presets.Read(t.Context(), v, path)
	if err != nil {
		t.Fatal(err)
	}
	return found.Preset
}

// writtenBack writes settings into the preset at path, which is what the window
// does when a person lets go of the control.
func writtenBack(t *testing.T, api *API, v domain.Vault, path string, p history.Preset) {
	t.Helper()
	if _, err := api.Presets.Save(t.Context(), v, path, p, domain.FileRef{}); err != nil {
		t.Fatal(err)
	}
}

// pictured is what the preset tab draws for these settings.
func pictured(t *testing.T, api *API, v domain.Vault, path string, p history.Preset) *v1.Curve {
	t.Helper()
	out, err := api.Curve(t.Context(), connect.NewRequest(&v1.FlashcardsServiceCurveRequest{
		VaultId: v.ID, Path: path, Settings: wire.SettingsOf(p),
	}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg.GetCurve()
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
		goal   history.Goal
		places []int
		// steered puts the value at a place of the grid into the settings.
		steered func(p *history.Preset, value float64)
	}{
		{
			// A curve of minutes runs to twice what carrying the whole load
			// costs, so a day in the lower half of the grid stands on the grid
			// it was read from and the picture is drawn for the day it is left
			// at.
			what: "steered by its minutes", goal: history.GoalMinutes,
			places:  []int{1, 4, 9},
			steered: func(p *history.Preset, value float64) { p.MinutesADay = int(value) },
		},
	} {
		t.Run(one.what, func(t *testing.T) {
			api, held := windowed(t, lived)
			v := held[0]
			lives(t, api, v, 14)
			standing(api, firstMorning.AddDate(0, 0, 14))

			p := asWritten(t, api, v, "Sanskrit.md")
			p.Goal = one.goal
			for _, at := range one.places {
				// The control is moved to a place of the grid and let go of, so
				// the preset the decks are held to is the one the curve drew.
				drawn := pictured(t, api, v, "Sanskrit.md", p)
				if len(drawn.GetGrid()) <= at {
					t.Fatalf("the curve has %d places", len(drawn.GetGrid()))
				}
				value := drawn.GetGrid()[at]
				one.steered(&p, value)
				writtenBack(t, api, v, "Sanskrit.md", p)

				drawn = pictured(t, api, v, "Sanskrit.md", p)
				if drawn.GetGrid()[at] != value {
					t.Fatalf("the control was left at %v and the grid holds %v there",
						value, drawn.GetGrid()[at])
				}
				draws := int(drawn.GetAt()[at].GetReviews())
				if got := offers(owing(t, api, v), underSanskrit); got != draws {
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
	api, held := windowed(t, lived)
	v := held[0]
	standing(api, firstMorning)

	drawn := pictured(t, api, v, "Sanskrit.md", asWritten(t, api, v, "Sanskrit.md"))
	want := facesIn["decks/Verbs.md"] + facesIn["decks/Nouns.md"]
	if int(drawn.GetDecks()) != len(underSanskrit) || int(drawn.GetCards()) != want {
		t.Errorf("the curve stands over %d decks and %d card faces, want %d and %d",
			drawn.GetDecks(), drawn.GetCards(), len(underSanskrit), want)
	}
}

// sole are the decks that are the only deck of their preset, so the budget the
// front door divided among the decks of that preset went to this one.
var sole = map[string]bool{"decks/Roots.md": true, "decks/Loose.md": true}

// What each deck offers adds up to what the vault offers, and sitting down to
// one deck asks for that deck's share and nothing else.
func TestEachDecksShareOfTheDayAddsUpToTheVaults(t *testing.T) {
	api, held := windowed(t, lived)
	v := held[0]
	lives(t, api, v, 14)
	standing(api, firstMorning.AddDate(0, 0, 14))

	said := owing(t, api, v)
	if int(said.GetFaces()) != 630 {
		t.Errorf("the vault holds %d card faces, want 630", said.GetFaces())
	}
	due, fresh := 0, 0
	for _, one := range said.GetDecks() {
		if got := int(one.GetFaces()); got != facesIn[one.GetDeck()] {
			t.Errorf("%s holds %d card faces, want %d",
				one.GetDeck(), got, facesIn[one.GetDeck()])
		}
		due += int(one.GetDue())
		fresh += int(one.GetNew())

		sat, err := api.Start(t.Context(), connect.NewRequest(&v1.StartRequest{
			VaultId: v.ID, Deck: one.GetDeck(),
		}))
		if err != nil {
			t.Fatal(err)
		}
		want := int(one.GetDue()) + int(one.GetNew())
		got := len(sat.Msg.GetAsked())
		if sole[one.GetDeck()] && got != want {
			t.Errorf("%s offers %d card faces and its sitting asks %d",
				one.GetDeck(), want, got)
		}
		if got < want {
			t.Errorf("%s offers %d card faces and its sitting asks %d",
				one.GetDeck(), want, got)
		}
		for _, card := range sat.Msg.GetAsked() {
			if card.GetDeck() != one.GetDeck() {
				t.Fatalf("a sitting over %s asked a card of %s",
					one.GetDeck(), card.GetDeck())
			}
		}
	}
	if due != int(said.GetDue()) || fresh != int(said.GetNew()) {
		t.Errorf("the decks come to %d owed and %d new, the vault to %d and %d",
			due, fresh, said.GetDue(), said.GetNew())
	}
}

// A deck says how much of it was answered in the day holding now, so a deck
// nobody has answered in today is told apart from one whose day is done.
func TestADeckSaysHowMuchOfItWasAnsweredToday(t *testing.T) {
	api, held := windowed(t, lived)
	v := held[0]
	lives(t, api, v, 14)

	// A new day, with nothing answered in it yet.
	standing(api, firstMorning.AddDate(0, 0, 14))
	for _, one := range owing(t, api, v).GetDecks() {
		if one.GetAnswered() != 0 {
			t.Errorf("nothing was answered today and %s counts %d",
				one.GetDeck(), one.GetAnswered())
		}
	}

	// One deck answered, and only that deck counts it.
	sat, err := api.Start(t.Context(), connect.NewRequest(&v1.StartRequest{
		VaultId: v.ID, Deck: "decks/Roots.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(sat.Msg.GetAsked()) < 3 {
		t.Fatalf("the deck offered %d card faces to answer", len(sat.Msg.GetAsked()))
	}
	for _, card := range sat.Msg.GetAsked()[:3] {
		if _, err := api.Answer(t.Context(), connect.NewRequest(&v1.AnswerRequest{
			VaultId: v.ID, Run: sat.Msg.GetRun(),
			Card: card.GetCard(), Face: card.GetFace(),
			Rating: v1.Rating_RATING_GOOD, TookMs: 6000,
		})); err != nil {
			t.Fatal(err)
		}
	}

	for _, one := range owing(t, api, v).GetDecks() {
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

// The front door holds each vault once, and each row carries that vault's own
// count.
func TestTheFrontDoorHoldsEachVaultOnce(t *testing.T) {
	api, held := windowed(t, lived, deck)
	standing(api, firstMorning)

	rows := front(t, api).GetVaults()
	if len(rows) != len(held) {
		t.Fatalf("the front door holds %d rows for %d vaults", len(rows), len(held))
	}
	seen := make(map[string]int, len(rows))
	for _, one := range rows {
		seen[one.GetVaultId()]++
	}
	for _, v := range held {
		if seen[v.ID] != 1 {
			t.Errorf("the vault %s stands on the front door %d times", v.ID, seen[v.ID])
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
	if got := owing(t, api, held[0]).GetFaces(); got != 630 {
		t.Errorf("the lived vault counts %d card faces, want 630", got)
	}
	if got := owing(t, api, held[1]).GetFaces(); got != 1 {
		t.Errorf("the vault of one card counts %d card faces", got)
	}
}

// A preset's target reaches the cards it schedules, and a working out kept
// under one target is not answered with under another.
//
// Asking for more of the cards back is shorter intervals, so the same history
// leaves more of them owed today.
func TestATargetMovedIsNotAnsweredFromTheWorkingOutUnderTheOldOne(t *testing.T) {
	api, held := windowed(t, lived)
	v := held[0]
	lives(t, api, v, 14)
	standing(api, firstMorning.AddDate(0, 0, 14))

	p := asWritten(t, api, v, "Grammar.md")
	p.Retention = history.RetentionBounds.Least
	writtenBack(t, api, v, "Grammar.md", p)
	// The first count is what writes the working out down, so the second is the
	// one that could be answered from it.
	least := offers(owing(t, api, v), underGrammar)

	p.Retention = history.RetentionBounds.Most
	writtenBack(t, api, v, "Grammar.md", p)
	most := offers(owing(t, api, v), underGrammar)

	if most <= least {
		t.Errorf("asking for %v of the cards back offers %d, and %v offers %d",
			history.RetentionBounds.Most, most, history.RetentionBounds.Least, least)
	}

	// And the preset beside it, whose target nobody moved, is where it was.
	if got := offers(owing(t, api, v), underSanskrit); got == 0 {
		t.Error("the preset nobody touched offers nothing")
	}
}
