package flashcardsui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// elsewhere is a second vault whose deck stands at the path the first's deck
// stands at, and is scheduled by a preset of its own. A claim about the vault a
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

// scheduled is the preset one vault's deck is scheduled by.
func scheduled(t *testing.T, api *API, v domain.Vault, path string) *v1.Preset {
	t.Helper()
	out, err := api.Scheduling(t.Context(), connect.NewRequest(
		&v1.FlashcardsServiceSchedulingRequest{Vault: string(v.ID), Deck: path}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetRefusal() != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("%s was refused: %v", path, out.Msg.GetRefusal())
	}
	return out.Msg.GetPreset()
}

// The preset a deck is scheduled by is the one the vault named in the request
// holds. The window is over every vault at once, and the two decks here stand
// at the same path.
func TestAPresetIsReadFromTheVaultTheRequestNames(t *testing.T) {
	api, held := windowed(t, pointed, elsewhere)
	one, two := held[0], held[1]

	first := scheduled(t, api, one, "decks/Words.md")
	if first.GetPath() != "Sanskrit.md" || first.GetTitle() != "Sanskrit" {
		t.Errorf("the first vault is scheduled by %+v", first)
	}
	if first.GetSettings().GetReviewsADay() != 45 {
		t.Errorf("the first vault's settings are %+v", first.GetSettings())
	}

	second := scheduled(t, api, two, "decks/Words.md")
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

// Sitting down mints a mark for the card that carries none, and the deck it
// writes is level in the index by the time the sitting is handed over. Nothing
// else is running over the vault, and nothing rescans it.
func TestAMarkMintedOnSittingDownLevelsTheDeck(t *testing.T) {
	api, held := windowed(t, handwritten)
	v := held[0]

	at := filepath.Join(v.Path, "decks", "Words.md")
	if err := os.WriteFile(at, []byte(typedByHand), 0o644); err != nil {
		t.Fatal(err)
	}

	// The index still holds the deck as it was scanned, so it names no preset.
	if before := scheduled(t, api, v, "decks/Words.md"); before.GetPath() != "" {
		t.Fatalf("the deck already stands under %+v", before)
	}

	started(t, api, v)

	written, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "## Yaffle ^") {
		t.Fatalf("the card was given no mark: %q", written)
	}

	after := scheduled(t, api, v, "decks/Words.md")
	if after.GetPath() != "Sanskrit.md" || after.GetTitle() != "Sanskrit" {
		t.Errorf("the deck stands under %+v", after)
	}
}

// A question about the presets of a vault this installation does not hold is
// refused, and nothing is read for it.
func TestAQuestionAboutThePresetsOfAVaultNobodyHoldsIsRefused(t *testing.T) {
	api, _ := windowed(t, pointed)

	_, err := api.Scheduling(t.Context(), connect.NewRequest(
		&v1.FlashcardsServiceSchedulingRequest{Vault: "nobody", Deck: "decks/Words.md"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("reading answered %v", err)
	}
}
