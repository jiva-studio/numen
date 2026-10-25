package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
)

// request names is worth nothing when the path alone tells the two apart.
var elsewhere = map[string]string{
	"Term.md": deck["Term.md"],
	"Grammar.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 5\n" +
		"new_a_day: 3\nreviews_a_day: 11\n---\n\n# Grammar\n",
	"decks/Words.md": "---\ntype: deck\nlinks:\n" +
		"  - to: Grammar\n    role: ref\n    type: preset\n---\n" +
		"\n## Yaffle ^zpqrstvwxy\n\n[[Term]]\n\n### Word\n\nYaffle\n" +
		"\n### Meaning\n\nA green woodpecker\n",
}

// getDeckPreset is the preset one vault's deck is scheduled by.
func getDeckPreset(t *testing.T, api *API, v domain.Vault, path string) *v1.Preset {
	t.Helper()
	out, err := api.GetVaultDeckPreset(t.Context(), connect.NewRequest(
		&v1.GetVaultDeckPresetRequest{Vault: string(v.ID), Deck: path}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("%s failed: %v", path, out.Msg.GetError())
	}
	return out.Msg.GetPreset()
}

// The preset a deck is scheduled by is the one the vault named in the request
// holds. The window is over every vault at once, and the two decks here stand
// at the same path.
func TestAPresetIsReadFromTheVaultTheRequestNames(t *testing.T) {
	api, target := newAPI(t, pointed, elsewhere)
	one, two := target[0], target[1]

	first := getDeckPreset(t, api, one, "decks/Words.md")
	if first.GetPath() != "Sanskrit.md" || first.GetTitle() != "Sanskrit" {
		t.Errorf("the first vault is scheduled by %+v", first)
	}
	if first.GetSettings().GetReviewsADay() != 45 {
		t.Errorf("the first vault's settings are %+v", first.GetSettings())
	}

	second := getDeckPreset(t, api, two, "decks/Words.md")
	if second.GetPath() != "Grammar.md" || second.GetTitle() != "Grammar" {
		t.Errorf("the second vault is scheduled by %+v", second)
	}
	if second.GetSettings().GetReviewsADay() != 11 {
		t.Errorf("the second vault's settings are %+v", second.GetSettings())
	}
}

// handwritten is a vault holding a preset that no deck names yet.
var handwritten = map[string]string{
	"Term.md":        deck["Term.md"],
	"Sanskrit.md":    pointed["Sanskrit.md"],
	"decks/Words.md": deck["decks/Words.md"],
}

// typedByHand is that deck as a person left it: pointed at the preset, and
// holding a card they typed and gave no mark.
const typedByHand = "---\ntype: deck\nlinks:\n" +
	"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n" +
	"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
	"\n### Meaning\n\nCompost made of fallen leaves alone\n" +
	"\n## Yaffle\n\n[[Term]]\n\n### Word\n\nYaffle\n" +
	"\n### Meaning\n\nA green woodpecker\n"

// Starting a session gives a mark to the card that carries none, and the deck it
// writes is level in the index by the time the session is handed over. Nothing
// else is running over the vault, and nothing rescans it.
func TestAMarkGivenOnSessionDownLevelsTheDeck(t *testing.T) {
	api, target := newAPI(t, handwritten)
	v := target[0]

	pos := filepath.Join(v.Path, "decks", "Words.md")
	if err := os.WriteFile(pos, []byte(typedByHand), 0o644); err != nil {
		t.Fatal(err)
	}

	// The index still holds the deck as it was scanned, so it names no preset.
	if before := getDeckPreset(t, api, v, "decks/Words.md"); before.GetPath() != "" {
		t.Fatalf("the deck already is under %+v", before)
	}

	startSession(t, api, v)

	content, err := os.ReadFile(pos)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "## Yaffle ^") {
		t.Fatalf("the card was given no mark: %q", content)
	}

	after := getDeckPreset(t, api, v, "decks/Words.md")
	if after.GetPath() != "Sanskrit.md" || after.GetTitle() != "Sanskrit" {
		t.Errorf("the deck is under %+v", after)
	}
}

// A question about the presets of a vault this installation does not hold is
// rejected, and nothing is read for it.
func TestAQuestionAboutThePresetsOfAVaultNobodyContainsIsRefused(t *testing.T) {
	api, _ := newAPI(t, pointed)

	_, err := api.GetVaultDeckPreset(t.Context(), connect.NewRequest(
		&v1.GetVaultDeckPresetRequest{Vault: "nobody", Deck: "decks/Words.md"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("reading answered %v", err)
	}
}

// brokenPresetLinks is an index that cannot say what a deck points at.
type brokenPresetLinks struct{ ports.LinkQueries }

func (brokenPresetLinks) GetLinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return nil, errors.New("the index is out of reach")
}

// The index being out of reach is an internal failure and not a deck on the
// defaults.
func TestAnIndexOutOfReachIsAnInternalFailureForAPreset(t *testing.T) {
	api, target := newAPI(t, pointed)
	api.scenarios().flash.Presets = flashcards.NewPresets(nil, nil, brokenPresetLinks{}, nil, nil, api.scenarios().day, nil)

	_, err := api.GetVaultDeckPreset(t.Context(), connect.NewRequest(
		&v1.GetVaultDeckPresetRequest{Vault: string(target[0].ID), Deck: "decks/Words.md"}))
	if code := connect.CodeOf(err); code != connect.CodeInternal {
		t.Errorf("reading answered %v", err)
	}
}

// A preset the deck names that cannot be read comes back rejected as the editor
// refuses a note, and carries no preset.
func TestAPresetThatCannotBeReadIsRefusedAsTheEditorRefusesIt(t *testing.T) {
	api, target := newAPI(t, pointed)
	v := target[0]
	if err := os.Remove(filepath.Join(v.Path, "Sanskrit.md")); err != nil {
		t.Fatal(err)
	}

	out, err := api.GetVaultDeckPreset(t.Context(), connect.NewRequest(
		&v1.GetVaultDeckPresetRequest{Vault: string(v.ID), Deck: "decks/Words.md"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetError(); got != v1.ErrorCode_ERROR_CODE_MISSING {
		t.Errorf("a preset that is not there fails as %v", got)
	}
	if out.Msg.GetPreset() != nil {
		t.Errorf("a rejected preset came with %+v", out.Msg.GetPreset())
	}
}

// newWordDeck is a deck of as many cards naming this preset. A deck naming none
// is written with an empty name.
func newWordDeck(pos string, cards int, from int) string {
	out := "---\ntype: deck\n"
	if pos != "" {
		out += "links:\n  - to: " + pos + "\n    role: ref\n    type: preset\n"
	}
	out += "---\n"
	for i := range cards {
		out += fmt.Sprintf(
			"\n## Card %d ^word%06d\n\n[[Term]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n",
			from+i, from+i, from+i, from+i)
	}
	return out
}

// presetted is one vault of two presets a deck each, and a third nothing points
// at. Each preset holds two cards a day.
var presetted = map[string]string{
	"Term.md": deck["Term.md"],
	"Steady.md": "---\ntype: preset\ngoal: retention\n" +
		"new_a_day: 2\nreviews_a_day: 0\n---\n\n# Steady\n",
	"Other.md": "---\ntype: preset\ngoal: retention\n" +
		"new_a_day: 2\nreviews_a_day: 0\n---\n\n# Other\n",
	"Lonely.md": "---\ntype: preset\ngoal: retention\n" +
		"new_a_day: 2\nreviews_a_day: 0\n---\n\n# Lonely\n",
	"decks/Birds.md":  newWordDeck("Steady", 4, 0),
	"decks/Rivers.md": newWordDeck("Other", 4, 100),
}

// newPresetName is the preset a request names, as the window sends it.
func newPresetName(preset string) *string { return &preset }

// The window draws a tile per preset, and pressing one opens a session over the
func TestASessionIsOpenedOverOnePreset(t *testing.T) {
	api, target := newAPI(t, presetted)
	v := target[0]

	out, err := api.StartSession(t.Context(), connect.NewRequest(&v1.StartSessionRequest{
		Vault: string(v.ID), Preset: newPresetName("Steady.md"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	request := out.Msg.GetCards()
	if len(request) != 2 {
		t.Fatalf("a preset of two cards a day offered %d", len(request))
	}
	for _, one := range request {
		if one.GetDeck() != "decks/Birds.md" {
			t.Errorf("the session asked %s, which another preset schedules", one.GetDeck())
		}
	}
}

// Which cards were meant is a question, and the window is asked it again.
func TestNamingADeckAndAPresetTogetherIsRefused(t *testing.T) {
	api, target := newAPI(t, presetted)
	v := target[0]

	_, err := api.StartSession(t.Context(), connect.NewRequest(&v1.StartSessionRequest{
		Vault: string(v.ID), Deck: "decks/Birds.md", Preset: newPresetName("Steady.md"),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("a deck and a preset together were answered with %v", err)
	}
}

// Why a preset schedules nothing is the core's verdict, and every preset the
// nothing else in the answer would tell a window that it is stopped.
func TestEachPresetOnTheFrontDoorReturnsWhyItSchedulesNothing(t *testing.T) {
	// The marks are the ones the application writes, so the cards stand without
	// the vault being sat down to.
	marked := func(pos string, mark string) string {
		return "---\ntype: deck\nlinks:\n  - to: " + pos + "\n    role: ref\n    type: preset\n" +
			"---\n\n## Card ^" + mark + "\n\n[[Term]]\n\n### Word\n\nw\n\n### Meaning\n\nm\n"
	}
	stoppedPresetFiles := map[string]string{
		"Term.md": deck["Term.md"],
		"Steady.md": "---\ntype: preset\ngoal: retention\n" +
			"new_a_day: 2\nreviews_a_day: 2\n---\n\n# Steady\n",
		"Quiet.md": "---\ntype: preset\ngoal: minutes_a_day\n" +
			"minutes_a_day: 0\n---\n\n# Quiet\n",
		"Lonely.md": "---\ntype: preset\ngoal: retention\n" +
			"new_a_day: 0\nreviews_a_day: 0\n---\n\n# Lonely\n",
		"decks/Birds.md":  marked("Steady", "3f4g5h6j7k"),
		"decks/Rivers.md": marked("Quiet", "zpqrstvwxy"),
	}
	api, _ := newAPI(t, stoppedPresetFiles)

	got := make(map[string]v1.StopReason)
	for _, one := range front(t, api).GetVaults()[0].GetPresets() {
		got[one.GetPreset()] = one.GetStopsOn()
	}
	want := map[string]v1.StopReason{
		"Steady.md": v1.StopReason_STOP_REASON_NOTHING,
		"Quiet.md":  v1.StopReason_STOP_REASON_NO_MINUTES,
		"Lonely.md": v1.StopReason_STOP_REASON_NO_CARDS,
	}
	for path, why := range want {
		if got[path] != why {
			t.Errorf("%s schedules nothing for %v, want %v", path, got[path], why)
		}
	}
}

// The preset a deck is scheduled by carries the same verdict, which is what a
// window has before any curve exists.
func TestThePresetOfADeckCarriesWhyItSchedulesNothing(t *testing.T) {
	api, target := newAPI(t, map[string]string{
		"Term.md": deck["Term.md"],
		"Quiet.md": "---\ntype: preset\ngoal: minutes_a_day\n" +
			"minutes_a_day: 0\n---\n\n# Quiet\n",
		"decks/Birds.md": newWordDeck("Quiet", 2, 0),
	})

	out, err := api.GetVaultDeckPreset(t.Context(), connect.NewRequest(
		&v1.GetVaultDeckPresetRequest{
			Vault: string(target[0].ID), Deck: "decks/Birds.md",
		}))
	if err != nil {
		t.Fatal(err)
	}
	one := out.Msg.GetPreset()
	if got := one.GetStops(); got != v1.StopReason_STOP_REASON_NO_MINUTES {
		t.Errorf("the preset schedules nothing for %v", got)
	}
	if got := one.GetStopsOn(); got != v1.StopReason_STOP_REASON_NO_MINUTES {
		t.Errorf("the day schedules nothing for %v", got)
	}
}

// window shows in place of an empty session.
func TestAPresetThatSchedulesNothingIsRefusedWithItsReason(t *testing.T) {
	api, target := newAPI(t, presetted)
	v := target[0]

	_, err := api.StartSession(t.Context(), connect.NewRequest(&v1.StartSessionRequest{
		Vault: string(v.ID), Preset: newPresetName("Lonely.md"),
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("a preset nothing points at was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "no deck") {
		t.Errorf("the reason shown is %q", err)
	}
}
