package flashcards_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// A vault where one deck names a preset, one names an ordinary note, and one
// names nothing.
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
	if held.Settings.MinutesADay != 20 || held.Settings.NewADay != 8 || held.Settings.ReviewsADay != 45 {
		t.Errorf("preset = %+v", held.Settings)
	}
}

// A deck naming no preset is scheduled by the defaults.
func TestADeckNamingNoPreset(t *testing.T) {
	t.Parallel()
	s := opened(t, pointing)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "" {
		t.Errorf("read from %q", held.Path)
	}
	if !reflect.DeepEqual(held.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", held.Settings)
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
	s := opened(t, notes)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "" {
		t.Errorf("read from %q", held.Path)
	}
	if !reflect.DeepEqual(held.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", held.Settings)
	}
	if len(held.Problems) != 1 || !strings.Contains(held.Problems[0], "no role") {
		t.Fatalf("problems = %v", held.Problems)
	}
	if !strings.Contains(held.Problems[0], "Sanskrit") {
		t.Errorf("the problem does not name the link: %q", held.Problems[0])
	}
}

// A preset note that is gone leaves the deck on the defaults, and the address
// that reaches nothing is named against it.
func TestADeckWhosePresetNoteIsGoneStandsOnTheDefaults(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Roots.md": deckOf("Sanskrit", 20, 0),
	})
	if got := unseen(s.sittingAt(t, today, saturday)); got != 2 {
		t.Fatalf("under the preset the day was asked %d new cards, want 2", got)
	}

	remove(t, s, "Sanskrit.md")

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Path != "" {
		t.Errorf("read from %q", held.Path)
	}
	if !reflect.DeepEqual(held.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", held.Settings)
	}
	if len(held.Problems) != 1 || !strings.Contains(held.Problems[0], "Sanskrit") {
		t.Errorf("problems = %v, want the one naming the note that is gone", held.Problems)
	}
	// The defaults are steered by their minutes, and twenty of them hold every
	// card the deck has left.
	if got := unseen(s.sittingAt(t, today, saturday)); got != 20 {
		t.Errorf("on the defaults the day was asked %d new cards, want 20", got)
	}
}

// A deck naming two presets is held to the budget of the first, which is the
// preset it is scheduled by.
func TestADeckNamingTwoPresetsIsHeldToTheFirstsBudget(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Few.md":         preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":        preset("new_a_day: 9\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Roots.md": deckNaming([]string{"Few", "Many"}, 20, 0),
	})

	if got := unseen(s.sittingAt(t, today, saturday)); got != 2 {
		t.Errorf("the day was asked %d new cards, want the two the first preset keeps", got)
	}
}

// A levelling that failed is not a write that failed. The fingerprint of the
// file the write produced comes back with it, and the caller's next save lands
// on that file.
func TestAPresetWrittenWithNoLevellingHandsBackItsFingerprint(t *testing.T) {
	t.Parallel()
	s := opened(t, pointing)
	presets := s.presets
	presets.Index = busy

	held, err := presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	settings := held.Settings
	settings.MinutesADay = 35

	at, err := presets.Save(t.Context(), s.vault, "Sanskrit.md", settings, held.Fingerprint)
	if !errors.Is(err, note.ErrUnlevelled) {
		t.Fatalf("a levelling that failed came back as %v", err)
	}
	if at == (domain.Fingerprint{}) {
		t.Fatal("the write handed back no fingerprint")
	}

	settings.MinutesADay = 40
	if _, err := presets.Save(t.Context(), s.vault, "Sanskrit.md", settings, at); err != nil &&
		!errors.Is(err, note.ErrUnlevelled) {
		t.Fatalf("the next save was refused: %v", err)
	}
	if raw := read(t, s.vault, "Sanskrit.md"); !strings.Contains(raw, "minutes_a_day: 40") {
		t.Errorf("the preset on disk is now %q", raw)
	}
}

// A write that never reached the vault is not one of those: the note is left as
// it stands and nothing was levelled.
func TestAPresetLeftAloneIsNotAnUnlevelledWrite(t *testing.T) {
	t.Parallel()
	s := opened(t, pointing)
	was := read(t, s.vault, "Sanskrit.md")

	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	// A fingerprint no file answers to, which is the note having moved on since
	// the caller read it.
	stale := held.Fingerprint
	stale.Size += 100
	settings := held.Settings
	settings.MinutesADay = 35

	_, err = s.presets.Save(t.Context(), s.vault, "Sanskrit.md", settings, stale)
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
	s := opened(t, pointing)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Terms.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Problems) != 0 {
		t.Errorf("problems = %v", held.Problems)
	}
}

// A link reaching a note that is not a preset leaves the deck on the defaults
// and says so against it.
func TestADeckNamingANoteThatIsNotAPreset(t *testing.T) {
	t.Parallel()
	s := opened(t, pointing)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Mantras.md")
	if err != nil {
		t.Fatal(err)
	}
	// The note schedules nothing, so the deck stands with the decks naming none.
	if held.Path != "" {
		t.Errorf("read from %q", held.Path)
	}
	if !reflect.DeepEqual(held.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", held.Settings)
	}
	if len(held.Problems) != 1 || !strings.Contains(held.Problems[0], "not a preset") {
		t.Errorf("problems = %v", held.Problems)
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
	s := opened(t, notes)

	held, err := s.presets.Of(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Settings.NewADay != 8 {
		t.Errorf("new a day = %d", held.Settings.NewADay)
	}
	if len(held.Problems) != 1 || !strings.Contains(held.Problems[0], "more than one preset") {
		t.Errorf("problems = %v", held.Problems)
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
func TestAWriteLeavesWhatItDoesNotOwn(t *testing.T) {
	t.Parallel()
	s := opened(t, settled)

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
	held := read(t, s.vault, "Sanskrit.md")
	if !strings.Contains(held, "# Sanskrit\n\nGrammar and vocabulary.\n") {
		t.Errorf("the body is gone from\n%s", held)
	}
	// The comments are the person's, on a key the application owns as much as
	// on any other.
	for _, kept := range []string{
		"# how long a day runs\n", "minutes_a_day: 35 # twenty is plenty\n",
	} {
		if !strings.Contains(held, kept) {
			t.Errorf("%q is gone from\n%s", kept, held)
		}
	}
}

// A block written in from the margin is spliced where its own keys stand. A key
// written flush ends the mapping, and the person's keys below it — the identity
// among them — stop being read at all.
func TestAWriteIntoAnIndentedBlock(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
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

	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Settings.MinutesADay != 35 {
		t.Errorf("the note reads back at %d minutes a day", held.Settings.MinutesADay)
	}
}

// A note written by an editor that marks its files and ends its lines the other
// way comes out of a write written the same way.
func TestAWriteKeepsTheLineEndingsAndTheMark(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Sanskrit.md": "\xef\xbb\xbf---\r\nid: 01J8F3K2M9QRSTVWXYZ012\r\ntype: preset\r\n" +
			"minutes_a_day: 20\r\n---\r\n\r\n# Sanskrit\r\n",
	})

	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", minutes(), domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	held := read(t, s.vault, "Sanskrit.md")
	if !strings.HasPrefix(held, "\xef\xbb\xbf") {
		t.Errorf("the mark is gone from\n%q", held)
	}
	if strings.Contains(strings.ReplaceAll(held, "\r\n", ""), "\n") {
		t.Errorf("a bare newline was written into a note ending its lines the other way:\n%q", held)
	}
	if got := frontmatter(t, s, "Sanskrit.md")["minutes_a_day"]; got != 35 {
		t.Errorf("minutes_a_day = %v", got)
	}
}

// A setting outside its bounds is refused and the note is left alone.
func TestASettingOutsideItsBoundsWritesNothing(t *testing.T) {
	t.Parallel()
	s := opened(t, settled)
	was := read(t, s.vault, "Sanskrit.md")

	p := minutes()
	p.Retention = 1.5
	_, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{})
	if !errors.Is(err, flashcards.ErrOutOfBounds) {
		t.Fatalf("saving a retention of 1.5 said %v", err)
	}
	if held := read(t, s.vault, "Sanskrit.md"); held != was {
		t.Errorf("the note was written:\n%s", held)
	}
}

// A note that is not a preset is refused.
func TestANoteThatIsNotAPresetIsNotWritten(t *testing.T) {
	t.Parallel()
	s := opened(t, settled)
	was := read(t, s.vault, "Grammar.md")

	_, err := s.presets.Save(t.Context(), s.vault, "Grammar.md", minutes(), domain.Fingerprint{})
	if !errors.Is(err, flashcards.ErrNotAPreset) {
		t.Fatalf("saving into an ordinary note said %v", err)
	}
	if held := read(t, s.vault, "Grammar.md"); held != was {
		t.Errorf("the note was written:\n%s", held)
	}
}

// The load each day of the week carries goes in and comes back out as it was.
func TestTheLoadComesBackAsItWentIn(t *testing.T) {
	t.Parallel()
	s := opened(t, settled)

	p := minutes()
	p.Load = map[time.Weekday]int{time.Saturday: 50, time.Sunday: 0}
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Problems) != 0 {
		t.Errorf("problems = %v", held.Problems)
	}
	if !reflect.DeepEqual(held.Settings.Load, p.Load) {
		t.Errorf("load = %v", held.Settings.Load)
	}
}

// A `load` entry the read could not make out stands where it was written. The
// file is never repaired, and a save that wrote only the days it understood
// would take the rest of the week out with it.
func TestASaveLeavesTheLoadItCouldNotRead(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
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
	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Problems) != 2 {
		t.Errorf("problems = %v", held.Problems)
	}
}

// A save into a note whose every `load` entry is one the read could not make
// out leaves the key where it stands.
func TestASaveLeavesALoadItCouldReadNoneOf(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
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
// save leaves it as it stands.
func TestASaveLeavesALoadThatIsNotAWeek(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
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
	s := opened(t, settled)

	p := minutes()
	p.Goal, p.By = review.GoalDate, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if held := read(t, s.vault, "Sanskrit.md"); !strings.Contains(held, "by_date: 2026-12-01") {
		t.Errorf("the day was not written to\n%s", held)
	}
	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if held.Settings.Goal != review.GoalDate || !held.Settings.By.Equal(p.By) {
		t.Errorf("preset = %+v", held.Settings)
	}
}

// The rule a preset does not name keeps its value through a write, so a person
// who set an interval finds it where they left it after a spell under the other
// rule.
func TestTheRuleNotNamedKeepsItsValue(t *testing.T) {
	t.Parallel()
	s := opened(t, settled)

	p := minutes()
	p.Rule, p.Interval, p.Retention = review.RuleRetention, 45, 0.87
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	held := read(t, s.vault, "Sanskrit.md")
	for _, written := range []string{"learned: retention", "interval: 45", "retention: 0.87"} {
		if !strings.Contains(held, written) {
			t.Errorf("%q was not written to\n%s", written, held)
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
func TestAPresetWrittenBeforeTheRuleCountsByTheDefault(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
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
	at := map[review.CardFaceID]review.Schedule{
		{Card: "near", Face: "Recognise"}: {
			Last: last, Due: last.AddDate(0, 0, 16), Reps: 1, Stability: 16,
		},
		{Card: "far", Face: "Recognise"}: {
			Last: last, Due: last.AddDate(0, 0, 21), Reps: 1, Stability: 21,
		},
	}
	if p.Learned(at[review.CardFaceID{Card: "near", Face: "Recognise"}], now) {
		t.Error("a card face sixteen days off is learned at an interval of twenty-one")
	}

	// And the counts the window draws off the projection say the same.
	run := review.Simulation{
		By: review.NewFSRSAt(p.Retention), Day: today, Cost: review.DefaultCost, Days: 1,
	}
	ran, err := run.Run(t.Context(), now, p, at, 0)
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
	s := opened(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\n" +
			"goal: minutes_a_day\nminutes_a_day: 20\n---\n\n" + body,
	})
	was := read(t, s.vault, "Sanskrit.md")

	_, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", minutes(), domain.Fingerprint{})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("saving into a note of %d bytes said %v", len(was), err)
	}
	if held := read(t, s.vault, "Sanskrit.md"); held != was {
		t.Error("the note was written")
	}
}

// A setting the read could not make out stands in the file exactly as it was
// written. The read names it as a problem and the default takes its place while
// the file is read; the file itself is never repaired, and a save that put the
// default down would write a value the person never chose.
func TestASaveLeavesTheSettingsItCouldNotRead(t *testing.T) {
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
			s := opened(t, map[string]string{
				"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\n" +
					"minutes_a_day: 20\n" + wrong + "\n---\n\n# Sanskrit\n",
			})

			// The person moved the minutes control and nothing else.
			held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
			if err != nil {
				t.Fatal(err)
			}
			settings := held.Settings
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
	s := opened(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\n" +
			"even_load: no\n---\n\n# Sanskrit\n",
	})

	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	settings := held.Settings
	settings.EvenLoad = false
	if _, err := s.presets.Save(
		t.Context(), s.vault, "Sanskrit.md", settings, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if got := read(t, s.vault, "Sanskrit.md"); !strings.Contains(got, "even_load: false\n") {
		t.Errorf("the setting the person moved was not written to\n%s", got)
	}
}

// A save that changes nothing leaves the file as it stands, comments and all.
func TestASaveOfWhatTheNoteAlreadySaysWritesNothing(t *testing.T) {
	t.Parallel()
	s := opened(t, settled)
	was := read(t, s.vault, "Sanskrit.md")

	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.presets.Save(
		t.Context(), s.vault, "Sanskrit.md", held.Settings, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	if got := read(t, s.vault, "Sanskrit.md"); got != was {
		t.Errorf("the note was rewritten\n was %q\n got %q", was, got)
	}
}

// A day the block already names carries its new share in the place it was
// written in, and is written once.
func TestADayTheBlockNamesIsRewrittenWhereItStands(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
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

	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(held.Settings.Load, p.Load) {
		t.Errorf("load = %v, want %v", held.Settings.Load, p.Load)
	}
}

// A day named twice, under names that differ only in how they are written, is
// one day to the read. The share goes into the entry the read takes, and the
// save does not report a value the next read will not find.
func TestADayNamedTwiceIsSavedWhereTheReadTakesIt(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Sanskrit.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: preset\ngoal: minutes_a_day\n" +
			"load:\n  Mon: 100\n  mon: 50\n---\n\n# Sanskrit\n",
	})

	p := minutes()
	p.Load = map[time.Weekday]int{time.Monday: 80}
	if _, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", p, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	held, err := s.presets.Read(t.Context(), s.vault, "Sanskrit.md")
	if err != nil {
		t.Fatal(err)
	}
	if got := held.Settings.Load[time.Monday]; got != 80 {
		t.Errorf("monday came back at %d, and the save put it at 80\n%s",
			got, read(t, s.vault, "Sanskrit.md"))
	}
}
