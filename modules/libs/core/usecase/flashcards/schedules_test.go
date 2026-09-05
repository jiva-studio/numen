package flashcards_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A build that keeps nothing works the schedules out from the answers at every
// launch, and says what a build that keeps them says. The cache is a saving and
// never an answer of its own.
func TestSchedulesAreTheSameWithNothingKept(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	kept, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	bare := s.kept
	bare.Cache = nil
	worked, err := bare.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	if len(worked) != len(kept) {
		t.Fatalf("kept %d schedules, worked out %d", len(kept), len(worked))
	}
	if !worked[on].Due.Equal(kept[on].Due) || worked[on].Reps != kept[on].Reps {
		t.Errorf("worked out %+v, kept %+v", worked[on], kept[on])
	}
}

// The cache is written in one order whatever order the schedules were worked
// out in, so two machines that answered the same cards hold the same file and a
// synchroniser has nothing to reconcile.
func TestTheCacheIsWrittenInOneOrder(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	// Both faces of one card, so the order turns on the face and not the card.
	recognise := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	name := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Name it"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), name, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := record.Answer(t.Context(), recognise, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.kept.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	raw, err := s.kept.Cache.Read(t.Context(), s.vault.ID)
	if err != nil {
		t.Fatal(err)
	}
	var was plantedCache
	if err := json.Unmarshal(raw, &was); err != nil {
		t.Fatal(err)
	}

	if len(was.Faces) != 2 {
		t.Fatalf("the cache holds %d faces, want the two answered", len(was.Faces))
	}
	if was.Faces[0].Face != "Name it" || was.Faces[1].Face != "Recognise" {
		t.Errorf("the cache holds %s then %s", was.Faces[0].Face, was.Faces[1].Face)
	}
}

// A cache nothing can read is nothing remembered: the answers are there, and
// the schedules are worked out from them again.
func TestACacheNothingCanReadIsWorkedOutAgain(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	want, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.kept.Cache.Write(t.Context(), s.vault.ID, []byte("not a cache")); err != nil {
		t.Fatal(err)
	}

	got, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a cache nothing can read refused the schedules: %v", err)
	}
	if !got[on].Due.Equal(want[on].Due) {
		t.Errorf("worked out %+v, want %+v", got[on], want[on])
	}
}

// targeted is a vault of two presets asking for different shares of the cards,
// with a deck of one card under each. Neither evens its days out, so what
// separates two cards here is the target each stands under.
func targeted(high, low float64) map[string]string {
	return map[string]string{
		"Term.md": term,
		"High.md": preset(fmt.Sprintf("retention: %g\neven_load: false\n", high)),
		"Low.md":  preset(fmt.Sprintf("retention: %g\neven_load: false\n", low)),
		"decks/High.md": "---\ntype: deck\nlinks:\n" +
			"  - to: High\n    role: ref\n    type: preset\n---\n" +
			"\n## One ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nbhu\n\n### Meaning\n\nto be\n",
		"decks/Low.md": "---\ntype: deck\nlinks:\n" +
			"  - to: Low\n    role: ref\n    type: preset\n---\n" +
			"\n## Two ^zpqrstvwxy\n\n[[Term]]\n\n### Word\n\ngam\n\n### Meaning\n\nto go\n",
	}
}

// The two card faces of the targeted vault, answered alike.
var (
	underHigh = review.CardFaceID{Card: "k7m2xq9fzp", Face: "Say it"}
	underLow  = review.CardFaceID{Card: "zpqrstvwxy", Face: "Say it"}
)

// answeredAlike takes both cards through the same answers at the same moments,
// far enough for each to be learned, so what separates their schedules is the
// preset each stands under.
func answeredAlike(t *testing.T, s vaulted) {
	t.Helper()
	for day := range 3 {
		record := s.run(t, saturday.AddDate(0, 0, day-3))
		answer(t, record, underHigh.Card, 6*time.Second)
		answer(t, record, underLow.Card, 6*time.Second)
	}
}

// A card is scheduled at the share of the cards its own preset asks for, so two
// presets asking for different shares send the same answer different distances.
func TestEachPresetSchedulesItsCardsAtItsOwnTarget(t *testing.T) {
	t.Parallel()
	s := opened(t, targeted(0.95, 0.75))
	answeredAlike(t, s)

	got, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	high, low := got[underHigh], got[underLow]
	if high.Due.IsZero() || low.Due.IsZero() {
		t.Fatalf("the two cards came to %+v and %+v", high, low)
	}
	// More of the cards asked back is shorter intervals.
	if !high.Due.Before(low.Due) {
		t.Errorf("asking for 0.95 comes round at %v and 0.75 at %v", high.Due, low.Due)
	}
}

// Moving one preset's target throws the cache away and works the schedules out
// again, so the cards under it come round somewhere else.
func TestMovingATargetWorksTheSchedulesOutAgain(t *testing.T) {
	t.Parallel()
	s := opened(t, targeted(0.95, 0.75))
	answeredAlike(t, s)

	was, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	settings, err := s.presets.Read(t.Context(), s.vault, "Low.md")
	if err != nil {
		t.Fatal(err)
	}
	moved := settings.Settings
	moved.Retention = 0.95
	if _, err := s.presets.Save(t.Context(), s.vault, "Low.md", moved, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	now, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if now[underLow].Due.Equal(was[underLow].Due) {
		t.Errorf("the target moved and the card still comes round at %v", now[underLow].Due)
	}
	// The two now ask for the same share, so they come round together.
	if !now[underLow].Due.Equal(now[underHigh].Due) {
		t.Errorf("two presets at one target come round at %v and %v",
			now[underLow].Due, now[underHigh].Due)
	}
}

// Pointing a deck at the other preset works the schedules out again. The two
// targets in force did not move; which card stands under which did, and that is
// what the cache is held against.
func TestRepointingADeckWorksTheSchedulesOutAgain(t *testing.T) {
	t.Parallel()
	files := targeted(0.95, 0.75)
	s := opened(t, files)
	answeredAlike(t, s)

	was, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if was[underHigh].Due.Equal(was[underLow].Due) {
		t.Fatalf("the two cards stand at one day, %v", was[underHigh].Due)
	}

	// The decks swap presets, so the card that asked for 0.95 now asks for 0.75
	// and the other way about.
	write(t, s, "decks/High.md", strings.Replace(files["decks/High.md"], "to: High", "to: Low", 1))
	write(t, s, "decks/Low.md", strings.Replace(files["decks/Low.md"], "to: Low", "to: High", 1))

	now, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if !now[underHigh].Due.Equal(was[underLow].Due) || !now[underLow].Due.Equal(was[underHigh].Due) {
		t.Errorf("the decks swapped presets and the cards come round at %v and %v, want %v and %v",
			now[underHigh].Due, now[underLow].Due, was[underLow].Due, was[underHigh].Due)
	}
}

// A card moved into a deck under another preset is scheduled by that preset.
// Both decks go on holding cards and go on pointing where they pointed, so the
// two targets in force are the two that were in force.
func TestACardMovedToAnotherDeckIsScheduledByItsPreset(t *testing.T) {
	t.Parallel()
	// The card that moves, written so that it can be cut from one deck and
	// pasted into the other.
	moving := "\n## One ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nbhu\n\n### Meaning\n\nto be\n"
	files := targeted(0.95, 0.75)
	files["decks/High.md"] += "\n## Stays ^card000001\n\n[[Term]]\n\n### Word\n\nkr\n" +
		"\n### Meaning\n\nto do\n"
	s := opened(t, files)
	answeredAlike(t, s)

	was, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	write(t, s, "decks/High.md", strings.Replace(files["decks/High.md"], moving, "", 1))
	write(t, s, "decks/Low.md", files["decks/Low.md"]+moving)

	now, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if !now[underHigh].Due.Equal(was[underLow].Due) {
		t.Errorf("the card moved under 0.75 and comes round at %v, want %v",
			now[underHigh].Due, was[underLow].Due)
	}
}

// A vault whose decks name no preset is scheduled as it always was: the
// defaults are the target the whole vault stood at.
func TestAVaultOfNoPresetsIsScheduledAsItWas(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, saturday).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	got, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	// The same answers, worked out by the one scheduler and nothing else.
	plain := s.kept
	plain.Cache = nil
	plain.CardFaces = flashcards.NewListCardFaces(nil, nil, nil)
	plain.Presets = flashcards.NewPresets(nil, nil, nil, nil, today, time.Now)
	want, err := plain.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || !got[on].Due.Equal(want[on].Due) {
		t.Errorf("worked out %+v, want %+v", got[on], want[on])
	}
}

// The day a card comes back on is one answer, whether it is answered or
// projected. A preset evening its days out moves the card off the day carrying
// none of the load, and the sitting and the picture move it to the same one.
//
// The card is answered twice at the hour the day opens: a card answered no time
// at all since its last answer is one the projection is certain came back, so
// the two work the same interval out and what is compared is where it is put.
func TestAnAnsweredCardAndAProjectedOneLandOnOneDay(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 9, 7, 4, 0, 0, 0, time.Local)
	by := review.NewFSRSAt(0.9)
	begun := by.Next(review.Schedule{}, when, review.Good)
	fell := by.Next(begun, when, review.Good).Due
	if away := fell.Sub(when).Hours() / 24; away < review.EvenFrom {
		t.Fatalf("the second answer sends the card %g days away", away)
	}

	s := opened(t, map[string]string{
		"Term.md": term,
		"Even.md": preset(fmt.Sprintf(
			"new_a_day: 0\nreviews_a_day: 9999\nretention: 0.9\neven_load: true\nload: {%s: 0}\n",
			review.DayName(fell.Weekday()))),
		"decks/Even.md": deckNaming([]string{"Even"}, 1, 0),
	})
	on := review.CardFaceID{Card: mark(0), Face: "Say it"}
	read, err := s.presets.Read(t.Context(), s.vault, "Even.md")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.run(t, when).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	first, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	// The projection takes the card on from where the first answer left it, and
	// its one day of review is the day that answer is given in.
	run := review.Simulation{By: by, Day: today, Cost: review.DefaultCost, Days: 21}
	projected, err := run.Run(t.Context(), when, read.Settings, first, 0)
	if err != nil {
		t.Fatal(err)
	}
	comes := 0
	for day, load := range projected.Load {
		if day > 0 && load > 0 {
			comes = day
			break
		}
	}

	if _, err := s.run(t, when).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	answered, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	stands := dayFrom(when, answered[on].Due)

	if comes == 0 || comes != stands {
		t.Errorf("the projection has the card back on day %d and the answer on day %d",
			comes, stands)
	}
	if stands == dayFrom(when, fell) {
		t.Errorf("the card stood on the %v it fell on", fell.Weekday())
	}
}

// dayFrom is how many days of review stand between the one holding from and the
// one holding at.
func dayFrom(from, at time.Time) int {
	open := from
	for i := range 400 {
		if at.Before(today.Ends(open)) {
			return i
		}
		open = today.Ends(open)
	}
	return -1
}

// A preset keeping no even load leaves a card where the scheduler puts it, so a
// card falls on a day carrying none of the load. Nothing is shown there: the
// card stands overdue, and the next day picks it up.
func TestACardFallingOnADayAtNoneOfTheLoadStandsOver(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 9, 7, 4, 0, 0, 0, time.Local)
	by := review.NewFSRSAt(0.9)
	begun := by.Next(review.Schedule{}, when, review.Good)
	fell := by.Next(begun, when, review.Good).Due

	s := opened(t, map[string]string{
		"Term.md": term,
		"Even.md": preset(fmt.Sprintf(
			"new_a_day: 0\nreviews_a_day: 9999\nretention: 0.9\neven_load: false\nload: {%s: 0}\n",
			review.DayName(fell.Weekday()))),
		"decks/Even.md": deckNaming([]string{"Even"}, 1, 0),
	})
	on := review.CardFaceID{Card: mark(0), Face: "Say it"}

	for range 2 {
		if _, err := s.run(t, when).Answer(t.Context(), on, review.Good, 0); err != nil {
			t.Fatal(err)
		}
	}
	answered, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got := answered[on].Due; !got.Equal(fell) {
		t.Fatalf("the card comes round at %v, and the scheduler put it at %v", got, fell)
	}

	opens := today.Ends(fell).AddDate(0, 0, -1)
	if asked := s.sittingAt(t, today, opens.Add(6*time.Hour)).Queue; len(asked) != 0 {
		t.Errorf("a %v carrying none of the load asked %d cards", fell.Weekday(), len(asked))
	}
	after := today.Ends(fell).Add(6 * time.Hour)
	if asked := s.sittingAt(t, today, after).Queue; len(asked) != 1 {
		t.Errorf("the day after asked %d cards, want the one standing over", len(asked))
	}
}
