package service

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/presetwire"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// bothWays is a card model of two review cards, so one card of it is two review cards.
const bothWays = "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
	"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n" +
	"\n## Read it\n\n### Front\n\n{{Meaning}}\n\n### Back\n\n{{Word}}\n"

// oneWay is a card model of one review card.
const oneWay = "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
	"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n"

func newCardModelDeck(cardModel, pos string, cards, from int) string {
	out := "---\ntype: deck\n"
	if pos != "" {
		out += "links:\n  - to: " + pos + "\n    role: ref\n    type: preset\n"
	}
	out += "---\n"
	for i := from; i < from+cards; i++ {
		out += fmt.Sprintf(
			"\n## Card %d ^card%06d\n\n[[%s]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n",
			i, i, cardModel, i, i)
	}
	return out
}

func getLived() map[string]string {
	return map[string]string{
		"Both.md": bothWays,
		"One.md":  oneWay,
		"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 12\n" +
			"new_a_day: 15\nreviews_a_day: 120\nretention: 0.9\ncounts: cards\n---\n\n# Sanskrit\n",
		"Grammar.md": "---\ntype: preset\ngoal: retention\nretention: 0.9\n" +
			"new_a_day: 10\nreviews_a_day: 40\nminutes_a_day: 15\ncounts: cards\n---\n\n# Grammar\n",
		"decks/Verbs.md": newCardModelDeck("Both", "Sanskrit", 150, 0),
		"decks/Nouns.md": newCardModelDeck("One", "Sanskrit", 120, 1000),
		"decks/Roots.md": newCardModelDeck("One", "Grammar", 150, 2000),
		"decks/Loose.md": newCardModelDeck("One", "", 60, 3000),
	}
}

func getReviewCardsIn() map[string]int {
	return map[string]int{
		"decks/Verbs.md": 300,
		"decks/Nouns.md": 120,
		"decks/Roots.md": 150,
		"decks/Loose.md": 60,
	}
}

func getUnderSanskrit() []string {
	return []string{"decks/Nouns.md", "decks/Verbs.md"}
}

func getUnderGrammar() []string {
	return []string{"decks/Roots.md"}
}

func getReviewDay() review.Day {
	return review.Day{Starts: 4 * time.Hour, In: time.UTC}
}

func getFirstMorning() time.Time {
	return time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)
}

func setNow(handler *API, now time.Time) {
	pos := clock.NewFake(now)
	handler.now = pos
	*handler.day() = getReviewDay()
	handler.flash().Record = flashcards.NewRecord(pos)
	handler.flash().CardsDue = handler.flash().CardsDue.WithDay(getReviewDay()).WithNow(pos)
	handler.flash().Session = handler.flash().Session.WithDay(getReviewDay()).WithNow(pos)
	handler.flash().GetReviewDays = handler.flash().GetReviewDays.WithDay(getReviewDay()).WithNow(pos)
}

func runDays(t *testing.T, handler *API, v domain.Vault, days int) {
	t.Helper()
	for day := range days {
		morning := getFirstMorning().AddDate(0, 0, day)
		setNow(handler, morning)
		if day%5 == 4 {
			continue
		}

		session := startSession(t, handler, v)
		// How far the session got before it stopped, which is not always the end.
		cardCount := len(session.Cards)
		if day%3 == 1 {
			cardCount = cardCount * 2 / 3
		}
		for i, one := range session.Cards[:cardCount] {
			// The times follow a realistic spread, and every one is counted in full.
			took := time.Duration(4+(i+day)%9) * time.Second
			rating := api.RatingGood
			if (i+day)%7 == 0 {
				rating = api.RatingAgain
			}
			_, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
				Vault: string(v.ID), Run: session.Run,
				CardID: one.CardID, Template: one.Template,
				Rating: rating, DurationMs: took.Milliseconds(),
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func getVaultCount(t *testing.T, handler *API, v domain.Vault) *api.VaultCardsDue {
	t.Helper()
	for _, one := range front(t, handler).Vaults {
		if one.ID == string(v.ID) {
			if one.Unread != "" {
				t.Fatalf("the vault could not be counted: %s", one.Unread)
			}
			return one
		}
	}
	t.Fatalf("summary does not contain vault %s", string(v.ID))
	return nil
}

func countSessionCards(due *api.VaultCardsDue, decks []string) int {
	out := 0
	for _, one := range due.Decks {
		for _, deck := range decks {
			if one.Deck == deck {
				out += int(one.Due) + int(one.New)
			}
		}
	}
	return out
}

func readPresetConfig(t *testing.T, handler *API, v domain.Vault, path string) review.Preset {
	t.Helper()
	found, err := handler.flash().Presets.Read(t.Context(), v, path)
	if err != nil {
		t.Fatal(err)
	}
	return found.Settings
}

func writePreset(t *testing.T, handler *API, v domain.Vault, path string, p review.Preset) {
	t.Helper()
	if _, err := handler.flash().Presets.Save(t.Context(), v, path, p, domain.FileStat{}); err != nil {
		t.Fatal(err)
	}
}

func getCurve(t *testing.T, handler *API, v domain.Vault, path string, p review.Preset) *api.Curve {
	t.Helper()
	target, err := curves(handler).Execute(t.Context(), v, path, p)
	if err != nil {
		t.Fatal(err)
	}
	return presetwire.ConvertCurveToWire(target)
}

func curves(handler *API) flashcards.ProjectCurve {
	c := flashcards.NewProjectCurve(
		handler.flash().Schedules.ReviewCards(),
		handler.flash().Schedules,
		handler.flash().Presets,
		*handler.day(),
		handler.now,
	)
	c.Cores = runtime.GOMAXPROCS(0)
	return c
}

// The curve over a preset's range and the day its decks offer use the same arithmetic: the
// card count read from the curve at the control position equals the one from the deck screen.
// This holds for a goal of minutes, because there the control changes today. A retention target
// changes nothing until tomorrow, and its curve is the load over the projected days.
func TestCurveAndDeckScreen_ShareSameDay(t *testing.T) {
	for _, one := range []struct {
		what    string
		goal    review.Goal
		places  []int
		steered func(p *review.Preset, value float64)
	}{
		{
			// A curve of minutes extends to twice the cost of carrying the whole load, so
			// a day in the lower half of the grid depends on the grid the curve was read
			// from, and the plot is drawn for the day at which the control is left.
			what: "steered by its minutes", goal: review.GoalMinutes,
			places:  []int{1, 4, 9},
			steered: func(p *review.Preset, value float64) { p.MinutesADay = int(value) },
		},
	} {
		t.Run(one.what, func(t *testing.T) {
			handler, target := newAPI(t, getLived())
			v := target[0]
			runDays(t, handler, v, 14)
			setNow(handler, getFirstMorning().AddDate(0, 0, 14))

			p := readPresetConfig(t, handler, v, "Sanskrit.md")
			p.Goal = one.goal
			for _, item := range one.places {
				// Apply curve configuration to preset.
				curveResp := getCurve(t, handler, v, "Sanskrit.md", p)
				if len(curveResp.Grid) <= item {
					t.Fatalf("the curve has %d places", len(curveResp.Grid))
				}
				value := curveResp.Grid[item]
				one.steered(&p, value)
				writePreset(t, handler, v, "Sanskrit.md", p)

				curveResp = getCurve(t, handler, v, "Sanskrit.md", p)
				if curveResp.Grid[item] != value {
					t.Fatalf("the control was left at %v and the grid holds %v there",
						value, curveResp.Grid[item])
				}
				draws := int(curveResp.At[item].Reviews)
				if got := countSessionCards(getVaultCount(t, handler, v), getUnderSanskrit()); got != draws {
					t.Errorf("at %v the curve draws %d cards for the day and the deck "+
						"screen offers %d", value, draws, got)
				}
			}
		})
	}
}

// A curve carries the decks and the review cards the preset schedules, which
// distinguishes a preset with nothing scheduled from one whose day is fully
// used.
func TestTheCurveCarriesWhatThePresetSchedules(t *testing.T) {
	handler, target := newAPI(t, getLived())
	v := target[0]
	setNow(handler, getFirstMorning())

	curveResp := getCurve(t, handler, v, "Sanskrit.md", readPresetConfig(t, handler, v, "Sanskrit.md"))
	want := getReviewCardsIn()["decks/Verbs.md"] + getReviewCardsIn()["decks/Nouns.md"]
	if int(curveResp.Decks) != len(getUnderSanskrit()) || int(curveResp.Cards) != want {
		t.Errorf("curve deck count = %d, review card count = %d, want %d and %d",
			curveResp.Decks, curveResp.Cards, len(getUnderSanskrit()), want)
	}
}

func getSole() map[string]bool {
	return map[string]bool{"decks/Roots.md": true, "decks/Loose.md": true}
}

// The cards each deck offers add up to the cards the vault offers, and a
// session on one deck requests only that deck's share.
func TestEachDecksShareOfTheDayAddsUpToTheVaults(t *testing.T) {
	handler, target := newAPI(t, getLived())
	v := target[0]
	runDays(t, handler, v, 14)
	setNow(handler, getFirstMorning().AddDate(0, 0, 14))

	response := getVaultCount(t, handler, v)
	if int(response.ReviewCards) != 630 {
		t.Errorf("the vault holds %d review cards, want 630", response.ReviewCards)
	}
	due, newCards := 0, 0
	for _, one := range response.Decks {
		if got := int(one.ReviewCards); got != getReviewCardsIn()[one.Deck] {
			t.Errorf("%s holds %d review cards, want %d",
				one.Deck, got, getReviewCardsIn()[one.Deck])
		}
		due += int(one.Due)
		newCards += int(one.New)

		sat, err := handler.StartSession(t.Context(), &api.StartSessionRequest{
			Vault: string(v.ID), Deck: one.Deck,
		})
		if err != nil {
			t.Fatal(err)
		}
		want := int(one.Due) + int(one.New)
		got := len(sat.Cards)
		if getSole()[one.Deck] && got != want {
			t.Errorf("%s offers %d review cards and its session asks %d",
				one.Deck, want, got)
		}
		if got < want {
			t.Errorf("%s offers %d review cards and its session asks %d",
				one.Deck, want, got)
		}
		for _, card := range sat.Cards {
			if card.Deck != one.Deck {
				t.Fatalf("a session over %s asked a card of %s",
					one.Deck, card.Deck)
			}
		}
	}
	if due != int(response.Due) || newCards != int(response.New) {
		t.Errorf("the decks come to %d due and %d new, the vault to %d and %d",
			due, newCards, response.Due, response.New)
	}
}

// TestVaultCount_ReturnsReviewedTodayPerDeck verifies answered count metrics per deck.
func TestVaultCount_ReturnsReviewedTodayPerDeck(t *testing.T) {
	handler, target := newAPI(t, getLived())
	v := target[0]
	runDays(t, handler, v, 14)

	// A new day, with nothing answered in it yet.
	setNow(handler, getFirstMorning().AddDate(0, 0, 14))
	for _, one := range getVaultCount(t, handler, v).Decks {
		if one.Reviewed != 0 {
			t.Errorf("nothing was answered today and %s counts %d",
				one.Deck, one.Reviewed)
		}
	}

	// One deck answered, and only that deck counts it.
	sat, err := handler.StartSession(t.Context(), &api.StartSessionRequest{
		Vault: string(v.ID), Deck: "decks/Roots.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sat.Cards) < 3 {
		t.Fatalf("the deck offered %d review cards to answer", len(sat.Cards))
	}
	for _, card := range sat.Cards[:3] {
		if _, err := handler.RecordReview(t.Context(), &api.RecordReviewRequest{
			Vault: string(v.ID), Run: sat.Run,
			CardID: card.CardID, Template: card.Template,
			Rating: api.RatingGood, DurationMs: 6000,
		}); err != nil {
			t.Fatal(err)
		}
	}

	for _, one := range getVaultCount(t, handler, v).Decks {
		want := 0
		if one.Deck == "decks/Roots.md" {
			want = 3
		}
		if int(one.Reviewed) != want {
			t.Errorf("%s counts %d answered today, want %d",
				one.Deck, one.Reviewed, want)
		}
	}
}

// CardsDue summary includes each vault once.
func TestCardsDueSummaryContainsEachVaultOnce(t *testing.T) {
	handler, target := newAPI(t, getLived(), getDeck())
	setNow(handler, getFirstMorning())

	rows := front(t, handler).Vaults
	if len(rows) != len(target) {
		t.Fatalf("summary contains %d rows for %d vaults", len(rows), len(target))
	}
	observed := make(map[string]int, len(rows))
	for _, one := range rows {
		observed[one.ID]++
	}
	for _, v := range target {
		if observed[string(v.ID)] != 1 {
			t.Errorf("vault %s count = %d", string(v.ID), observed[string(v.ID)])
		}
	}

	// The counts are each vault's own, and the deck rows of one add up to it.
	for _, row := range rows {
		reviewCards := 0
		for _, one := range row.Decks {
			reviewCards += int(one.ReviewCards)
		}
		if reviewCards != int(row.ReviewCards) {
			t.Errorf("the decks of %s hold %d review cards and the vault holds %d",
				row.Name, reviewCards, row.ReviewCards)
		}
	}
	if got := getVaultCount(t, handler, target[0]).ReviewCards; got != 630 {
		t.Errorf("the lived vault counts %d review cards, want 630", got)
	}
	if got := getVaultCount(t, handler, target[1]).ReviewCards; got != 1 {
		t.Errorf("the vault of one card counts %d review cards", got)
	}
}

// A preset's target applies to the cards it schedules, and a computation cached
// under one target is not reused under another.
//
// A higher retention target gives shorter intervals, so the same history leaves
// more cards due today.
func TestVaultCount_RecomputesAfterRetentionChange(t *testing.T) {
	handler, target := newAPI(t, getLived())
	v := target[0]
	runDays(t, handler, v, 14)
	setNow(handler, getFirstMorning().AddDate(0, 0, 14))

	p := readPresetConfig(t, handler, v, "Grammar.md")
	p.Retention = review.GetRetentionBounds().Min
	writePreset(t, handler, v, "Grammar.md", p)
	// The first count is what writes the working out down, so the second is the
	// one that could be answered from it.
	least := countSessionCards(getVaultCount(t, handler, v), getUnderGrammar())

	p.Retention = review.GetRetentionBounds().Max
	writePreset(t, handler, v, "Grammar.md", p)
	most := countSessionCards(getVaultCount(t, handler, v), getUnderGrammar())

	if most <= least {
		t.Errorf("asking for %v of the cards back offers %d, and %v offers %d",
			review.GetRetentionBounds().Max, most, review.GetRetentionBounds().Min, least)
	}

	// The adjacent preset, whose target is unchanged, keeps its count.
	if got := countSessionCards(getVaultCount(t, handler, v), getUnderSanskrit()); got == 0 {
		t.Error("the preset nobody touched offers nothing")
	}
}
