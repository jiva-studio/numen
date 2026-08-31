package webui_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if !answer.Msg.GetChanged() {
		t.Error("a write over a preset the person had edited was not answered as changed")
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

	answer, err := f.client.Scheduling(t.Context(), connect.NewRequest(&v1.SchedulingRequest{
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

	answer, err := f.client.Scheduling(t.Context(), connect.NewRequest(&v1.SchedulingRequest{
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

// TestACurveComesBackWithItsTwoMarks. The control is drawn from where the
// preset stands to what is suggested, so a curve that came back without both
// marks is a control with nothing to point at.
func TestACurveComesBackWithItsTwoMarks(t *testing.T) {
	f := steering(t, pointed)

	answer, err := f.client.Curve(t.Context(), connect.NewRequest(&v1.CurveRequest{
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
	for _, mark := range []struct {
		what string
		at   *v1.Mark
	}{
		{"where the preset stands", curve.GetNow()},
		{"what is suggested", curve.GetSuggested()},
	} {
		if mark.at.GetAt() < 0 || int(mark.at.GetAt()) >= len(grid) {
			t.Errorf("%s is at place %d of %d", mark.what, mark.at.GetAt(), len(grid))
		}
	}
	if value := curve.GetNow().GetValue(); value != 20 {
		t.Errorf("the preset keeps 20 minutes a day and stands at %v", value)
	}
}
