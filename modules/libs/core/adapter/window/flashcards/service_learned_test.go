package flashcards

import (
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// learnedDecks is a vault of two decks: one on a preset that counts a card
// learned while it is likely to be recalled, and one naming no preset, which
// counts by the defaults and their three weeks.
var learnedDecks = map[string]string{
	"Word.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
	"Recall.md": "---\ntype: preset\ngoal: retention\nretention: 0.9\n" +
		"learned: retention\n---\n\n# Recall\n",
	"decks/Near.md": written("Word", "Recall", 2, 5000),
	"decks/Far.md":  written("Word", "", 2, 5100),
}

// The count carries how much of each deck stands learned, so the deck screen
// draws the core's own answer and works out no rule of its own.
func TestTheCountCarriesHowMuchOfADeckStandsLearned(t *testing.T) {
	api, held := windowed(t, learnedDecks)
	v := held[0]
	setNow(api, firstMorning)

	// One card of each deck answered, so the two decks differ by their rule
	// alone and not by what was done to them.
	sitting := started(t, api, v)
	answered := map[string]bool{}
	for _, one := range sitting.GetAsked() {
		if answered[one.GetDeck()] {
			continue
		}
		answered[one.GetDeck()] = true
		if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
			Vault: string(v.ID), Run: sitting.GetRun(),
			Mark: one.GetMark(), Face: one.GetFace(),
			Rating: v1.Rating_RATING_GOOD, TookMs: 5000,
		})); err != nil {
			t.Fatal(err)
		}
	}
	setNow(api, firstMorning.Add(time.Minute))

	want := map[string]int32{"decks/Near.md": 1, "decks/Far.md": 0}
	for _, one := range owing(t, api, v).GetDecks() {
		if one.GetFaces() != 2 {
			t.Errorf("%s stands at %d faces, want 2", one.GetDeck(), one.GetFaces())
		}
		if one.GetLearned() != want[one.GetDeck()] {
			t.Errorf("%s stands at %d learned, want %d",
				one.GetDeck(), one.GetLearned(), want[one.GetDeck()])
		}
	}
}

// A vault nobody has answered carries no learned face on the wire, and the
// screen has a nought to draw.
func TestAVaultNobodyHasAnsweredCarriesNothingLearned(t *testing.T) {
	api, held := windowed(t, learnedDecks)
	v := held[0]
	setNow(api, firstMorning)

	for _, one := range owing(t, api, v).GetDecks() {
		if one.GetLearned() != 0 {
			t.Errorf("%s stands at %d learned before anything was answered",
				one.GetDeck(), one.GetLearned())
		}
	}
}
