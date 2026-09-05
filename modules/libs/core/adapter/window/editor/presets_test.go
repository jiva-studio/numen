package editor_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// A preset says how the decks pointing at it are scheduled. What the window
// asks about one is what these say.

// scheduling is a vault with the notes given in it, and a client asking about
// its presets the way the window does.
type scheduling struct {
	client numenv1connect.PresetsServiceClient
	root   string
}

func steering(t *testing.T, notes map[string]string) *scheduling {
	t.Helper()

	f := quitting(t, nil, notes)
	scanned(t, f)

	route, handler := numenv1connect.NewPresetsServiceHandler(f.opened.API)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &scheduling{
		client: numenv1connect.NewPresetsServiceClient(server.Client(), server.URL),
		root:   f.root,
	}
}

// sanskrit is a preset a person wrote, with a deck pointing at it and a deck
// pointing at nothing.
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
func settings() *v1.Settings {
	return &v1.Settings{
		Goal:        v1.Goal_GOAL_MINUTES_A_DAY,
		Counts:      v1.Counts_COUNTS_CARDS,
		MinutesADay: 20,
		NewADay:     8,
		ReviewsADay: 45,
		Retention:   0.87,
		Learned:     v1.Rule_RULE_INTERVAL,
		Interval:    21,
		Load:        map[string]int32{"sat": 50, "sun": 0},
		EvenLoad:    true,
	}
}

// TestASettingOutsideItsBoundsIsNotWritten. A retention target outside what
// memory does is a number to correct, and the file is not opened to find out.
func TestASettingOutsideItsBoundsIsNotWritten(t *testing.T) {
	f := steering(t, pointed)

	asked := settings()
	asked.Retention = 1.5
	_, err := f.client.WritePreset(t.Context(), connect.NewRequest(&v1.WritePresetRequest{
		Path: "Sanskrit.md", Settings: asked,
	}))
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Errorf("a retention of 1.5 was answered %v", code)
	}
	if held := onDisk(t, f.root, "Sanskrit.md"); held != sanskrit {
		t.Errorf("the preset on disk is now %q", held)
	}
}

// TestWritingAPresetLeavesAloneOneThatChangedSinceItWasRead. Somebody editing
// their own preset outranks a client that read it, thought about it and arrived
// late.
func TestWritingAPresetLeavesAloneOneThatChangedSinceItWasRead(t *testing.T) {
	f := steering(t, pointed)

	read, err := f.client.ReadPreset(t.Context(), connect.NewRequest(&v1.ReadPresetRequest{
		Path: "Sanskrit.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	// The person writes their own preset while the client is thinking about
	// what it read.
	theirs := sanskrit + "\nThree decks point here.\n"
	if err := os.WriteFile(filepath.Join(f.root, "Sanskrit.md"), []byte(theirs), 0o644); err != nil {
		t.Fatal(err)
	}

	asked := settings()
	asked.MinutesADay = 35
	answer, err := f.client.WritePreset(t.Context(), connect.NewRequest(&v1.WritePresetRequest{
		Path: "Sanskrit.md", Settings: asked, Seen: read.Msg.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_STALE {
		t.Errorf("a write over a preset the person had edited answered %+v", answer.Msg)
	}
	if held := onDisk(t, f.root, "Sanskrit.md"); held != theirs {
		t.Errorf("the preset on disk is now %q", held)
	}
}

// TestADeckNamingNoPresetIsScheduledByTheDefaults. A vault holding no preset at
// all schedules every deck, so a deck pointing at nothing is answered with
// settings and not with a refusal.
func TestADeckNamingNoPresetIsScheduledByTheDefaults(t *testing.T) {
	f := steering(t, pointed)

	answer, err := f.client.GetDeckPreset(t.Context(), connect.NewRequest(&v1.GetDeckPresetRequest{
		Deck: "decks/Terms.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("a deck naming no preset was refused %v", refusal)
	}
	if path := answer.Msg.GetPreset().GetPath(); path != "" {
		t.Errorf("a deck naming no preset was answered from %q", path)
	}
	held := answer.Msg.GetPreset().GetSettings()
	if held.GetGoal() != v1.Goal_GOAL_MINUTES_A_DAY {
		t.Errorf("the defaults steer %v", held.GetGoal())
	}
	if held.GetMinutesADay() != 20 || held.GetNewADay() != 10 || held.GetReviewsADay() != 200 {
		t.Errorf("the defaults are %+v", held)
	}
	if held.GetRetention() != 0.9 || !held.GetEvenLoad() {
		t.Errorf("the defaults are %+v", held)
	}
}

// TestADeckWhosePresetLinkHasNoRoleIsToldSo. An entry of the `links:` block
// written with no role is not read, so a preset named in one schedules nothing
// and the window says why the deck is on the defaults.
func TestADeckWhosePresetLinkHasNoRoleIsToldSo(t *testing.T) {
	f := steering(t, map[string]string{
		"Sanskrit.md": sanskrit,
		"decks/Roots.md": "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    type: preset\n---\n\n" +
			"## Root ^k7m2xq9fzp\n",
	})

	answer, err := f.client.GetDeckPreset(t.Context(), connect.NewRequest(&v1.GetDeckPresetRequest{
		Deck: "decks/Roots.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if path := answer.Msg.GetPreset().GetPath(); path != "" {
		t.Fatalf("a deck whose link has no role was answered from %q", path)
	}
	said := answer.Msg.GetPreset().GetProblems()
	if len(said) != 1 || !strings.Contains(said[0], "no role") {
		t.Errorf("the deck is shown %v", said)
	}
}

// counting is a preset spending its budget on every showing.
const counting = "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
	"new_a_day: 8\nreviews_a_day: 45\nretention: 0.87\ncounts: shows\n---\n\n# Shows\n"

// TestWhatABudgetCountsSurvivesAReadAndAWrite. A preset spending its budget per
// showing is read, written back as it was read, and still says so.
func TestWhatABudgetCountsSurvivesAReadAndAWrite(t *testing.T) {
	f := steering(t, map[string]string{"Shows.md": counting})

	read, err := f.client.ReadPreset(t.Context(), connect.NewRequest(&v1.ReadPresetRequest{
		Path: "Shows.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	held := read.Msg.GetPreset().GetSettings()
	if held.GetCounts() != v1.Counts_COUNTS_SHOWS {
		t.Fatalf("the preset was read as counting %v", held.GetCounts())
	}

	if _, err := f.client.WritePreset(t.Context(), connect.NewRequest(&v1.WritePresetRequest{
		Path: "Shows.md", Settings: held, Seen: read.Msg.GetAt(),
	})); err != nil {
		t.Fatal(err)
	}
	if raw := onDisk(t, f.root, "Shows.md"); !strings.Contains(raw, "counts: shows") {
		t.Errorf("the preset on disk is now %q", raw)
	}
}

// TestChangingWhatABudgetCountsIsWritten. What a budget counts is settled on
// every save, so a preset moved from one to the other holds what was settled.
func TestChangingWhatABudgetCountsIsWritten(t *testing.T) {
	cards := strings.Replace(counting, "counts: shows", "counts: cards", 1)
	f := steering(t, map[string]string{"Shows.md": cards})

	read, err := f.client.ReadPreset(t.Context(), connect.NewRequest(&v1.ReadPresetRequest{
		Path: "Shows.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	held := read.Msg.GetPreset().GetSettings()
	held.Counts = v1.Counts_COUNTS_SHOWS

	if _, err := f.client.WritePreset(t.Context(), connect.NewRequest(&v1.WritePresetRequest{
		Path: "Shows.md", Settings: held, Seen: read.Msg.GetAt(),
	})); err != nil {
		t.Fatal(err)
	}
	if raw := onDisk(t, f.root, "Shows.md"); !strings.Contains(raw, "counts: shows") {
		t.Errorf("the preset on disk is now %q", raw)
	}

	again, err := f.client.ReadPreset(t.Context(), connect.NewRequest(&v1.ReadPresetRequest{
		Path: "Shows.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Msg.GetPreset().GetSettings().GetCounts(); got != v1.Counts_COUNTS_SHOWS {
		t.Errorf("the preset is read back as counting %v", got)
	}
}

// TestAClientNamingNoCountsIsRefused. What a budget counts is written on every
// save, so a save that named none would move a preset off what the person set
// it to.
func TestAClientNamingNoCountsIsRefused(t *testing.T) {
	f := steering(t, map[string]string{"Shows.md": counting})

	asked := settings()
	asked.Counts = v1.Counts_COUNTS_UNSPECIFIED
	_, err := f.client.WritePreset(t.Context(), connect.NewRequest(&v1.WritePresetRequest{
		Path: "Shows.md", Settings: asked,
	}))
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Errorf("a save naming no counts was answered %v", code)
	}
	if held := onDisk(t, f.root, "Shows.md"); held != counting {
		t.Errorf("the preset on disk is now %q", held)
	}
}

// TestACurveComesBackWithItsTwoPlaces. The control is drawn from where the
// preset stands to what is suggested, so a curve that came back without both
// places is a control with nothing to point at.
func TestACurveComesBackWithItsTwoPlaces(t *testing.T) {
	f := steering(t, pointed)

	answer, err := f.client.ComputeCurve(t.Context(), connect.NewRequest(&v1.ComputeCurveRequest{
		Path: "Sanskrit.md", Settings: settings(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	curve := answer.Msg.GetCurve()
	if curve.GetGoal() != v1.Goal_GOAL_MINUTES_A_DAY {
		t.Fatalf("the curve is of %v", curve.GetGoal())
	}
	grid := curve.GetGrid()
	if len(grid) == 0 || len(curve.GetAt()) != len(grid) {
		t.Fatalf("the curve has %d places on its grid and %d points",
			len(grid), len(curve.GetAt()))
	}
	for _, place := range []struct {
		what string
		at   *v1.Place
	}{
		{"where the preset stands", curve.GetNow()},
		{"what is suggested", curve.GetSuggested()},
	} {
		if place.at.GetAt() < 0 || int(place.at.GetAt()) >= len(grid) {
			t.Errorf("%s is at place %d of %d", place.what, place.at.GetAt(), len(grid))
		}
	}
	if value := curve.GetNow().GetValue(); value != 20 {
		t.Errorf("the preset keeps 20 minutes a day and stands at %v", value)
	}
}

// TestThePresetsOfTheVaultAreListed. The list a deck's preset is chosen from is
// every preset the vault holds, and the defaults are no note.
func TestThePresetsOfTheVaultAreListed(t *testing.T) {
	f := steering(t, map[string]string{
		"Sanskrit.md":     sanskrit,
		"presets/Slow.md": "---\ntype: preset\ntitle: Slow going\n---\n\n# Slow going\n",
		"decks/Roots.md":  roots,
	})

	answer, err := f.client.ListPresets(t.Context(), connect.NewRequest(&v1.ListPresetsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	held := answer.Msg.GetPresets()
	if len(held) != 2 {
		t.Fatalf("listed %+v", held)
	}
	if held[0].GetPath() != "Sanskrit.md" || held[0].GetTitle() != "Sanskrit" {
		t.Errorf("the first is %+v", held[0])
	}
	if held[1].GetPath() != "presets/Slow.md" || held[1].GetTitle() != "Slow going" {
		t.Errorf("the second is %+v", held[1])
	}
}

// TestAPresetMadeIsAPresetToRead. A preset is made where there was no file, and
// it says it is a preset from the moment it exists. It names none of its
// settings, so the read that follows stands at the defaults.
func TestAPresetMadeIsAPresetToRead(t *testing.T) {
	f := steering(t, pointed)

	made, err := f.client.CreatePreset(t.Context(), connect.NewRequest(&v1.CreatePresetRequest{
		Title: "Prosody", Folder: "presets",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := made.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("making a preset answered %v", refusal)
	}
	if path := made.Msg.GetPath(); path != "presets/Prosody.md" {
		t.Fatalf("the preset was filed at %q", path)
	}
	held := onDisk(t, f.root, "presets/Prosody.md")
	if !strings.Contains(held, "type: preset\n") {
		t.Errorf("the file does not say what it is: %q", held)
	}
	for _, key := range []string{"goal", "minutes_a_day", "new_a_day", "reviews_a_day", "load"} {
		if strings.Contains(held, key+":") {
			t.Errorf("the file names %s: %q", key, held)
		}
	}

	read, err := f.client.ReadPreset(t.Context(), connect.NewRequest(&v1.ReadPresetRequest{
		Path: "presets/Prosody.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := read.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the preset just made was refused with %v", refusal)
	}
	if problems := read.Msg.GetPreset().GetProblems(); len(problems) != 0 {
		t.Errorf("the preset just made stands against %v", problems)
	}
	if stops := read.Msg.GetPreset().GetStops(); stops != v1.StopReason_STOP_REASON_NOTHING {
		t.Errorf("the preset just made schedules nothing, and says %v", stops)
	}
	settled := read.Msg.GetPreset().GetSettings()
	if settled.GetGoal() != v1.Goal_GOAL_MINUTES_A_DAY || settled.GetMinutesADay() != 20 {
		t.Errorf("the goal is %v at %d minutes", settled.GetGoal(), settled.GetMinutesADay())
	}
	if settled.GetNewADay() != 10 || settled.GetReviewsADay() != 200 {
		t.Errorf("a day holds %d new and %d reviews",
			settled.GetNewADay(), settled.GetReviewsADay())
	}
	if settled.GetRetention() != 0.9 || settled.GetInterval() != 21 {
		t.Errorf("retention %g at an interval of %d days",
			settled.GetRetention(), settled.GetInterval())
	}
	if settled.GetCounts() != v1.Counts_COUNTS_CARDS || settled.GetBacklog() != 100 {
		t.Errorf("a day counts %v and gives %d to the debt",
			settled.GetCounts(), settled.GetBacklog())
	}
	if len(settled.GetLoad()) != 0 || !settled.GetEvenLoad() {
		t.Errorf("the load is %v, evened %t", settled.GetLoad(), settled.GetEvenLoad())
	}
}

// TestAPresetMadeIsOneTheVaultLists. The list a deck's preset is chosen from
// holds a preset the moment it is made.
func TestAPresetMadeIsOneTheVaultLists(t *testing.T) {
	f := steering(t, pointed)

	if _, err := f.client.CreatePreset(t.Context(), connect.NewRequest(&v1.CreatePresetRequest{
		Title: "Prosody", Folder: "presets",
	})); err != nil {
		t.Fatal(err)
	}

	answer, err := f.client.ListPresets(t.Context(), connect.NewRequest(&v1.ListPresetsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	listed := make([]string, 0, len(answer.Msg.GetPresets()))
	for _, one := range answer.Msg.GetPresets() {
		listed = append(listed, one.GetPath())
	}
	if !slices.Contains(listed, "presets/Prosody.md") {
		t.Errorf("the vault lists %v", listed)
	}
}

// TestADeckIsPutOnAPresetAndTakenOffAgain. Choosing a preset writes the deck's
// one entry, and choosing the defaults takes it out.
func TestADeckIsPutOnAPresetAndTakenOffAgain(t *testing.T) {
	f := steering(t, pointed)

	put, err := f.client.ScheduleDeck(t.Context(), connect.NewRequest(&v1.ScheduleDeckRequest{
		Deck: "decks/Terms.md", Preset: "Sanskrit.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if put.Msg.GetRefusal() != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the write was refused: %+v", put.Msg)
	}
	if put.Msg.GetAt() == nil {
		t.Error("the write says nothing about the file it made")
	}

	read, err := f.client.GetDeckPreset(t.Context(), connect.NewRequest(&v1.GetDeckPresetRequest{
		Deck: "decks/Terms.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if read.Msg.GetPreset().GetPath() != "Sanskrit.md" {
		t.Errorf("the deck is scheduled by %+v", read.Msg.GetPreset())
	}
	if read.Msg.GetPreset().GetSettings().GetMinutesADay() != 20 {
		t.Errorf("the settings are %+v", read.Msg.GetPreset().GetSettings())
	}

	// The fingerprint the write answered with is what the next write presents.
	off, err := f.client.ScheduleDeck(t.Context(), connect.NewRequest(&v1.ScheduleDeckRequest{
		Deck: "decks/Terms.md", Seen: put.Msg.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if off.Msg.GetRefusal() == v1.Refusal_REFUSAL_STALE {
		t.Fatal("the deck this caller had just written was answered as changed")
	}
	if held := onDisk(t, f.root, "decks/Terms.md"); strings.Contains(held, "preset") {
		t.Errorf("the deck still names a preset: %q", held)
	}

	again, err := f.client.GetDeckPreset(t.Context(), connect.NewRequest(&v1.GetDeckPresetRequest{
		Deck: "decks/Terms.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if again.Msg.GetPreset().GetPath() != "" {
		t.Errorf("the deck is scheduled by %+v", again.Msg.GetPreset())
	}
}

// TestADeckIsNotScheduledByANoteThatIsNotAPreset. A note that is not a preset
// schedules nothing, so the client is told and the deck is left alone.
func TestADeckIsNotScheduledByANoteThatIsNotAPreset(t *testing.T) {
	f := steering(t, map[string]string{
		"Sanskrit.md":    sanskrit,
		"Grammar.md":     "---\ntype: note\n---\n\n# Grammar\n",
		"decks/Terms.md": terms,
	})

	answer, err := f.client.ScheduleDeck(t.Context(), connect.NewRequest(&v1.ScheduleDeckRequest{
		Deck: "decks/Terms.md", Preset: "Grammar.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_NOT_A_PRESET {
		t.Errorf("the write was answered %v", answer.Msg.GetRefusal())
	}
	if held := onDisk(t, f.root, "decks/Terms.md"); held != terms {
		t.Errorf("the deck on disk is now %q", held)
	}
}

// TestSchedulingAPresetThatIsNotThere. A path the vault holds no note at is
// refused, and nothing is written.
func TestSchedulingAPresetThatIsNotThere(t *testing.T) {
	f := steering(t, pointed)

	answer, err := f.client.ScheduleDeck(t.Context(), connect.NewRequest(&v1.ScheduleDeckRequest{
		Deck: "decks/Terms.md", Preset: "Pali.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_MISSING {
		t.Errorf("the write was answered %v", answer.Msg.GetRefusal())
	}
	if held := onDisk(t, f.root, "decks/Terms.md"); held != terms {
		t.Errorf("the deck on disk is now %q", held)
	}
}

// TestSchedulingLeavesAloneADeckThatChangedSinceItWasRead. Somebody editing
// their own deck outranks a client that read it and arrived late.
func TestSchedulingLeavesAloneADeckThatChangedSinceItWasRead(t *testing.T) {
	f := steering(t, pointed)

	read, err := f.client.ScheduleDeck(t.Context(), connect.NewRequest(&v1.ScheduleDeckRequest{
		Deck: "decks/Roots.md", Preset: "Sanskrit.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	theirs := roots + "\n## Another ^zpqrstvwxy\n"
	if err := os.WriteFile(
		filepath.Join(f.root, "decks", "Roots.md"), []byte(theirs), 0o644); err != nil {
		t.Fatal(err)
	}

	answer, err := f.client.ScheduleDeck(t.Context(), connect.NewRequest(&v1.ScheduleDeckRequest{
		Deck: "decks/Roots.md", Seen: read.Msg.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_STALE {
		t.Errorf("a write over a deck the person had edited answered %+v", answer.Msg)
	}
	if held := onDisk(t, f.root, "decks/Roots.md"); held != theirs {
		t.Errorf("the deck on disk is now %q", held)
	}
}
