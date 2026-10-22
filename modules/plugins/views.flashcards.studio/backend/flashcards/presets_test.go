package flashcards_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/note"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review/projection"
)

// A vault where one deck names a preset, one names an ordinary note, and one
// is empty.
var pointing = map[string]string{
	"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
		"new_a_day: 8\nreviews_a_day: 45\nretention: 0.87\nload: {sat: 50}\n---\n\n# Sanskrit\n",
	"Grammar.md": "---\ntype: note\n---\n\n# Grammar\n",
	"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	"decks/Mantras.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Grammar\n    role: ref\n    type: preset\n---\n\n## Gayatri ^zpqrstvwxy\n",
	"decks/Terms.md": "---\ntype: deck\n---\n\n## Term ^3f4g5h6j7k\n",
}

// A deck is scheduled by the preset its link names.
func TestADeckIsScheduledByThePresetItNames(t *testing.T) {
	t.Parallel()
	s := openVault(t, pointing)

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "Sanskrit.md" {
		t.Errorf("read from %q", target.Path)
	}
	if len(target.Problems) != 0 {
		t.Errorf("problems = %v", target.Problems)
	}
	if target.Settings.MinutesADay != 20 || target.Settings.NewADay != 8 || target.Settings.ReviewsADay != 45 {
		t.Errorf("preset = %+v", target.Settings)
	}
}

// A deck naming no preset is scheduled by the defaults.
func TestADeckNamingNoPreset(t *testing.T) {
	t.Parallel()
	s := openVault(t, pointing)

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "" {
		t.Errorf("read from %q", target.Path)
	}
	if !reflect.DeepEqual(target.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", target.Settings)
	}
}

// An entry of the `links:` block written with no role is not read, so a preset
// named in one schedules nothing. The deck is told why it is on the defaults.
func TestADeckWhosePresetLinkHasNoRole(t *testing.T) {
	t.Parallel()
	notes := map[string]string{
		"Sanskrit.md": pointing["Sanskrit.md"],
		"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
			"  - to: Sanskrit\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	}
	s := openVault(t, notes)

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "" {
		t.Errorf("read from %q", target.Path)
	}
	if !reflect.DeepEqual(target.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", target.Settings)
	}
	if len(target.Problems) != 1 || !strings.Contains(target.Problems[0], "no role") {
		t.Fatalf("problems = %v", target.Problems)
	}
	if !strings.Contains(target.Problems[0], "Sanskrit") {
		t.Errorf("the problem does not name the link: %q", target.Problems[0])
	}
}

// A preset note that is gone leaves the deck on the defaults, and the address
// that reaches nothing is named against it.
func TestADeckWhosePresetNoteIsMissingExistsOnTheDefaults(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Roots.md": deckOf("Sanskrit", 20, 0),
	})
	if got := unseen(s.sessionAt(t, today, saturday)); got != 2 {
		t.Fatalf("under the preset the day was asked %d new cards, want 2", got)
	}

	remove(t, s, "Sanskrit.md")

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "" {
		t.Errorf("read from %q", target.Path)
	}
	if !reflect.DeepEqual(target.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", target.Settings)
	}
	if len(target.Problems) != 1 || !strings.Contains(target.Problems[0], "Sanskrit") {
		t.Errorf("problems = %v, want the one naming the note that is gone", target.Problems)
	}
	// The defaults are steered by their minutes, and twenty of them hold every
	// card the deck has left.
	if got := unseen(s.sessionAt(t, today, saturday)); got != 20 {
		t.Errorf("on the defaults the day was asked %d new cards, want 20", got)
	}
}

// preset it is scheduled by.
func TestADeckNamingTwoPresetsIsHeldToTheFirstsBudget(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Few.md":         preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":        preset("new_a_day: 9\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Roots.md": deckNaming([]string{"Few", "Many"}, 20, 0),
	})

	if got := unseen(s.sessionAt(t, today, saturday)); got != 2 {
		t.Errorf("the day was asked %d new cards, want the two the first preset keeps", got)
	}
}

// A levelling that failed is not a write that failed. The fingerprint of the
// file the write produced comes back with it, and the caller's next save lands
// on that file.
func TestAPresetSavedWithNoLevellingPassesBackItsFingerprint(t *testing.T) {
	t.Parallel()
	s := openVault(t, pointing)
	presets := flashcards.NewPresets(
		s.env.Ports.Readers, s.env.Ports.Writers,
		s.links, s.notes, busy, review.Day{}, time.Now,
	)

	target, err := presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	settings := target.Settings
	settings.MinutesADay = 35

	pos, err := presets.Save(t.Context(), s.vault, "Sanskrit.md", settings, target.Fingerprint)
	if !errors.Is(err, note.ErrUnlevelled) {
		t.Fatalf("a levelling that failed came back as %v", err)
	}
	if pos.IsZero() {
		t.Fatal("the write handed back no fingerprint")
	}

	settings.MinutesADay = 40
	if _, err := presets.Save(t.Context(), s.vault, "Sanskrit.md", settings, pos); err != nil &&
		!errors.Is(err, note.ErrUnlevelled) {
		t.Fatalf("the next save failed: %v", err)
	}
	if raw := read(t, s.vault, "Sanskrit.md"); !strings.Contains(raw, "minutes_a_day: 40") {
		t.Errorf("the preset on disk is now %q", raw)
	}
}

// A write that never reached the vault is not one of those: the note is left as
// it stands and nothing was levelled.
func TestAPresetLeftAloneIsNotAnUnlevelledWrite(t *testing.T) {
	t.Parallel()
	s := openVault(t, pointing)
	was := read(t, s.vault, "Sanskrit.md")

	target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	// A fingerprint no file answers to, which is the note having moved on since
	// the caller read it.
	staleFingerprint := target.Fingerprint
	staleFingerprint.Size += 100
	settings := target.Settings
	settings.MinutesADay = 35

	_, err = s.presets.Save(t.Context(), s.vault, "Sanskrit.md", settings, staleFingerprint)
	if errors.Is(err, note.ErrUnlevelled) {
		t.Errorf("a write that did not land came back as a levelling: %v", err)
	}
	if err == nil {
		t.Fatal("a stale fingerprint was written over")
	}
	if now := read(t, s.vault, "Sanskrit.md"); now != was {
		t.Errorf("the preset on disk is now %q", now)
	}
}

// A deck naming no preset and carrying no such entry has nothing said against
// it: standing on the defaults is not a problem.
func TestADeckNamingNoPresetHasNothingSaidAgainstIt(t *testing.T) {
	t.Parallel()
	s := openVault(t, pointing)

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Problems) != 0 {
		t.Errorf("problems = %v", target.Problems)
	}
}

// A link reaching a note that is not a preset leaves the deck on the defaults
// and says so against it.
func TestADeckNamingANoteThatIsNotAPreset(t *testing.T) {
	t.Parallel()
	s := openVault(t, pointing)

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Mantras.md")
	if err != nil {
		t.Fatal(err)
	}
	// The note schedules nothing, so the deck stands with the decks naming none.
	if target.Path != "" {
		t.Errorf("read from %q", target.Path)
	}
	if !reflect.DeepEqual(target.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", target.Settings)
	}
	if len(target.Problems) != 1 || !strings.Contains(target.Problems[0], "not a preset") {
		t.Errorf("problems = %v", target.Problems)
	}
}

// A deck naming two presets is scheduled by the first and carries a problem.
func TestADeckNamingTwoPresets(t *testing.T) {
	t.Parallel()
	notes := map[string]string{
		"Sanskrit.md": "---\ntype: preset\nnew_a_day: 8\n---\n\n# Sanskrit\n",
		"Mantras.md":  "---\ntype: preset\nnew_a_day: 3\n---\n\n# Mantras\n",
		"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
			"  - to: Sanskrit\n    role: ref\n    type: preset\n" +
			"  - to: Mantras\n    role: ref\n    type: preset\n---\n\n## Root ^k7m2xq9fzp\n",
	}
	s := openVault(t, notes)

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Settings.NewADay != 8 {
		t.Errorf("new a day = %d", target.Settings.NewADay)
	}
	if len(target.Problems) != 1 || !strings.Contains(target.Problems[0], "more than one preset") {
		t.Errorf("problems = %v", target.Problems)
	}
}

// A vault whose preset carries keys the application does not own, an identity,
// comments on a key it owns, and an owned key standing last in the block.
var settled = map[string]string{
	"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ncolour: green\ntype: preset\n" +
		"# how long a day runs\nminutes_a_day: 20 # twenty is plenty\n" +
		"tags:\n  - study\ngoal: minutes_a_day\n---\n\n" +
		"# Sanskrit\n\nGrammar and vocabulary.\n",
	"Grammar.md": "---\ntype: note\n---\n\n# Grammar\n",
}

// minutes is a preset steered by how long a day runs.
func minutes() review.Preset {
	p := review.Defaults()
	p.MinutesADay, p.NewADay, p.ReviewsADay, p.Retention = 35, 8, 45, 0.87
	return p
}

// frontmatter is the whole of what a note's block says after a write, read by
// the parser every other note is read by. A key that went missing, a key that
// arrived twice and a block a person can no longer open all show here and in
// none of the substrings a write puts in.
func frontmatter(t *testing.T, s vaulted, path string) map[string]any {
	t.Helper()
	raw := []byte(read(t, s.vault, path))
	if _, err := markdown.Open(raw); err != nil {
		t.Fatalf("the note cannot be opened after the write: %v\n%s", err, raw)
	}
	n := markdown.Parse(domain.Fingerprint{Path: path}, raw)
	if n.FrontmatterErr != "" {
		t.Fatalf("the frontmatter cannot be read after the write: %s\n%s", n.FrontmatterErr, raw)
	}
	return n.Frontmatter
}

// A write puts the settings in and leaves every other key, and the body, as
// they were.
func TestAWritePreservesWhatItDoesNotOwn(t *testing.T) {
	t.Parallel()
	s := openVault(t, settled)

	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", minutes(), domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"id":     "01J8F3K2M9QRSTVWXYZ012",
		"colour": "green",
		"tags":   []any{"study"},
		"type":   "preset",
		// Owned, and written where the setting differs from what the note was
		// read as. The goal it already said is not rewritten, and a setting
		// standing at the default the note never named stays unnamed.
		"goal": "minutes_a_day", "minutes_a_day": 35, "new_a_day": 8,
		"reviews_a_day": 45, "retention": 0.87,
	}
	if got := frontmatter(t, s, "Sanskrit.md"); !reflect.DeepEqual(got, want) {
		t.Errorf("frontmatter = %v,\n           want %v", got, want)
	}
	target := read(t, s.vault, "Sanskrit.md")
	if !strings.Contains(target, "# Sanskrit\n\nGrammar and vocabulary.\n") {
		t.Errorf("the body is gone from\n%s", target)
	}
	// The comments are the person's, on a key the application owns as much as
	// on any other.
	for _, expectedComment := range []string{
		"# how long a day runs\n", "minutes_a_day: 35 # twenty is plenty\n",
	} {
		if !strings.Contains(target, expectedComment) {
			t.Errorf("%q is gone from\n%s", expectedComment, target)
		}
	}
}

// A block written in from the margin is spliced where its own keys stand. A key
// written flush ends the mapping, and the person's keys below it — the identity
// among them — stop being read at all.
func TestAWriteIntoAnIndentedBlock(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Sanskrit.md": "---\n  id: 01J8F3K2M9QRSTVWXYZ012\n  type: preset\n" +
			"  goal: minutes_a_day\n  minutes_a_day: 20\n  colour: green\n---\n\n# Sanskrit\n",
	})

	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", minutes(), domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"id": "01J8F3K2M9QRSTVWXYZ012", "colour": "green", "type": "preset",
		"goal": "minutes_a_day", "minutes_a_day": 35, "new_a_day": 8,
		"reviews_a_day": 45, "retention": 0.87,
	}
	if got := frontmatter(t, s, "Sanskrit.md"); !reflect.DeepEqual(got, want) {
		t.Errorf("frontmatter = %v,\n           want %v", got, want)
	}

	target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Settings.MinutesADay != 35 {
		t.Errorf("the note reads back at %d minutes a day", target.Settings.MinutesADay)
	}
}

// A note written by an editor that marks its files and ends its lines the other
// way comes out of a write written the same way.
func TestAWriteKeepsTheLineEndingsAndTheMark(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Sanskrit.md": "\xef\xbb\xbf---\r\nid: 01J8F3K2M9QRSTVWXYZ012\r\ntype: preset\r\n" +
			"minutes_a_day: 20\r\n---\r\n\r\n# Sanskrit\r\n",
	})

	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", minutes(), domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	target := read(t, s.vault, "Sanskrit.md")
	if !strings.HasPrefix(target, "\xef\xbb\xbf") {
		t.Errorf("the mark is gone from\n%q", target)
	}
	if strings.Contains(strings.ReplaceAll(target, "\r\n", ""), "\n") {
		t.Errorf("a bare newline was written into a note ending its lines the other way:\n%q", target)
	}
	if got := frontmatter(t, s, "Sanskrit.md")["minutes_a_day"]; got != 35 {
		t.Errorf("minutes_a_day = %v", got)
	}
}

// A setting outside its bounds fails and the note is left alone.
func TestASettingOutsideItsBoundsWritesNothing(t *testing.T) {
	t.Parallel()
	s := openVault(t, settled)
	was := read(t, s.vault, "Sanskrit.md")

	p := minutes()
	p.Retention = 1.5
	_, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{})
	if !errors.Is(err, flashcards.ErrOutOfBounds) {
		t.Fatalf("saving a retention of 1.5 said %v", err)
	}
	if target := read(t, s.vault, "Sanskrit.md"); target != was {
		t.Errorf("the note was written:\n%s", target)
	}
}

// A note that is not a preset fails.
func TestANoteThatIsNotAPresetIsNotSaved(t *testing.T) {
	t.Parallel()
	s := openVault(t, settled)
	was := read(t, s.vault, "Grammar.md")

	_, err := s.presets.Save(t.Context(), s.vault, "Grammar.md", minutes(), domain.Fingerprint{})
	if !errors.Is(err, flashcards.ErrNotAPreset) {
		t.Fatalf("saving into an ordinary note said %v", err)
	}
	if target := read(t, s.vault, "Grammar.md"); target != was {
		t.Errorf("the note was written:\n%s", target)
	}
}

// The load each day of the week carries goes in and comes back out as it was.
func TestTheLoadReachesBackAsItWentIn(t *testing.T) {
	t.Parallel()
	s := openVault(t, settled)

	p := minutes()
	p.Load = map[time.Weekday]int{time.Saturday: 50, time.Sunday: 0}
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Problems) != 0 {
		t.Errorf("problems = %v", target.Problems)
	}
	if !reflect.DeepEqual(target.Settings.Load, p.Load) {
		t.Errorf("load = %v", target.Settings.Load)
	}
}

// A `load` entry the read could not make out stands where it was written. The
// file is never repaired, and a save that wrote only the days it understood
// would take the rest of the week out with it.
func TestASavePreservesTheLoadItCouldNotRead(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\ngoal: minutes_a_day\n" +
			"load:\n  sat: 50\n  fri: half\n  caturdasi: 20\n---\n\n# Sanskrit\n",
	})

	p := minutes()
	p.Load = map[time.Weekday]int{time.Sunday: 0}
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"fri": "half", "caturdasi": 20, "sun": 0,
	}
	if got := frontmatter(t, s, "Sanskrit.md")["load"]; !reflect.DeepEqual(got, want) {
		t.Errorf("load = %v, want %v\n%s", got, want, read(t, s.vault, "Sanskrit.md"))
	}
	if got := read(t, s.vault, "Sanskrit.md"); !strings.Contains(
		got, "load:\n  fri: half\n  caturdasi: 20\n  sun: 0\n") {
		t.Errorf("the entries moved:\n%s", got)
	}
	// The two the read could not make out are still what it cannot read, and
	// they are still said against the note.
	target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Problems) != 2 {
		t.Errorf("problems = %v", target.Problems)
	}
}

// A save into a note whose every `load` entry is one the read could not make
// out leaves the key at its current position.
func TestASavePreservesALoadItCouldReadNoneOf(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\ngoal: minutes_a_day\n" +
			"load:\n  fri: half\n---\n\n# Sanskrit\n",
	})

	p := minutes()
	p.Load = map[time.Weekday]int{time.Sunday: 0}
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	want := map[string]any{"fri": "half", "sun": 0}
	if got := frontmatter(t, s, "Sanskrit.md")["load"]; !reflect.DeepEqual(got, want) {
		t.Errorf("load = %v, want %v\n%s", got, want, read(t, s.vault, "Sanskrit.md"))
	}
}

// A `load` written as anything but a week of days is the person's whole, and a
// save leaves it currently.
func TestASavePreservesALoadThatIsNotAWeek(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\ngoal: minutes_a_day\n" +
			"load: every other day\n---\n\n# Sanskrit\n",
	})

	p := minutes()
	p.Load = map[time.Weekday]int{time.Saturday: 50}
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if got := frontmatter(t, s, "Sanskrit.md")["load"]; got != "every other day" {
		t.Errorf("load = %v\n%s", got, read(t, s.vault, "Sanskrit.md"))
	}
}

// A goal of a day writes the day, and it is read back as the day it was.
func TestAGoalOfADateWritesTheDay(t *testing.T) {
	t.Parallel()
	s := openVault(t, settled)

	p := minutes()
	p.Goal, p.By = review.GoalDate, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if target := read(t, s.vault, "Sanskrit.md"); !strings.Contains(target, "by_date: 2026-12-01") {
		t.Errorf("the day was not written to\n%s", target)
	}
	targetVal, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if targetVal.Settings.Goal != review.GoalDate || !targetVal.Settings.By.Equal(p.By) {
		t.Errorf("preset = %+v", targetVal.Settings)
	}
}

// The rule a preset does not name keeps its value through a write, so a person
// who set an interval finds it where they left it after a spell under the other
// rule.
func TestTheRuleNotNamedKeepsItsValue(t *testing.T) {
	t.Parallel()
	s := openVault(t, settled)

	p := minutes()
	p.Rule, p.Interval, p.Retention = review.RuleRetention, 45, 0.87
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	target := read(t, s.vault, "Sanskrit.md")
	for _, content := range []string{"learned: retention", "interval: 45", "retention: 0.87"} {
		if !strings.Contains(target, content) {
			t.Errorf("%q was not written to\n%s", content, target)
		}
	}

	back, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Problems) != 0 {
		t.Errorf("problems = %v", back.Problems)
	}
	if back.Settings.Rule != review.RuleRetention || back.Settings.Interval != 45 {
		t.Errorf("preset = %+v", back.Settings)
	}
}

// A preset written before the rule existed is read at the default rule, and its
// cards are counted against the default threshold.
//
// The file names no rule and no interval, which is every preset a person wrote
// before there was one to name. A card face put off by less than the default
// interval is not learned, and the counts a window draws say so.
func TestAPresetSavedBeforeTheRuleCountsByTheDefault(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Old.md": "---\ntype: preset\ngoal: retention\nminutes_a_day: 10\n" +
			"new_a_day: 12\nreviews_a_day: 5\nretention: 0.8\nbacklog: 68\n" +
			"even_load: true\n---\n\n# Steady\n",
	})

	read, err := s.presets.Read(t.Context(), s.vault, "Old.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Problems) != 0 {
		t.Fatalf("problems = %v", read.Problems)
	}
	p := read.Settings
	if p.Rule != review.RuleInterval || p.Interval != review.Defaults().Interval {
		t.Errorf("a file naming no rule was read as %q at %d days", p.Rule, p.Interval)
	}

	// Two card faces first seen yesterday, one put off by sixteen days and one
	// by twenty-one.
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.Local)
	last := now.AddDate(0, 0, -1)
	pos := map[review.CardFaceID]review.Schedule{
		{Card: "near", Face: "Recognise"}: {
			Last: last, Due: last.AddDate(0, 0, 16), Reps: 1, Stability: 16,
		},
		{Card: "far", Face: "Recognise"}: {
			Last: last, Due: last.AddDate(0, 0, 21), Reps: 1, Stability: 21,
		},
	}
	if p.IsLearned(pos[review.CardFaceID{Card: "near", Face: "Recognise"}], now) {
		t.Error("a card face sixteen days off is learned at an interval of twenty-one")
	}

	// And the counts the window draws off the projection say the same.
	run := projection.Simulation{
		By: review.NewFSRSAt(p.Retention), Day: today, Cost: review.DefaultCost, Days: 1,
	}
	ran, err := run.Run(t.Context(), now, p, pos, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ran.Learned != 1 {
		t.Errorf("%d of the two card faces stand learned, want the one sent away for 21 days",
			ran.Learned)
	}
	if ran.Through[0] >= 1 {
		t.Errorf("the day leaves the material %v learned, and one of the two is not", ran.Through[0])
	}
}

// A note over the bound a read refuses is a note a save refuses too, and
// nothing is written.
func TestASaveRefusesANoteOverTheBound(t *testing.T) {
	t.Parallel()
	body := "# Sanskrit\n\n" + strings.Repeat("Grammar and vocabulary. ", note.MaxBytes/20)
	s := openVault(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\n" +
			"goal: minutes_a_day\nminutes_a_day: 20\n---\n\n" + body,
	})
	was := read(t, s.vault, "Sanskrit.md")

	_, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", minutes(), domain.Fingerprint{})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("saving into a note of %d bytes said %v", len(was), err)
	}
	if target := read(t, s.vault, "Sanskrit.md"); target != was {
		t.Error("the note was written")
	}
}

// written. The read names it as a problem and the default takes its place while
// the file is read; the file itself is never repaired, and a save that put the
// default down would write a value the person never chose.
func TestASavePreservesTheSettingsItCouldNotRead(t *testing.T) {
	t.Parallel()
	for _, wrong := range []string{
		"even_load: no",
		"retention: ninety per cent",
		"retention: 90",
		"new_a_day: 12.5",
		"new_a_day: -5",
		"interval: 0",
		"interval: \"21\"",
		"backlog: 50%",
		"goal: Minutes a day",
		"learned: by days",
		"counts: every showing",
		"by_date: sometime",
		"reviews_a_day: many",
	} {
		t.Run(wrong, func(t *testing.T) {
			s := openVault(t, map[string]string{
				"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\n" +
					"minutes_a_day: 20\n" + wrong + "\n---\n\n# Sanskrit\n",
			})

			// The person moved the minutes control and nothing else.
			target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
			if err != nil {
				t.Fatal(err)
			}
			settings := target.Settings
			settings.MinutesADay = 35
			if _, err := s.presets.Save(
				t.Context(), s.vault, "Sanskrit.md", settings, domain.Fingerprint{}); err != nil {
				t.Fatal(err)
			}

			got := read(t, s.vault, "Sanskrit.md")
			if !strings.Contains(got, wrong+"\n") {
				t.Errorf("%q was written over in\n%s", wrong, got)
			}
			if !strings.Contains(got, "minutes_a_day: 35\n") {
				t.Errorf("the minutes the person moved were not written to\n%s", got)
			}
		})
	}
}

// A setting the read could not make out and the person has since moved is
// written: that is them settling it.
func TestASaveWritesTheSettingThePersonMoved(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\n" +
			"even_load: no\n---\n\n# Sanskrit\n",
	})

	target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	settings := target.Settings
	settings.IsEvenLoad = false
	if _, err := s.presets.Save(
		t.Context(), s.vault, "Sanskrit.md", settings, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if got := read(t, s.vault, "Sanskrit.md"); !strings.Contains(got, "even_load: false\n") {
		t.Errorf("the setting the person moved was not written to\n%s", got)
	}
}

// A save that changes nothing leaves the file currently, comments and all.
func TestASaveOfWhatTheNoteAlreadyReturnsWritesNothing(t *testing.T) {
	t.Parallel()
	s := openVault(t, settled)
	was := read(t, s.vault, "Sanskrit.md")

	target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.presets.Save(
		t.Context(), s.vault, "Sanskrit.md", target.Settings, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if got := read(t, s.vault, "Sanskrit.md"); got != was {
		t.Errorf("the note was rewritten\n was %q\n got %q", was, got)
	}
}

// A day the block already names carries its new share in the place it was
// written in, and is written once.
func TestADayTheBlockNamesIsRewrittenWhereItExists(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\ngoal: minutes_a_day\n" +
			"load:\n  wed: 50\n  mon: 80\n---\n\n# Sanskrit\n",
	})

	p := minutes()
	p.Load = map[time.Weekday]int{time.Wednesday: 30, time.Monday: 80}
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if got := read(t, s.vault, "Sanskrit.md"); !strings.Contains(
		got, "load:\n  wed: 30\n  mon: 80\n") {
		t.Errorf("the load was rewritten as:\n%s", got)
	}

	target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(target.Settings.Load, p.Load) {
		t.Errorf("load = %v, want %v", target.Settings.Load, p.Load)
	}
}

// A day named twice, under names that differ only in how they are written, is
// one day to the read. The share goes into the entry the read takes, and the
// save does not report a value the next read will not find.
func TestADayNamedTwiceIsSavedWhereTheReadConsumesIt(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\ngoal: minutes_a_day\n" +
			"load:\n  Mon: 100\n  mon: 50\n---\n\n# Sanskrit\n",
	})

	p := minutes()
	p.Load = map[time.Weekday]int{time.Monday: 80}
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	target, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if got := target.Settings.Load[time.Monday]; got != 80 {
		t.Errorf("monday came back at %d, and the save put it at 80\n%s",
			got, read(t, s.vault, "Sanskrit.md"))
	}
}

// note is left as the person wrote it.
//
// A client that named none of goal, learned or counts sends a value that is not
// one of them, and a day the goal names has to be there for the goal to read.
func TestSettingsAPresetMayNotHoldAreRefused(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		what   string
		of     func(review.Preset) review.Preset
		reason string
	}{{
		what:   "a goal the application does not know",
		of:     func(p review.Preset) review.Preset { p.Goal = ""; return p },
		reason: "goal",
	}, {
		what:   "a rule the application does not know",
		of:     func(p review.Preset) review.Preset { p.Rule = ""; return p },
		reason: "learned",
	}, {
		what:   "a goal of a date naming no day",
		of:     func(p review.Preset) review.Preset { p.Goal, p.By = review.GoalDate, time.Time{}; return p },
		reason: "day",
	}, {
		what: "a share of a day outside its bounds",
		of: func(p review.Preset) review.Preset {
			p.Load = map[time.Weekday]int{time.Saturday: 140}
			return p
		},
		reason: "the load of sat",
	}} {
		t.Run(one.what, func(t *testing.T) {
			s := openVault(t, settled)
			was := read(t, s.vault, "Sanskrit.md")

			_, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", one.of(minutes()), domain.Fingerprint{})
			if !errors.Is(err, flashcards.ErrOutOfBounds) {
				t.Fatalf("saving %s said %v", one.what, err)
			}
			if !strings.Contains(err.Error(), one.reason) {
				t.Errorf("the rejection of %s says %q, and names nothing of %q", one.what, err, one.reason)
			}
			if target := read(t, s.vault, "Sanskrit.md"); target != was {
				t.Errorf("the note was written:\n%s", target)
			}
		})
	}
}

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
	s := openVault(t, choosing)

	target, err := s.presets.List(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 2 {
		t.Fatalf("listed %+v", target)
	}
	if target[0].Path != "Sanskrit.md" || target[0].Title != "Sanskrit" {
		t.Errorf("the first is %+v", target[0])
	}
	if target[1].Path != "presets/Slow.md" || target[1].Title != "Slow going" {
		t.Errorf("the second is %+v", target[1])
	}
}

// A vault holding no preset lists none. Every deck in it is scheduled by the
// defaults, which are no note.
func TestAVaultOfNoPresetsListsNone(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	target, err := s.presets.List(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 0 {
		t.Errorf("listed %+v", target)
	}
}

// A build that cannot ask which notes are presets says so. A vault holding none
// answers with an empty list, and only that is a fact about the vault.
func TestABuildThatCannotReachThePresetsRefusesToListThem(t *testing.T) {
	t.Parallel()
	s := openVault(t, choosing)

	blind := flashcards.NewPresets(
		s.env.Ports.Readers, s.env.Ports.Writers,
		s.links, nil, s.scan, review.Day{}, time.Now,
	)
	if _, err := blind.List(t.Context(), s.vault); !errors.Is(err, flashcards.ErrNoPresets) {
		t.Errorf("listing the presets of a vault it cannot reach answered %v", err)
	}
}

// A deck that named no preset names one, and is scheduled by it afterwards.
func TestADeckIsInsertOnAPreset(t *testing.T) {
	t.Parallel()
	s := openVault(t, choosing)

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

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "Sanskrit.md" || target.Settings.MinutesADay != 20 {
		t.Errorf("the deck is scheduled by %q at %+v", target.Path, target.Settings)
	}
}

// A deck already on a preset is put on another, and names one preset
// afterwards.
func TestADeckIsMovedFromOnePresetToAnother(t *testing.T) {
	t.Parallel()
	s := openVault(t, choosing)

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

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "presets/Slow.md" {
		t.Errorf("the deck is scheduled by %q", target.Path)
	}
	if len(target.Problems) != 0 {
		t.Errorf("problems = %v", target.Problems)
	}
}

// A deck is taken off its preset, and is scheduled by the defaults again. The
// `links:` block goes with the entry it held.
func TestADeckIsTakenOffItsPreset(t *testing.T) {
	t.Parallel()
	s := openVault(t, choosing)

	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	got := read(t, s.vault, "decks/Roots.md")
	if strings.Contains(got, "links:") || strings.Contains(got, "preset") {
		t.Errorf("the deck still names a preset:\n%s", got)
	}

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "" {
		t.Errorf("the deck is scheduled by %q", target.Path)
	}
}

// Everything the deck's frontmatter holds beside the one entry comes out of the
// write as the bytes it went in as.
func TestPointingADeckPreservesTheRestOfTheFrontmatter(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
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
	for _, fragment := range []string{
		"id: 01J8F3K2M9QRSTVWXYZ012\n", "tags: [grammar, roots]\n",
		"# the ones I keep coming back to\n",
		"  - to: Grammar\n    role: parent\n    note: where these come from\n",
	} {
		if !strings.Contains(got, fragment) {
			t.Errorf("%q was written over in\n%s", fragment, got)
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
	s := openVault(t, map[string]string{
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
// one and the file is left currently.
func TestADeckIsNotPointedAtANoteThatIsNotAPreset(t *testing.T) {
	t.Parallel()
	s := openVault(t, choosing)
	was := read(t, s.vault, "decks/Terms.md")

	_, err := s.presets.Point(t.Context(), s.vault, "decks/Terms.md", "Grammar.md", domain.Fingerprint{})
	if !errors.Is(err, flashcards.ErrNotAPreset) {
		t.Fatalf("err = %v", err)
	}
	if got := read(t, s.vault, "decks/Terms.md"); got != was {
		t.Errorf("the deck was written\n was %q\n got %q", was, got)
	}
}

func TestADeckIsNotPointedAtANoteThatIsNotThere(t *testing.T) {
	t.Parallel()
	s := openVault(t, choosing)
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
	s := openVault(t, choosing)

	pos, err := s.presets.Point(
		t.Context(), s.vault, "decks/Terms.md", "Sanskrit.md", domain.Fingerprint{})
	if err != nil {
		t.Fatal(err)
	}
	was := read(t, s.vault, "decks/Terms.md")

	staleFingerprint := pos
	staleFingerprint.Size += 3
	_, err = s.presets.Point(t.Context(), s.vault, "decks/Terms.md", "presets/Slow.md", staleFingerprint)
	if !errors.Is(err, ports.ErrStale) {
		t.Fatalf("err = %v", err)
	}
	if got := read(t, s.vault, "decks/Terms.md"); got != was {
		t.Errorf("the deck was written\n was %q\n got %q", was, got)
	}
}

func TestTheFingerprintAPointAnswersWithIsPresentedAgain(t *testing.T) {
	t.Parallel()
	s := openVault(t, choosing)

	pos, err := s.presets.Point(
		t.Context(), s.vault, "decks/Terms.md", "Sanskrit.md", domain.Fingerprint{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Terms.md", "presets/Slow.md", pos); err != nil {
		t.Fatal(err)
	}

	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "presets/Slow.md" {
		t.Errorf("the deck is scheduled by %q", target.Path)
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
			s := openVault(t, map[string]string{
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
	s := openVault(t, choosing)

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
	if n.Type != format.TypeDeck {
		t.Errorf("the note reads as a %s", n.Type)
	}
}

// A block holding other entries keeps them when the preset entry goes.
func TestTakingADeckOffItsPresetKeepsItsOtherLinks(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
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
	s := openVault(t, map[string]string{
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
	target, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "presets/Slow.md" || len(target.Problems) != 0 {
		t.Errorf("the deck is scheduled by %q with %v", target.Path, target.Problems)
	}
}

// The entry a choice writes is a link the index resolves, so what points at a
// preset is answered by asking for its backlinks. A deck that named its preset
// by identifier is answered the same way after it is moved.
func TestWhatPointsAtAPresetIsAnsweredAfterAChoice(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
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
	if got := getPointingDecks(t, s, "Sanskrit.md"); !slices.Equal(got, []string{"decks/Roots.md", "decks/Terms.md"}) {
		t.Errorf("what points at the preset is %v", got)
	}

	// The deck that named it by identifier is moved, and the entry is written
	// in the form a link is written in.
	if _, err := s.presets.Point(
		t.Context(), s.vault, "decks/Roots.md", "presets/Slow.md", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}
	if got := getPointingDecks(t, s, "Sanskrit.md"); !slices.Equal(got, []string{"decks/Terms.md"}) {
		t.Errorf("what points at the preset is %v", got)
	}
	if got := getPointingDecks(t, s, "presets/Slow.md"); !slices.Equal(got, []string{"decks/Roots.md"}) {
		t.Errorf("what points at the other preset is %v", got)
	}
}

// getPointingDecks is the decks whose preset link reaches the note at path, in
// the order they are filed under.
func getPointingDecks(t *testing.T, s vaulted, path string) []string {
	t.Helper()
	found, err := s.links.GetBacklinks(t.Context(), s.vault.ID, path)
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
