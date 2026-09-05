package flashcards_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// A vault of two presets and three decks: one on a preset, one on none, and one
// on a note that is not a preset.
var choosing = map[string]string{
	"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n---\n\n# Sanskrit\n",
	"presets/Slow.md": "---\ntype: preset\ntitle: Slow going\ngoal: retention\n" +
		"retention: 0.8\n---\n\n# Slow going\n",
	"Grammar.md": "---\ntype: note\n---\n\n# Grammar\n",
	"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	"decks/Terms.md": "---\ntype: deck\n---\n\n## Term ^3f4g5h6j7k\n",
	"decks/Verses.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Grammar\n    role: ref\n    type: preset\n---\n\n## Verse ^zpqrstvwxy\n",
}

// Every preset the vault holds is listed, by path and by what it is called.
func TestThePresetsOfAVaultAreListed(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)

	held, err := s.presets.List(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 2 {
		t.Fatalf("listed %+v", held)
	}
	if held[0].Path != "Sanskrit.md" || held[0].Title != "Sanskrit" {
		t.Errorf("the first is %+v", held[0])
	}
	if held[1].Path != "presets/Slow.md" || held[1].Title != "Slow going" {
		t.Errorf("the second is %+v", held[1])
	}
}

// A vault holding no preset lists none. Every deck in it is scheduled by the
// defaults, which are no note.
func TestAVaultOfNoPresetsListsNone(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)

	held, err := s.presets.List(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 0 {
		t.Errorf("listed %+v", held)
	}
}

// A build that cannot ask which notes are presets says so, rather than
// answering the list a vault holding none would get. The two are read the same
// way on the screen, and only one of them is a fact about the vault.
func TestABuildThatCannotReachThePresetsRefusesToListThem(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)

	blind := s.presets
	blind.Notes = nil
	if _, err := blind.List(t.Context(), s.vault); !errors.Is(err, flashcards.ErrNoPresets) {
		t.Errorf("listing the presets of a vault it cannot reach answered %v", err)
	}
}

// A deck that named no preset names one, and is scheduled by it afterwards.
func TestADeckIsPutOnAPreset(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Terms.md", "Sanskrit.md", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	got := read(t, s.vault, "decks/Terms.md")
	if !strings.Contains(got, "to: Sanskrit\n") || !strings.Contains(got, "type: preset\n") {
		t.Errorf("the deck does not name the preset:\n%s", got)
	}
	if !strings.Contains(got, "## Term ^3f4g5h6j7k\n") {
		t.Errorf("the card was not left alone:\n%s", got)
	}

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "Sanskrit.md" || held.Settings.MinutesADay != 20 {
		t.Errorf("the deck is scheduled by %q at %+v", held.Path, held.Settings)
	}
}

// A deck already on a preset is put on another, and names one preset
// afterwards.
func TestADeckIsMovedFromOnePresetToAnother(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "presets/Slow.md", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	got := read(t, s.vault, "decks/Roots.md")
	if strings.Contains(got, "to: Sanskrit") {
		t.Errorf("the deck still names the preset it was moved off:\n%s", got)
	}
	if strings.Count(got, "type: preset") != 1 {
		t.Errorf("the deck names more than one preset:\n%s", got)
	}

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "presets/Slow.md" {
		t.Errorf("the deck is scheduled by %q", held.Path)
	}
	if len(held.Problems) != 0 {
		t.Errorf("problems = %v", held.Problems)
	}
}

// A deck is taken off its preset, and is scheduled by the defaults again. The
// `links:` block goes with the entry it held.
func TestADeckIsTakenOffItsPreset(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	got := read(t, s.vault, "decks/Roots.md")
	if strings.Contains(got, "links:") || strings.Contains(got, "preset") {
		t.Errorf("the deck still names a preset:\n%s", got)
	}

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "" {
		t.Errorf("the deck is scheduled by %q", held.Path)
	}
}

// Everything the deck's frontmatter holds beside the one entry comes out of the
// write as the bytes it went in as.
func TestPointingADeckLeavesTheRestOfTheFrontmatter(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Sanskrit.md": choosing["Sanskrit.md"],
		"decks/Roots.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: deck\n" +
			"tags: [grammar, roots]\n# the ones I keep coming back to\n" +
			"links:\n  - to: Grammar\n    role: parent\n    note: where these come from\n" +
			"---\n\n## Root ^k7m2xq9fzp\n",
	})

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "Sanskrit.md", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	got := read(t, s.vault, "decks/Roots.md")
	for _, kept := range []string{
		"id: 01J8F3K2M9QRSTVWXYZ012\n", "tags: [grammar, roots]\n",
		"# the ones I keep coming back to\n",
		"  - to: Grammar\n    role: parent\n    note: where these come from\n",
	} {
		if !strings.Contains(got, kept) {
			t.Errorf("%q was written over in\n%s", kept, got)
		}
	}
	if !strings.Contains(got, "type: preset\n") {
		t.Errorf("the preset was not written to\n%s", got)
	}
}

// What the person wrote on the entry stays on it when the deck is moved to
// another preset.
func TestMovingADeckKeepsWhatThePersonWroteOnTheEntry(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Sanskrit.md":     choosing["Sanskrit.md"],
		"presets/Slow.md": choosing["presets/Slow.md"],
		"decks/Roots.md": "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n" +
			"    type: preset\n    note: twenty minutes is as much as I have\n---\n\n## Root ^k7m2xq9fzp\n",
	})

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "presets/Slow.md", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	got := read(t, s.vault, "decks/Roots.md")
	if !strings.Contains(got, "note: twenty minutes is as much as I have") {
		t.Errorf("the person's words are gone from\n%s", got)
	}
	if !strings.Contains(got, "to: Slow") {
		t.Errorf("the deck does not name the preset it was moved to:\n%s", got)
	}
}

// A note that is not a preset schedules nothing, so a deck is not pointed at
// one and the file is left as it stands.
func TestADeckIsNotPointedAtANoteThatIsNotAPreset(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)
	was := read(t, s.vault, "decks/Terms.md")

	_, err := s.presets.Point(t.Context(), s.vault, "decks/Terms.md", "Grammar.md", domain.Fingerprint{})
	if !errors.Is(err, flashcards.ErrNotAPreset) {
		t.Fatalf("err = %v", err)
	}
	if got := read(t, s.vault, "decks/Terms.md"); got != was {
		t.Errorf("the deck was written\n was %q\n got %q", was, got)
	}
}

// A path the vault holds no note at is refused, and nothing is written.
func TestADeckIsNotPointedAtANoteThatIsNotThere(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)
	was := read(t, s.vault, "decks/Terms.md")

	_, err := s.presets.Point(t.Context(), s.vault, "decks/Terms.md", "Pali.md", domain.Fingerprint{})
	if !errors.Is(err, note.ErrNoNote) {
		t.Fatalf("err = %v", err)
	}
	if got := read(t, s.vault, "decks/Terms.md"); got != was {
		t.Errorf("the deck was written\n was %q\n got %q", was, got)
	}
}

// A deck the person has edited since the caller read it is left alone.
func TestADeckThatChangedSinceItWasReadIsNotPointed(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)

	at, err := s.presets.Point(
		t.Context(), s.vault, "decks/Terms.md", "Sanskrit.md", domain.Fingerprint{})
	if err != nil {
		t.Fatal(err)
	}
	was := read(t, s.vault, "decks/Terms.md")

	stale := at
	stale.Size += 3
	_, err = s.presets.Point(t.Context(), s.vault, "decks/Terms.md", "presets/Slow.md", stale)
	if !errors.Is(err, port.ErrChanged) {
		t.Fatalf("err = %v", err)
	}
	if got := read(t, s.vault, "decks/Terms.md"); got != was {
		t.Errorf("the deck was written\n was %q\n got %q", was, got)
	}
}

// The fingerprint a write answers with is the one the next write is held to.
func TestTheFingerprintAPointAnswersWithIsPresentedAgain(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)

	at, err := s.presets.Point(
		t.Context(), s.vault, "decks/Terms.md", "Sanskrit.md", domain.Fingerprint{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Terms.md", "presets/Slow.md", at); err != nil {
		t.Fatal(err)
	}

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "presets/Slow.md" {
		t.Errorf("the deck is scheduled by %q", held.Path)
	}
}

// The whole frontmatter comes back as it went in but for the one entry: the
// identity, the keys the application does not own, the comments beside them,
// the block's own indentation and the order of the entries.
func TestPointingADeckWritesOneEntryAndNothingElse(t *testing.T) {
	t.Parallel()
	deck := "---\n" +
		"id: 01J8F3K2M9QRSTVWXYZ012\n" +
		"type: deck\n" +
		"# the ones I keep coming back to\n" +
		"tags: [grammar, roots]   # mine\n" +
		"links:\n" +
		"    - to: Grammar\n" +
		"      role: parent\n" +
		"    - to: Sanskrit\n" +
		"      role: ref\n" +
		"      type: preset\n" +
		"    - to: Verses\n" +
		"      role: jump\n" +
		"---\n\n## Root ^k7m2xq9fzp\n"
	for name, one := range map[string]struct{ preset, want string }{
		"moved to another preset": {"presets/Slow.md", "    - to: Slow\n      role: ref\n      type: preset\n"},
		"taken off its preset":    {"", ""},
	} {
		t.Run(name, func(t *testing.T) {
			s := opened(t, map[string]string{
				"Sanskrit.md":     choosing["Sanskrit.md"],
				"presets/Slow.md": choosing["presets/Slow.md"],
				"decks/Roots.md":  deck,
			})

			if _, err := s.presets.Point(
				t.Context(), s.vault, "decks/Roots.md", one.preset, domain.Fingerprint{}); err != nil {
				t.Fatal(err)
			}

			want := strings.Replace(deck,
				"    - to: Sanskrit\n      role: ref\n      type: preset\n", one.want, 1)
			if got := read(t, s.vault, "decks/Roots.md"); got != want {
				t.Errorf("the deck was written\n want %q\n  got %q", want, got)
			}
		})
	}
}

// A `links:` block left with nothing in it is taken out with the entry, and the
// note is read back as the deck it is.
func TestABlockLeftEmptyIsTakenOutWithTheEntry(t *testing.T) {
	t.Parallel()
	s := opened(t, choosing)

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	got := read(t, s.vault, "decks/Roots.md")
	if strings.Contains(got, "links:") {
		t.Errorf("the block was left standing with nothing in it:\n%s", got)
	}
	if _, err := markdown.Open([]byte(got)); err != nil {
		t.Fatalf("the note cannot be read back: %v\n%s", err, got)
	}
	n := markdown.Parse(domain.Fingerprint{Path: "decks/Roots.md"}, []byte(got))
	if n.Type != domain.TypeDeck {
		t.Errorf("the note reads as a %s", n.Type)
	}
}

// A block holding other entries keeps them when the preset entry goes.
func TestTakingADeckOffItsPresetKeepsItsOtherLinks(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Sanskrit.md": choosing["Sanskrit.md"],
		"Grammar.md":  choosing["Grammar.md"],
		"decks/Roots.md": "---\ntype: deck\nlinks:\n  - to: Grammar\n    role: parent\n" +
			"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	})

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	got := read(t, s.vault, "decks/Roots.md")
	if !strings.Contains(got, "  - to: Grammar\n    role: parent\n") {
		t.Errorf("the other link went with it:\n%s", got)
	}
	if strings.Contains(got, "type: preset") {
		t.Errorf("the deck still names a preset:\n%s", got)
	}
}

// A deck that already names two presets is a problem against it. Choosing one
// leaves it naming one, which is what settles the problem.
func TestADeckNamingTwoPresetsIsLeftNamingOne(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Sanskrit.md":     choosing["Sanskrit.md"],
		"presets/Slow.md": choosing["presets/Slow.md"],
		"decks/Roots.md": "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n" +
			"  - to: Slow\n    role: ref\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	})

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "presets/Slow.md", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if got := read(t, s.vault, "decks/Roots.md"); strings.Count(got, "type: preset") != 1 {
		t.Errorf("the deck still names more than one preset:\n%s", got)
	}
	held, err := s.presets.Of(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "presets/Slow.md" || len(held.Problems) != 0 {
		t.Errorf("the deck is scheduled by %q with %v", held.Path, held.Problems)
	}
}

// The entry a choice writes is a link the index resolves, so what points at a
// preset is answered by asking for its backlinks. A deck that named its preset
// by identifier is answered the same way after it is moved.
func TestWhatPointsAtAPresetIsAnsweredAfterAChoice(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Sanskrit.md":     "---\nid: 01M02ACGM0FYMSXNDP29C90JNR\ntype: preset\ngoal: minutes_a_day\n---\n\n# Sanskrit\n",
		"presets/Slow.md": choosing["presets/Slow.md"],
		"decks/Terms.md":  choosing["decks/Terms.md"],
		"decks/Roots.md": "---\ntype: deck\nlinks:\n  - to: \"note://01M02ACGM0FYMSXNDP29C90JNR\"\n" +
			"    role: ref\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	})

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Terms.md", "Sanskrit.md", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}
	if got := pointedAt(t, s, "Sanskrit.md"); !slices.Equal(got, []string{"decks/Roots.md", "decks/Terms.md"}) {
		t.Errorf("what points at the preset is %v", got)
	}

	// The deck that named it by identifier is moved, and the entry is written
	// in the form a link is written in.
	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "presets/Slow.md", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}
	if got := pointedAt(t, s, "Sanskrit.md"); !slices.Equal(got, []string{"decks/Terms.md"}) {
		t.Errorf("what points at the preset is %v", got)
	}
	if got := pointedAt(t, s, "presets/Slow.md"); !slices.Equal(got, []string{"decks/Roots.md"}) {
		t.Errorf("what points at the other preset is %v", got)
	}
}

// pointing is the decks whose preset link reaches the note at path, in the
// order they are filed under.
func pointedAt(t *testing.T, s vaulted, path string) []string {
	t.Helper()
	found, err := s.presets.Links.Backlinks(t.Context(), s.vault.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, one := range found {
		if one.Type == flashcards.LinkType {
			out = append(out, one.From)
		}
	}
	slices.Sort(out)
	return out
}
