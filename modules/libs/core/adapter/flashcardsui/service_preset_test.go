package flashcardsui

import (
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// pointing is a deck of as many cards naming this preset. A deck naming none is
// written with an empty name.
func pointing(at string, cards int, from int) string {
	out := "---\ntype: deck\n"
	if at != "" {
		out += "links:\n  - to: " + at + "\n    role: ref\n    type: preset\n"
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
	"decks/Birds.md":  pointing("Steady", 4, 0),
	"decks/Rivers.md": pointing("Other", 4, 100),
}

// naming is the preset a request names, as the window sends it.
func naming(preset string) *string { return &preset }

// The window draws a tile per preset, and pressing one opens a sitting over the
// cards of every deck pointing at it, held to that preset's budget.
func TestASittingIsOpenedOverOnePreset(t *testing.T) {
	api, held := windowed(t, presetted)
	v := held[0]

	out, err := api.StartSession(t.Context(), connect.NewRequest(&v1.StartSessionRequest{
		Vault: string(v.ID), Preset: naming("Steady.md"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	asked := out.Msg.GetAsked()
	if len(asked) != 2 {
		t.Fatalf("a preset of two cards a day offered %d", len(asked))
	}
	for _, one := range asked {
		if one.GetDeck() != "decks/Birds.md" {
			t.Errorf("the sitting asked %s, which another preset schedules", one.GetDeck())
		}
	}
}

// Which cards were meant is a question, and the window is asked it again.
func TestNamingADeckAndAPresetTogetherIsRefused(t *testing.T) {
	api, held := windowed(t, presetted)
	v := held[0]

	_, err := api.StartSession(t.Context(), connect.NewRequest(&v1.StartSessionRequest{
		Vault: string(v.ID), Deck: "decks/Birds.md", Preset: naming("Steady.md"),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("a deck and a preset together were answered with %v", err)
	}
}

// Why a preset schedules nothing is the core's verdict, and every preset the
// front door counts carries it. A preset no deck points at carries it too:
// nothing else in the answer would tell a window that it is stopped.
func TestEachPresetOnTheFrontDoorSaysWhyItSchedulesNothing(t *testing.T) {
	// The marks are the ones the application writes, so the cards stand without
	// the vault being sat down to.
	marked := func(at string, mark string) string {
		return "---\ntype: deck\nlinks:\n  - to: " + at + "\n    role: ref\n    type: preset\n" +
			"---\n\n## Card ^" + mark + "\n\n[[Term]]\n\n### Word\n\nw\n\n### Meaning\n\nm\n"
	}
	stopped := map[string]string{
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
	api, _ := windowed(t, stopped)

	got := make(map[string]v1.Stopped)
	for _, one := range front(t, api).GetVaults()[0].GetPresets() {
		got[one.GetPreset()] = one.GetStopsOn()
	}
	want := map[string]v1.Stopped{
		"Steady.md": v1.Stopped_STOPPED_NOTHING,
		"Quiet.md":  v1.Stopped_STOPPED_NO_MINUTES,
		"Lonely.md": v1.Stopped_STOPPED_NO_CARDS,
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
	api, held := windowed(t, map[string]string{
		"Term.md": deck["Term.md"],
		"Quiet.md": "---\ntype: preset\ngoal: minutes_a_day\n" +
			"minutes_a_day: 0\n---\n\n# Quiet\n",
		"decks/Birds.md": pointing("Quiet", 2, 0),
	})

	out, err := api.GetVaultDeckPreset(t.Context(), connect.NewRequest(
		&v1.GetVaultDeckPresetRequest{
			Vault: string(held[0].ID), Deck: "decks/Birds.md",
		}))
	if err != nil {
		t.Fatal(err)
	}
	one := out.Msg.GetPreset()
	if got := one.GetStops(); got != v1.Stopped_STOPPED_NO_MINUTES {
		t.Errorf("the preset schedules nothing for %v", got)
	}
	if got := one.GetStopsOn(); got != v1.Stopped_STOPPED_NO_MINUTES {
		t.Errorf("the day schedules nothing for %v", got)
	}
}

// A preset with nothing to ask is refused with the reason, which is what the
// window shows in place of an empty sitting.
func TestAPresetThatSchedulesNothingIsRefusedWithItsReason(t *testing.T) {
	api, held := windowed(t, presetted)
	v := held[0]

	_, err := api.StartSession(t.Context(), connect.NewRequest(&v1.StartSessionRequest{
		Vault: string(v.ID), Preset: naming("Lonely.md"),
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("a preset nothing points at was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "no deck") {
		t.Errorf("the reason shown is %q", err)
	}
}
