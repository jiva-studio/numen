package service_test

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service"
)

// A preset defines how the decks pointing at it are scheduled. These tests
// cover what the window reads and writes about a preset.

// scheduling is a vault with the notes given in it, and a client asking about
// its presets the way the window does.
type scheduling struct {
	client service.PresetsAPI
	root   string
}

func newScheduling(t *testing.T, notes map[string]string) *scheduling {
	t.Helper()

	env := vaulttest.New(t, notes)
	backend := service.New(service.Config{Vault: env.Host, Store: env.Store})

	return &scheduling{
		client: backend.Presets(),
		root:   env.Vault.Path,
	}
}

// sanskrit is a preset note, with one deck pointing at it and one deck
// pointing at no preset.
const sanskrit = "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
	"new_a_day: 8\nreviews_a_day: 45\nretention: 0.87\nload:\n  sat: 50\n  sun: 0\n" +
	"even_load: true\n---\n\n# Sanskrit\n\nGrammar and vocabulary.\n"

const roots = "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n---\n\n" +
	"## Root ^k7m2xq9fzp\n"

const terms = "---\ntype: deck\n---\n\n## Term ^3f4g5h6j7k\n"

// pointed is that vault, as every test here opens it.
var pointed = map[string]string{
	"Sanskrit.md":    sanskrit,
	"decks/Roots.md": roots,
	"decks/Terms.md": terms,
}

// settings is what a preset holds, as a write hands them over.
func settings() *api.Settings {
	return &api.Settings{
		Goal:        api.GoalMinutesADay,
		Counts:      api.BudgetUnitCards,
		MinutesADay: 20,
		NewADay:     8,
		ReviewsADay: 45,
		Retention:   0.87,
		Learned:     api.RuleInterval,
		Interval:    21,
		Load:        map[string]int32{"sat": 50, "sun": 0},
		HasEvenLoad: true,
	}
}

// TestASettingOutsideItsBoundsIsNotWritten. A retention target outside what
// memory does is a number to correct, and the file is not opened to find out.
func TestASettingOutsideItsBoundsIsNotSaved(t *testing.T) {
	f := newScheduling(t, pointed)

	targetSettings := settings()
	targetSettings.Retention = 1.5
	_, err := f.client.WritePreset(t.Context(), &api.WritePresetRequest{
		Path: "Sanskrit.md", Settings: targetSettings,
	})
	if code := codeOf(err); code != server.CodeInvalidArgument {
		t.Errorf("a retention of 1.5 was answered %v", code)
	}
	if onDiskContent := onDisk(t, f.root, "Sanskrit.md"); onDiskContent != sanskrit {
		t.Errorf("the preset on disk is now %q", onDiskContent)
	}
}

func TestWritingAPresetPreservesAloneOneThatChangedSinceItWasRead(t *testing.T) {
	f := newScheduling(t, pointed)

	read, err := f.client.ReadPreset(t.Context(), &api.ReadPresetRequest{
		Path: "Sanskrit.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	concurrentEdit := sanskrit + "\nThree decks point here.\n"
	if err := os.WriteFile(filepath.Join(f.root, "Sanskrit.md"), []byte(concurrentEdit), 0o644); err != nil {
		t.Fatal(err)
	}

	targetSettings := settings()
	targetSettings.MinutesADay = 35
	answer, err := f.client.WritePreset(t.Context(), &api.WritePresetRequest{
		Path: "Sanskrit.md", Settings: targetSettings, Baseline: read.At,
	})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Error != api.ErrorCodeStale {
		t.Errorf("a write over a preset the person had edited answered %+v", answer)
	}
	if onDiskContent := onDisk(t, f.root, "Sanskrit.md"); onDiskContent != concurrentEdit {
		t.Errorf("the preset on disk is now %q", onDiskContent)
	}
}

func TestADeckNamingNoPresetIsScheduledByTheDefaults(t *testing.T) {
	f := newScheduling(t, pointed)

	answer, err := f.client.GetDeckPreset(t.Context(), &api.GetDeckPresetRequest{
		Deck: "decks/Terms.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Error; code != api.ErrorCodeUnspecified {
		t.Fatalf("a deck naming no preset failed %v", code)
	}
	if path := answer.Preset.Path; path != "" {
		t.Errorf("a deck naming no preset was answered from %q", path)
	}
	actualSettings := answer.Preset.Settings
	if actualSettings.Goal != api.GoalMinutesADay {
		t.Errorf("the defaults steer %v", actualSettings.Goal)
	}
	if actualSettings.MinutesADay != 20 || actualSettings.NewADay != 10 || actualSettings.ReviewsADay != 200 {
		t.Errorf("the defaults are %+v", actualSettings)
	}
	if actualSettings.Retention != 0.9 || !actualSettings.HasEvenLoad {
		t.Errorf("the defaults are %+v", actualSettings)
	}
}

// TestADeckWhosePresetLinkHasNoRoleIsReportedSo: an entry of the `links:` block
// without a role is ignored, so a preset named in it schedules nothing and the
// client reports why the deck uses the defaults.
func TestADeckWhosePresetLinkHasNoRoleIsReportedSo(t *testing.T) {
	f := newScheduling(t, map[string]string{
		"Sanskrit.md": sanskrit,
		"decks/Roots.md": "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    type: preset\n---\n\n" +
			"## Root ^k7m2xq9fzp\n",
	})

	answer, err := f.client.GetDeckPreset(t.Context(), &api.GetDeckPresetRequest{
		Deck: "decks/Roots.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if path := answer.Preset.Path; path != "" {
		t.Fatalf("a deck whose link has no role was answered from %q", path)
	}
	response := answer.Preset.Problems
	if len(response) != 1 || !strings.Contains(response[0], "no role") {
		t.Errorf("the deck is shown %v", response)
	}
}

// counting is a preset spending its budget on every showing.
const counting = "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
	"new_a_day: 8\nreviews_a_day: 45\nretention: 0.87\ncounts: shows\n---\n\n# Shows\n"

// TestWhatABudgetCountsSurvivesAReadAndAWrite: a preset whose budget counts
// showings is read, written unchanged, and still counts showings.
func TestWhatABudgetCountsSurvivesAReadAndAWrite(t *testing.T) {
	f := newScheduling(t, map[string]string{"Shows.md": counting})

	read, err := f.client.ReadPreset(t.Context(), &api.ReadPresetRequest{
		Path: "Shows.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	presetSettings := read.Preset.Settings
	if presetSettings.Counts != api.BudgetUnitShows {
		t.Fatalf("the preset was read as counting %v", presetSettings.Counts)
	}

	if _, err := f.client.WritePreset(t.Context(), &api.WritePresetRequest{
		Path: "Shows.md", Settings: presetSettings, Baseline: read.At,
	}); err != nil {
		t.Fatal(err)
	}
	if raw := onDisk(t, f.root, "Shows.md"); !strings.Contains(raw, "counts: shows") {
		t.Errorf("the preset on disk is now %q", raw)
	}
}

// TestChangingWhatABudgetCountsIsWritten. What a budget counts is settled on
// every save, so a preset moved from one to the other holds what was settled.
func TestChangingWhatABudgetCountsIsSaved(t *testing.T) {
	cards := strings.Replace(counting, "counts: shows", "counts: cards", 1)
	f := newScheduling(t, map[string]string{"Shows.md": cards})

	read, err := f.client.ReadPreset(t.Context(), &api.ReadPresetRequest{
		Path: "Shows.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	presetSettings := read.Preset.Settings
	presetSettings.Counts = api.BudgetUnitShows

	if _, err := f.client.WritePreset(t.Context(), &api.WritePresetRequest{
		Path: "Shows.md", Settings: presetSettings, Baseline: read.At,
	}); err != nil {
		t.Fatal(err)
	}
	if raw := onDisk(t, f.root, "Shows.md"); !strings.Contains(raw, "counts: shows") {
		t.Errorf("the preset on disk is now %q", raw)
	}

	again, err := f.client.ReadPreset(t.Context(), &api.ReadPresetRequest{
		Path: "Shows.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Preset.Settings.Counts; got != api.BudgetUnitShows {
		t.Errorf("the preset is read back as counting %v", got)
	}
}

// TestAClientNamingNoCountsIsRefused: what a budget counts is written on every
// save, so a save without it would change the preset's setting. The request is
// rejected.
func TestAClientNamingNoCountsIsRefused(t *testing.T) {
	f := newScheduling(t, map[string]string{"Shows.md": counting})

	targetSettings := settings()
	targetSettings.Counts = api.BudgetUnitUnspecified
	_, err := f.client.WritePreset(t.Context(), &api.WritePresetRequest{
		Path: "Shows.md", Settings: targetSettings,
	})
	if code := codeOf(err); code != server.CodeInvalidArgument {
		t.Errorf("a save naming no counts was answered %v", code)
	}
	if onDiskContent := onDisk(t, f.root, "Shows.md"); onDiskContent != counting {
		t.Errorf("the preset on disk is now %q", onDiskContent)
	}
}

// TestACurveReachesBackWithItsTwoPlaces: the control is drawn from the current
// setting to the suggested setting, so a curve missing either place cannot be
// drawn.
func TestACurveReachesBackWithItsTwoPlaces(t *testing.T) {
	f := newScheduling(t, pointed)

	answer, err := f.client.ComputeCurve(t.Context(), &api.ComputeCurveRequest{
		Path: "Sanskrit.md", Settings: settings(),
	})
	if err != nil {
		t.Fatal(err)
	}
	curve := answer.Curve
	if curve.Goal != api.GoalMinutesADay {
		t.Fatalf("the curve is of %v", curve.Goal)
	}
	grid := curve.Grid
	if len(grid) == 0 || len(curve.At) != len(grid) {
		t.Fatalf("the curve has %d places on its grid and %d points",
			len(grid), len(curve.At))
	}
	for _, place := range []struct {
		what  string
		point *api.CurvePoint
	}{
		{"where the preset stands", curve.Now},
		{"what is suggested", curve.Suggested},
	} {
		if place.point.At < 0 || int(place.point.At) >= len(grid) {
			t.Errorf("%s is at place %d of %d", place.what, place.point.At, len(grid))
		}
	}
	if value := curve.Now.Value; value != 20 {
		t.Errorf("preset retention limit is %v", value)
	}
}

// TestThePresetsOfTheVaultAreListed. The list a deck's preset is chosen from is
// every preset the vault holds, and the defaults are no note.
func TestThePresetsOfTheVaultAreListed(t *testing.T) {
	f := newScheduling(t, map[string]string{
		"Sanskrit.md":     sanskrit,
		"presets/Slow.md": "---\ntype: preset\ntitle: Slow going\n---\n\n# Slow going\n",
		"decks/Roots.md":  roots,
	})

	answer, err := f.client.ListPresets(t.Context(), &api.ListPresetsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	presets := answer.Presets
	if len(presets) != 2 {
		t.Fatalf("listed %+v", presets)
	}
	if presets[0].Path != "Sanskrit.md" || presets[0].Title != "Sanskrit" {
		t.Errorf("the first is %+v", presets[0])
	}
	if presets[1].Path != "presets/Slow.md" || presets[1].Title != "Slow going" {
		t.Errorf("the second is %+v", presets[1])
	}
}

// TestAPresetMadeIsAPresetToRead: a preset is created where no file existed,
// and has the preset type from the moment it exists. It writes none of its
// settings, so every setting takes its default.
func TestAPresetMadeIsAPresetToRead(t *testing.T) {
	f := newScheduling(t, pointed)

	made, err := f.client.CreatePreset(t.Context(), &api.CreatePresetRequest{
		Title: "Prosody", Path: "presets",
	})
	if err != nil {
		t.Fatal(err)
	}
	if code := made.Error; code != api.ErrorCodeUnspecified {
		t.Fatalf("making a preset answered %v", code)
	}
	if path := made.Path; path != "presets/Prosody.md" {
		t.Fatalf("the preset was filed at %q", path)
	}
	onDiskContent := onDisk(t, f.root, "presets/Prosody.md")
	if !strings.Contains(onDiskContent, "type: preset\n") {
		t.Errorf("the file does not say what it is: %q", onDiskContent)
	}
	for _, key := range []string{"goal", "minutes_a_day", "new_a_day", "reviews_a_day", "load"} {
		if strings.Contains(onDiskContent, key+":") {
			t.Errorf("the file names %s: %q", key, onDiskContent)
		}
	}

	read, err := f.client.ReadPreset(t.Context(), &api.ReadPresetRequest{
		Path: "presets/Prosody.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if code := read.Error; code != api.ErrorCodeUnspecified {
		t.Fatalf("the preset just made failed with %v", code)
	}
	if problems := read.Preset.Problems; len(problems) != 0 {
		t.Errorf("new preset validation errors: %v", problems)
	}
	if stops := read.Preset.Stops; stops != api.StopReasonNothing {
		t.Errorf("the preset just made schedules nothing, and says %v", stops)
	}
	resolved := read.Preset.Settings
	if resolved.Goal != api.GoalMinutesADay || resolved.MinutesADay != 20 {
		t.Errorf("the goal is %v at %d minutes", resolved.Goal, resolved.MinutesADay)
	}
	if resolved.NewADay != 10 || resolved.ReviewsADay != 200 {
		t.Errorf("a day holds %d new and %d reviews",
			resolved.NewADay, resolved.ReviewsADay)
	}
	if resolved.Retention != 0.9 || resolved.Interval != 21 {
		t.Errorf("retention %g at an interval of %d days",
			resolved.Retention, resolved.Interval)
	}
	if resolved.Counts != api.BudgetUnitCards || resolved.Backlog != 100 {
		t.Errorf("a day counts %v and gives %d to the debt",
			resolved.Counts, resolved.Backlog)
	}
	if len(resolved.Load) != 0 || !resolved.HasEvenLoad {
		t.Errorf("the load is %v, evened %t", resolved.Load, resolved.HasEvenLoad)
	}
}

// TestAPresetMadeIsOneTheVaultLists. The list a deck's preset is chosen from
// holds a preset the moment it is made.
func TestAPresetMadeIsOneTheVaultLists(t *testing.T) {
	f := newScheduling(t, pointed)

	if _, err := f.client.CreatePreset(t.Context(), &api.CreatePresetRequest{
		Title: "Prosody", Path: "presets",
	}); err != nil {
		t.Fatal(err)
	}

	answer, err := f.client.ListPresets(t.Context(), &api.ListPresetsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	items := make([]string, 0, len(answer.Presets))
	for _, one := range answer.Presets {
		items = append(items, one.Path)
	}
	if !slices.Contains(items, "presets/Prosody.md") {
		t.Errorf("the vault lists %v", items)
	}
}

// TestADeckIsPutOnAPresetAndTakenOffAgain. Choosing a preset writes the deck's
// one entry, and choosing the defaults takes it out.
func TestADeckIsInsertOnAPresetAndTakenOffAgain(t *testing.T) {
	f := newScheduling(t, pointed)

	placed, err := f.client.ScheduleDeck(t.Context(), &api.ScheduleDeckRequest{
		Deck: "decks/Terms.md", Preset: "Sanskrit.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if placed.Error != api.ErrorCodeUnspecified {
		t.Fatalf("the write failed: %+v", placed)
	}
	if placed.At == nil {
		t.Error("the write says nothing about the file it made")
	}

	read, err := f.client.GetDeckPreset(t.Context(), &api.GetDeckPresetRequest{
		Deck: "decks/Terms.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if read.Preset.Path != "Sanskrit.md" {
		t.Errorf("the deck is scheduled by %+v", read.Preset)
	}
	if read.Preset.Settings.MinutesADay != 20 {
		t.Errorf("the settings are %+v", read.Preset.Settings)
	}

	// The file stat the write answered with is what the next write presents.
	off, err := f.client.ScheduleDeck(t.Context(), &api.ScheduleDeckRequest{
		Deck: "decks/Terms.md", Baseline: placed.At,
	})
	if err != nil {
		t.Fatal(err)
	}
	if off.Error == api.ErrorCodeStale {
		t.Fatal("the deck this caller had just written was answered as changed")
	}
	if onDiskContent := onDisk(t, f.root, "decks/Terms.md"); strings.Contains(onDiskContent, "preset") {
		t.Errorf("the deck still names a preset: %q", onDiskContent)
	}

	again, err := f.client.GetDeckPreset(t.Context(), &api.GetDeckPresetRequest{
		Deck: "decks/Terms.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.Preset.Path != "" {
		t.Errorf("the deck is scheduled by %+v", again.Preset)
	}
}

// TestADeckIsNotScheduledByANoteThatIsNotAPreset: a note that is not a preset
// schedules nothing, so the client receives an error and the deck is unchanged.
func TestADeckIsNotScheduledByANoteThatIsNotAPreset(t *testing.T) {
	f := newScheduling(t, map[string]string{
		"Sanskrit.md":    sanskrit,
		"Grammar.md":     "---\ntype: note\n---\n\n# Grammar\n",
		"decks/Terms.md": terms,
	})

	answer, err := f.client.ScheduleDeck(t.Context(), &api.ScheduleDeckRequest{
		Deck: "decks/Terms.md", Preset: "Grammar.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Error != api.ErrorCodeNotAPreset {
		t.Errorf("the write was answered %v", answer.Error)
	}
	if onDiskContent := onDisk(t, f.root, "decks/Terms.md"); onDiskContent != terms {
		t.Errorf("the deck on disk is now %q", onDiskContent)
	}
}

// TestSchedulingAPresetThatIsNotThere. A path the vault holds no note at is
func TestSchedulingAPresetThatIsNotThere(t *testing.T) {
	f := newScheduling(t, pointed)

	answer, err := f.client.ScheduleDeck(t.Context(), &api.ScheduleDeckRequest{
		Deck: "decks/Terms.md", Preset: "Pali.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Error != api.ErrorCodeMissing {
		t.Errorf("the write was answered %v", answer.Error)
	}
	if onDiskContent := onDisk(t, f.root, "decks/Terms.md"); onDiskContent != terms {
		t.Errorf("the deck on disk is now %q", onDiskContent)
	}
}

// TestSchedulingPreservesAloneADeckThatChangedSinceItWasRead: a concurrent
// edit of the deck takes precedence over a client holding an outdated read.
func TestSchedulingPreservesAloneADeckThatChangedSinceItWasRead(t *testing.T) {
	f := newScheduling(t, pointed)

	read, err := f.client.ScheduleDeck(t.Context(), &api.ScheduleDeckRequest{
		Deck: "decks/Roots.md", Preset: "Sanskrit.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	concurrentEdit := roots + "\n## Another ^zpqrstvwxy\n"
	if err := os.WriteFile(
		filepath.Join(f.root, "decks", "Roots.md"), []byte(concurrentEdit), 0o644); err != nil {
		t.Fatal(err)
	}

	answer, err := f.client.ScheduleDeck(t.Context(), &api.ScheduleDeckRequest{
		Deck: "decks/Roots.md", Baseline: read.At,
	})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Error != api.ErrorCodeStale {
		t.Errorf("a write over a deck the person had edited answered %+v", answer)
	}
	if onDiskContent := onDisk(t, f.root, "decks/Roots.md"); onDiskContent != concurrentEdit {
		t.Errorf("the deck on disk is now %q", onDiskContent)
	}
}
