package flashcards_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A build that keeps nothing works the schedules out from the answers at every
// launch, and says what a build that keeps them says. The cache is a saving and
// never an answer of its own.
func TestSchedulesAreTheSameWithNothingKept(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	kept, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	bare := s.kept
	bare.Kept = nil
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
	s := opened(t, vault)
	// Both faces of one card, so the order turns on the face and not the card.
	recognise := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	name := history.CardFace{Card: "k7m2xq9fzp", Face: "Name it"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), name, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := record.Answer(t.Context(), recognise, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.kept.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	raw, err := s.kept.Kept.Read(t.Context(), s.vault.ID)
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
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	want, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.kept.Kept.Write(t.Context(), s.vault.ID, []byte("not a cache")); err != nil {
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
// with a deck of one card under each.
func targeted(high, low float64) map[string]string {
	return map[string]string{
		"Term.md": term,
		"High.md": preset(fmt.Sprintf("retention: %g\n", high)),
		"Low.md":  preset(fmt.Sprintf("retention: %g\n", low)),
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
	underHigh = history.CardFace{Card: "k7m2xq9fzp", Face: "Say it"}
	underLow  = history.CardFace{Card: "zpqrstvwxy", Face: "Say it"}
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
	moved := settings.Preset
	moved.Retention = 0.95
	if _, err := s.presets.Save(t.Context(), s.vault, "Low.md", moved, domain.FileRef{}); err != nil {
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
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, saturday).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	got, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	// The same answers, worked out by the one scheduler and nothing else.
	plain := s.kept
	plain.Standings, plain.Presets, plain.Kept = flashcards.Standings{}, flashcards.Presets{}, nil
	want, err := plain.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || !got[on].Due.Equal(want[on].Due) {
		t.Errorf("worked out %+v, want %+v", got[on], want[on])
	}
}
