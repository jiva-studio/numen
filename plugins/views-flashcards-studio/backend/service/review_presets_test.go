package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
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
func getDeckPreset(t *testing.T, handler *API, v domain.Vault, path string) *api.Preset {
	t.Helper()
	out, err := handler.GetVaultDeckPreset(t.Context(),
		&api.GetVaultDeckPresetRequest{Vault: string(v.ID), Deck: path})
	if err != nil {
		t.Fatal(err)
	}
	if out.Error != api.ErrorCodeUnspecified {
		t.Fatalf("%s failed: %v", path, out.Error)
	}
	return out.Preset
}

// The preset a deck is scheduled by is the one the vault named in the request
// holds. The window is over every vault at once, and the two decks here stand
// at the same path.
func TestAPresetIsReadFromTheVaultTheRequestNames(t *testing.T) {
	handler, target := newAPI(t, pointed, elsewhere)
	one, two := target[0], target[1]

	first := getDeckPreset(t, handler, one, "decks/Words.md")
	if first.Path != "Sanskrit.md" || first.Title != "Sanskrit" {
		t.Errorf("the first vault is scheduled by %+v", first)
	}
	if first.Settings.ReviewsADay != 45 {
		t.Errorf("the first vault's settings are %+v", first.Settings)
	}

	second := getDeckPreset(t, handler, two, "decks/Words.md")
	if second.Path != "Grammar.md" || second.Title != "Grammar" {
		t.Errorf("the second vault is scheduled by %+v", second)
	}
	if second.Settings.ReviewsADay != 11 {
		t.Errorf("the second vault's settings are %+v", second.Settings)
	}
}

// handwritten is a vault holding a preset that no deck names yet.
var handwritten = map[string]string{
	"Term.md":        deck["Term.md"],
	"Sanskrit.md":    pointed["Sanskrit.md"],
	"decks/Words.md": deck["decks/Words.md"],
}

// typedManually is that deck as a person left it: pointed at the preset, and
// holding a card they typed and gave no card id.
const typedManually = "---\ntype: deck\nlinks:\n" +
	"  - to: Sanskrit\n    role: ref\n    type: preset\n---\n" +
	"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n" +
	"\n### Meaning\n\nCompost made of fallen leaves alone\n" +
	"\n## Yaffle\n\n[[Term]]\n\n### Word\n\nYaffle\n" +
	"\n### Meaning\n\nA green woodpecker\n"

// Starting a session gives a card id to the card that carries none, and the deck it
// writes is level in the index by the time the session is handed over. Nothing
// else is running over the vault, and nothing rescans it.
func TestAMarkGivenOnSessionDownLevelsTheDeck(t *testing.T) {
	handler, target := newAPI(t, handwritten)
	v := target[0]

	pos := filepath.Join(v.Path, "decks", "Words.md")
	if err := os.WriteFile(pos, []byte(typedManually), 0o644); err != nil {
		t.Fatal(err)
	}

	// The index still holds the deck as it was scanned, so it names no preset.
	if before := getDeckPreset(t, handler, v, "decks/Words.md"); before.Path != "" {
		t.Fatalf("the deck already is under %+v", before)
	}

	startSession(t, handler, v)

	content, err := os.ReadFile(pos)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "## Yaffle ^") {
		t.Fatalf("the card was given no card id: %q", content)
	}

	after := getDeckPreset(t, handler, v, "decks/Words.md")
	if after.Path != "Sanskrit.md" || after.Title != "Sanskrit" {
		t.Errorf("the deck is under %+v", after)
	}
}

// A question about the presets of a vault this installation does not hold is
// rejected, and nothing is read for it.
func TestAQuestionAboutThePresetsOfAVaultNobodyContainsIsRefused(t *testing.T) {
	handler, _ := newAPI(t, pointed)

	_, err := handler.GetVaultDeckPreset(t.Context(),
		&api.GetVaultDeckPresetRequest{Vault: "nobody", Deck: "decks/Words.md"})
	if codeOf(err) != server.CodeNotFound {
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
	handler, target := newAPI(t, pointed)
	handler.scenarios().flash.Presets = flashcards.NewPresets(nil, nil, brokenPresetLinks{}, nil, nil, handler.scenarios().day, nil)

	_, err := handler.GetVaultDeckPreset(t.Context(),
		&api.GetVaultDeckPresetRequest{Vault: string(target[0].ID), Deck: "decks/Words.md"})
	if code := codeOf(err); code != server.CodeInternal {
		t.Errorf("reading answered %v", err)
	}
}

// A preset the deck names that cannot be read comes back rejected as the editor
// refuses a note, and carries no preset.
func TestGetVaultDeckPreset_UnreadablePresetMatchesEditorError(t *testing.T) {
	handler, target := newAPI(t, pointed)
	v := target[0]
	if err := os.Remove(filepath.Join(v.Path, "Sanskrit.md")); err != nil {
		t.Fatal(err)
	}

	out, err := handler.GetVaultDeckPreset(t.Context(),
		&api.GetVaultDeckPresetRequest{Vault: string(v.ID), Deck: "decks/Words.md"})
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Error; got != api.ErrorCodeMissing {
		t.Errorf("a preset that is not there fails as %v", got)
	}
	if out.Preset != nil {
		t.Errorf("a rejected preset came with %+v", out.Preset)
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
	handler, target := newAPI(t, presetted)
	v := target[0]

	out, err := handler.StartSession(t.Context(), &api.StartSessionRequest{
		Vault: string(v.ID), Preset: newPresetName("Steady.md"),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := out.Cards
	if len(request) != 2 {
		t.Fatalf("a preset of two cards a day offered %d", len(request))
	}
	for _, one := range request {
		if one.Deck != "decks/Birds.md" {
			t.Errorf("the session asked %s, which another preset schedules", one.Deck)
		}
	}
}

// Which cards were meant is a question, and the window is asked it again.
func TestNamingADeckAndAPresetTogetherIsRefused(t *testing.T) {
	handler, target := newAPI(t, presetted)
	v := target[0]

	_, err := handler.StartSession(t.Context(), &api.StartSessionRequest{
		Vault: string(v.ID), Deck: "decks/Birds.md", Preset: newPresetName("Steady.md"),
	})
	if codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("a deck and a preset together were answered with %v", err)
	}
}

// Why a preset schedules nothing is the core's verdict, and every preset the
// nothing else in the answer would tell a window that it is stopped.
func TestWatchCardsDue_ReturnsStoppedReasonPerPreset(t *testing.T) {
	// The card ids are the ones the application writes, so the cards stand without
	// the vault being sat down to.
	assigned := func(pos string, cardID string) string {
		return "---\ntype: deck\nlinks:\n  - to: " + pos + "\n    role: ref\n    type: preset\n" +
			"---\n\n## Card ^" + cardID + "\n\n[[Term]]\n\n### Word\n\nw\n\n### Meaning\n\nm\n"
	}
	stoppedPresetFiles := map[string]string{
		"Term.md": deck["Term.md"],
		"Steady.md": "---\ntype: preset\ngoal: retention\n" +
			"new_a_day: 2\nreviews_a_day: 2\n---\n\n# Steady\n",
		"Quiet.md": "---\ntype: preset\ngoal: minutes_a_day\n" +
			"minutes_a_day: 0\n---\n\n# Quiet\n",
		"Lonely.md": "---\ntype: preset\ngoal: retention\n" +
			"new_a_day: 0\nreviews_a_day: 0\n---\n\n# Lonely\n",
		"decks/Birds.md":  assigned("Steady", "3f4g5h6j7k"),
		"decks/Rivers.md": assigned("Quiet", "zpqrstvwxy"),
	}
	handler, _ := newAPI(t, stoppedPresetFiles)

	got := make(map[string]api.StopReason)
	for _, one := range front(t, handler).Vaults[0].Presets {
		got[one.Preset] = one.StopsOn
	}
	want := map[string]api.StopReason{
		"Steady.md": api.StopReasonNothing,
		"Quiet.md":  api.StopReasonNoMinutes,
		"Lonely.md": api.StopReasonNoCards,
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
	handler, target := newAPI(t, map[string]string{
		"Term.md": deck["Term.md"],
		"Quiet.md": "---\ntype: preset\ngoal: minutes_a_day\n" +
			"minutes_a_day: 0\n---\n\n# Quiet\n",
		"decks/Birds.md": newWordDeck("Quiet", 2, 0),
	})

	out, err := handler.GetVaultDeckPreset(t.Context(),
		&api.GetVaultDeckPresetRequest{
			Vault: string(target[0].ID), Deck: "decks/Birds.md",
		})
	if err != nil {
		t.Fatal(err)
	}
	one := out.Preset
	if got := one.Stops; got != api.StopReasonNoMinutes {
		t.Errorf("the preset schedules nothing for %v", got)
	}
	if got := one.StopsOn; got != api.StopReasonNoMinutes {
		t.Errorf("the day schedules nothing for %v", got)
	}
}

// window shows in place of an empty session.
func TestAPresetThatSchedulesNothingIsRefusedWithItsReason(t *testing.T) {
	handler, target := newAPI(t, presetted)
	v := target[0]

	_, err := handler.StartSession(t.Context(), &api.StartSessionRequest{
		Vault: string(v.ID), Preset: newPresetName("Lonely.md"),
	})
	if codeOf(err) != server.CodeFailedPrecondition {
		t.Fatalf("a preset nothing points at was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "no deck") {
		t.Errorf("the reason shown is %q", err)
	}
}
