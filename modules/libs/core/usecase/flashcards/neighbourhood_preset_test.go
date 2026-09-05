package flashcards_test

import "testing"

// A deck points at the preset it is scheduled by, so that preset stands among
// the notes the deck is joined to. A preset is how the cards come round and not
// what they were written from.
func TestThePresetADeckIsScheduledByIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"presets/Steady.md": "---\ntype: preset\ngoal: retention\nretention: 0.95\n---\n" +
			"\n# Steady\n\nA slower climb through the same cards.\n",
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: \"presets/Steady\"\n    role: ref\n    type: preset\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird that sleeps flying. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 1 || got[0] != "Migration.md" {
		t.Fatalf("joined to %v", got)
	}
}

// A preset written into a note that points back at the deck is read like any
// other note: what the deck is scheduled by is what the deck's own link says.
func TestAPresetIsLeftOutOfEveryCardsPanel(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"presets/Steady.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n---\n" +
			"\n# Steady\n",
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: \"presets/Steady\"\n    role: ref\n    type: preset\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n\n### Meaning\n\nA bird.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 0 {
		t.Fatalf("joined to %v", got)
	}
}
