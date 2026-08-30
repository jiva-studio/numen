package flashcards_test

import (
	"reflect"
	"strings"
	"testing"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// A vault where one deck names a preset, one names an ordinary note, and one
// names nothing.
var pointing = map[string]string{
	"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
		"new_a_day: 8\nreviews_a_day: 45\nretention: 0.87\nlight_days: [sat]\n---\n\n# Sanskrit\n",
	"Grammar.md": "---\ntype: note\n---\n\n# Grammar\n",
	"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	"decks/Mantras.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Grammar\n    role: ref\n    type: preset\n---\n\n## Gayatri ^zpqrstvwxy\n",
	"decks/Terms.md": "---\ntype: deck\n---\n\n## Term ^3f4g5h6j7k\n",
}

// A deck is scheduled by the preset its link names.
func TestADeckIsScheduledByThePresetItNames(t *testing.T) {
	s := opened(t, pointing)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "Sanskrit.md" {
		t.Errorf("read from %q", held.Path)
	}
	if len(held.Problems) != 0 {
		t.Errorf("problems = %v", held.Problems)
	}
	if held.Preset.MinutesADay != 20 || held.Preset.NewADay != 8 || held.Preset.ReviewsADay != 45 {
		t.Errorf("preset = %+v", held.Preset)
	}
}

// A deck naming no preset is scheduled by the defaults.
func TestADeckNamingNoPreset(t *testing.T) {
	s := opened(t, pointing)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "" {
		t.Errorf("read from %q", held.Path)
	}
	if !reflect.DeepEqual(held.Preset, history.Defaults()) {
		t.Errorf("preset = %+v", held.Preset)
	}
}

// A link reaching a note that is not a preset leaves the deck on the defaults
// and says so against it.
func TestADeckNamingANoteThatIsNotAPreset(t *testing.T) {
	s := opened(t, pointing)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Mantras.md")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(held.Preset, history.Defaults()) {
		t.Errorf("preset = %+v", held.Preset)
	}
	if len(held.Problems) != 1 || !strings.Contains(held.Problems[0], "not a preset") {
		t.Errorf("problems = %v", held.Problems)
	}
}

// A deck naming two presets is scheduled by the first and carries a problem.
func TestADeckNamingTwoPresets(t *testing.T) {
	notes := map[string]string{
		"Sanskrit.md": "---\ntype: preset\nnew_a_day: 8\n---\n\n# Sanskrit\n",
		"Mantras.md":  "---\ntype: preset\nnew_a_day: 3\n---\n\n# Mantras\n",
		"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
			"  - to: Sanskrit\n    role: ref\n    type: preset\n" +
			"  - to: Mantras\n    role: ref\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	}
	s := opened(t, notes)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Preset.NewADay != 8 {
		t.Errorf("new a day = %d", held.Preset.NewADay)
	}
	if len(held.Problems) != 1 || !strings.Contains(held.Problems[0], "more than one preset") {
		t.Errorf("problems = %v", held.Problems)
	}
}
