package service

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
	"decks/Near.md": newStencilDeck("Word", "Recall", 2, 5000),
	"decks/Far.md":  newStencilDeck("Word", "", 2, 5100),
}

// The count carries how much of each deck stands learned, so the deck screen
// draws the core's own answer and works out no rule of its own.
func TestTheCountCarriesHowMuchOfADeckExistsLearned(t *testing.T) {
	api, target := newAPI(t, learnedDecks)
	v := target[0]
	setNow(api, firstMorning)

	// One card of each deck answered, so the two decks differ by their rule
	// alone and not by what was done to them.
	session := startSession(t, api, v)
	reply := map[string]bool{}
	for _, one := range session.GetCards() {
		if reply[one.GetDeck()] {
			continue
		}
		reply[one.GetDeck()] = true
		if _, err := api.AnswerCard(t.Context(), connect.NewRequest(&v1.AnswerCardRequest{
			Vault: string(v.ID), Run: session.GetRun(),
			Mark: one.GetMark(), Face: one.GetFace(),
			Rating: v1.Rating_RATING_GOOD, TookMs: 5000,
		})); err != nil {
			t.Fatal(err)
		}
	}
	setNow(api, firstMorning.Add(time.Minute))

	want := map[string]int32{"decks/Near.md": 1, "decks/Far.md": 0}
	for _, one := range getVaultCount(t, api, v).GetDecks() {
		if one.GetFaces() != 2 {
			t.Errorf("%s has %d faces, want 2", one.GetDeck(), one.GetFaces())
		}
		if one.GetLearned() != want[one.GetDeck()] {
			t.Errorf("%s has %d learned, want %d",
				one.GetDeck(), one.GetLearned(), want[one.GetDeck()])
		}
	}
}

// screen has a nought to draw.
func TestAVaultNobodyHasAnsweredCarriesNothingLearned(t *testing.T) {
	api, target := newAPI(t, learnedDecks)
	v := target[0]
	setNow(api, firstMorning)

	for _, one := range getVaultCount(t, api, v).GetDecks() {
		if one.GetLearned() != 0 {
			t.Errorf("%s has %d learned before review",
				one.GetDeck(), one.GetLearned())
		}
	}
}
