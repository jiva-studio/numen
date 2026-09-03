package flashcards_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A vault of three decks: two scheduled by a preset that gives Saturday half
// the load, and one naming no preset at all.
var scheduled = map[string]string{
	"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
		"new_a_day: 8\nreviews_a_day: 45\nload: {sat: 50}\n---\n\n# Sanskrit\n",
	"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n" +
		"\n## Root ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nbhu\n\n### Meaning\n\nto be\n",
	"decks/Mantras.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n" +
		"\n## Gayatri ^zpqrstvwxy\n\n[[Term]]\n\n### Word\n\ngayatri\n\n### Meaning\n\na metre\n",
	"decks/Terms.md": "---\ntype: deck\n---\n" +
		"\n## Term ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nsutra\n\n### Meaning\n\na thread\n",
}

// saturday is a day the Sanskrit preset gives half the load, at an hour well
// inside it.
var saturday = time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local)

// What a day came to is counted under the preset each deck names, over as many
// sittings as the day held. A deck naming no preset comes under the defaults.
func TestWhatADayCameToUnderEachPresetOfAVault(t *testing.T) {
	t.Parallel()
	s := opened(t, scheduled)

	// Two sittings of the one day, each writing a file of its own.
	morning := s.run(t, saturday)
	answer(t, morning, "k7m2xq9fzp", 6*time.Second)
	evening := s.run(t, saturday.Add(9*time.Hour))
	answer(t, evening, "zpqrstvwxy", 9*time.Second)
	answer(t, evening, "3f4g5h6j7k", 4*time.Second)

	owing, err := s.owedAt(today, func() time.Time { return saturday.Add(10 * time.Hour) }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	want := []flashcards.PresetOwing{
		{
			Preset: "", Decks: 1, Cards: 1, Due: 1,
			Answered: 1, AnsweredNew: 1, Took: 4 * time.Second,
			Budget: history.Budget{New: 10, Reviews: 200, Minutes: 20},
			Closes: history.Closes{
				Minutes: history.ClosedMinutes, Backlog: history.ClosedBacklog,
			},
		},
		{
			Preset: "Sanskrit.md", Decks: 2, Cards: 2, Due: 2,
			Answered: 2, AnsweredNew: 2, Took: 15 * time.Second,
			Budget: history.Budget{New: 4, Reviews: 23, Minutes: 10},
			Closes: history.Closes{
				Minutes: history.ClosedMinutes, Backlog: history.ClosedBacklog,
			},
		},
	}
	if len(owing.Presets) != len(want) {
		t.Fatalf("the day came to %+v, want %+v", owing.Presets, want)
	}
	for at, one := range want {
		if owing.Presets[at] != one {
			t.Errorf("%q came to %+v, want %+v", one.Preset, owing.Presets[at], one)
		}
	}
}

// A vault holding no preset at all is one scope: the defaults, with every deck
// under them.
func TestAVaultHoldingNoPresetStandsOnTheDefaults(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":      term,
		"decks/One.md": deckOf("", 20, 0),
		"decks/Two.md": deckOf("", 20, 100),
	})

	owing, err := s.owedAt(today, func() time.Time { return saturday }).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	want := flashcards.PresetOwing{
		Preset: "", Decks: 2, Cards: 40, New: 40,
		Budget: history.Defaults().Admits(today, saturday, history.Spent{}, history.Left{}).Keeps,
		Closes: history.Defaults().Admits(today, saturday, history.Spent{}, history.Left{}).Closes,
	}
	if len(owing.Presets) != 1 {
		t.Fatalf("the vault came to %+v, want the defaults alone", owing.Presets)
	}
	if owing.Presets[0] != want {
		t.Errorf("the defaults came to %+v, want %+v", owing.Presets[0], want)
	}
}

// Every preset the vault holds stands in the count. A person who wrote one and
// pointed nothing at it can still see it, and it says nothing of a day.
func TestAPresetNoDeckPointsAtStandsInTheCount(t *testing.T) {
	t.Parallel()
	files := make(map[string]string, len(scheduled)+1)
	for path, raw := range scheduled {
		files[path] = raw
	}
	files["Empty.md"] = "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 137\n---\n\n# Empty\n"
	s := opened(t, files)

	owing, err := s.owedAt(today, func() time.Time { return saturday }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	var held *flashcards.PresetOwing
	for at, one := range owing.Presets {
		if one.Preset == "Empty.md" {
			held = &owing.Presets[at]
		}
	}
	if held == nil {
		t.Fatalf("the preset nothing points at is not in the count: %+v", owing.Presets)
	}
	if *held != (flashcards.PresetOwing{Preset: "Empty.md"}) {
		t.Errorf("it came to %+v, want a preset nothing stands under", *held)
	}
}

// What was answered on another day is not what today came to, and neither is an
// answer taken back.
func TestADayHoldsWhatWasAnsweredInIt(t *testing.T) {
	t.Parallel()
	s := opened(t, scheduled)

	before := s.run(t, saturday.AddDate(0, 0, -1))
	answer(t, before, "k7m2xq9fzp", 6*time.Second)

	sitting := s.run(t, saturday)
	given := answer(t, sitting, "zpqrstvwxy", 9*time.Second)
	if _, err := sitting.TakeBack(t.Context(), given); err != nil {
		t.Fatal(err)
	}

	owing, err := s.owedAt(today, func() time.Time { return saturday.Add(time.Hour) }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range owing.Presets {
		if one.Answered != 0 || one.Took != 0 {
			t.Errorf("%q came to %+v, want a day nothing stands on", one.Preset, one)
		}
	}
}

// What a day came to is counted deck by deck as well as preset by preset, so a
// deck nobody answered today is not carried by the one beside it under the same
// preset.
func TestWhatEachDeckWasAnsweredIsCountedOnTheDeck(t *testing.T) {
	t.Parallel()
	s := opened(t, scheduled)

	// One card of one of the two decks the Sanskrit preset schedules.
	answer(t, s.run(t, saturday), "k7m2xq9fzp", 6*time.Second)

	owing, err := s.owedAt(today, func() time.Time { return saturday.Add(time.Hour) }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]int{"decks/Roots.md": 1, "decks/Mantras.md": 0, "decks/Terms.md": 0}
	for _, one := range owing.Decks {
		if one.Answered != want[one.Deck] {
			t.Errorf("%s was answered %d today, want %d", one.Deck, one.Answered, want[one.Deck])
		}
	}
	for _, one := range owing.Presets {
		if one.Preset == "Sanskrit.md" && one.Answered != 1 {
			t.Errorf("the preset of the two decks was answered %d today, want 1", one.Answered)
		}
	}
}

// The front door counts out of the cache it filled.
//
// It is asked for every vault a person holds, and again whenever the window
// opens, a sitting ends or a vault moves, so a count over answers nothing has
// changed replays nothing.
func TestASecondCountReadsTheSchedulesOutOfTheCache(t *testing.T) {
	t.Parallel()
	s := opened(t, scheduled)
	answer(t, s.run(t, saturday), "k7m2xq9fzp", 6*time.Second)

	replayed := 0
	s.kept.At = func(retention float64) history.Scheduler {
		return replaying{Scheduler: history.NewFSRSAt(retention), answers: &replayed}
	}
	owed := s.owedAt(today, func() time.Time { return saturday.Add(time.Hour) })

	if _, err := owed.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	first := replayed
	if first == 0 {
		t.Fatal("the first count replayed nothing, and there is no cache to have filled")
	}
	if _, err := owed.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	if replayed != first {
		t.Errorf("the second count replayed %d answers, want none", replayed-first)
	}
}

// replaying is a scheduler saying how many answers were worked out through it.
type replaying struct {
	history.Scheduler
	answers *int
}

func (r replaying) Next(
	s history.Schedule, at time.Time, rating history.Rating,
) history.Schedule {
	*r.answers++
	return r.Scheduler.Next(s, at, rating)
}

// answer writes down one card answered well, and hands back the line it stands
// as. Every card of this vault is shown through the one face.
func answer(t *testing.T, record flashcards.Record, card string, took time.Duration) string {
	t.Helper()
	return said(t, record, card, history.Good, took)
}

// again writes down one card the person could not recall, which comes round
// again in the same sitting.
func again(t *testing.T, record flashcards.Record, card string, took time.Duration) string {
	t.Helper()
	return said(t, record, card, history.Again, took)
}

func said(
	t *testing.T, record flashcards.Record, card string,
	rating history.Rating, took time.Duration,
) string {
	t.Helper()
	given, err := record.Answer(
		t.Context(), history.CardFace{Card: card, Face: "Say it"}, rating, took)
	if err != nil {
		t.Fatal(err)
	}
	return given.ID
}

// A preset a deck names is pointed at whatever the deck holds, and how many
// cards stand under it is counted beside that.
func TestAnEmptyDeckStillPointsAtItsPreset(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Empty.md":       preset("new_a_day: 4\nreviews_a_day: 20\n"),
		"Unnamed.md":     preset("new_a_day: 4\nreviews_a_day: 20\n"),
		"decks/Empty.md": deckOf("Empty", 0, 0),
		"decks/Full.md":  deckOf("Empty", 3, 0),
	})

	owing, err := s.owedAt(today, func() time.Time { return saturday }).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string][2]int, len(owing.Presets))
	for _, one := range owing.Presets {
		got[one.Preset] = [2]int{one.Decks, one.Cards}
	}
	want := map[string][2]int{"Empty.md": {2, 3}, "Unnamed.md": {0, 0}}
	for path, one := range want {
		if got[path] != one {
			t.Errorf("%s is named by %d decks holding %d cards, want %d and %d",
				path, got[path][0], got[path][1], one[0], one[1])
		}
	}
}

// A preset nothing but an empty deck names is still named by that deck.
func TestAPresetOnlyAnEmptyDeckNamesIsPointedAt(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Empty.md":       preset("new_a_day: 4\nreviews_a_day: 20\n"),
		"decks/Empty.md": deckOf("Empty", 0, 0),
	})

	owing, err := s.owedAt(today, func() time.Time { return saturday }).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range owing.Presets {
		if one.Preset != "Empty.md" {
			continue
		}
		if one.Decks != 1 || one.Cards != 0 {
			t.Errorf("the preset is named by %d decks holding %d cards, want 1 and 0",
				one.Decks, one.Cards)
		}
		return
	}
	t.Error("the preset was not among what the vault holds")
}

// A count is dropped once the window that asked for it has gone. Counting a
// vault reads every deck in it and replays its whole answer log, and nobody is
// waiting for either.
//
// The scenario here holds nothing to read a vault through, so nothing past the
// first check can run.
func TestACountIsDroppedOnceTheWindowHasGone(t *testing.T) {
	t.Parallel()
	gone, went := context.WithCancel(t.Context())
	went()

	_, err := flashcards.Owed{}.Execute(gone, domain.Vault{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the count came back with %v", err)
	}
}
