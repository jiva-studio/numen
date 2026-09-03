package flashcardsui

import (
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The list a person sees is drawn from the first message, and counting a vault
// reads every deck in it. So the vaults stand there by name and by where they
// are, with nothing counted and nothing said about what they hold.
func TestTheVaultsStandBeforeAnyOfThemIsCounted(t *testing.T) {
	api, held := windowed(t, deck, other)

	stream, err := serving(t, api).Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("the front door said nothing: %v", stream.Err())
	}

	first := stream.Msg()
	if first.GetDay() == "" {
		t.Error("the vaults arrived in no day")
	}
	if first.GetCounted() != nil {
		t.Errorf("a count stands beside the vaults: %+v", first.GetCounted())
	}
	if len(first.GetVaults()) != len(held) {
		t.Fatalf("the front door opens on %d of %d vaults", len(first.GetVaults()), len(held))
	}
	for at, one := range first.GetVaults() {
		if one.GetVaultId() != held[at].ID || one.GetName() == "" || one.GetPath() == "" {
			t.Errorf("the vault stands as %+v", one)
		}
		if one.GetFaces() != 0 || one.GetDue() != 0 || one.GetNew() != 0 ||
			len(one.GetDecks()) != 0 || len(one.GetPresets()) != 0 || one.GetUnread() != "" {
			t.Errorf("%s arrived counted: %+v", one.GetName(), one)
		}
	}
}

// Each vault's count is a message of its own, so a vault of fifty thousand
// cards holds up nothing but itself.
func TestEachVaultsCountArrivesOnItsOwn(t *testing.T) {
	api, held := windowed(t, deck, other)

	// The vaults are read when the window opens them, and what a count comes to
	// is what is under test here.
	front(t, api)

	stream, err := serving(t, api).Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("the front door said nothing: %v", stream.Err())
	}

	counts := make(map[string]int, len(held))
	for stream.Receive() {
		said := stream.Msg()
		if len(said.GetVaults()) != 0 {
			t.Errorf("the vaults were listed again: %+v", said.GetVaults())
		}
		one := said.GetCounted()
		if one.GetFaces() != 1 || one.GetNew() != 1 {
			t.Errorf("%s comes to %+v", one.GetName(), one)
		}
		counts[one.GetVaultId()]++
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	for _, v := range held {
		if counts[v.ID] != 1 {
			t.Errorf("the vault %s was counted %d times", v.ID, counts[v.ID])
		}
	}
}

// The vault a window was last opened on is counted first: a person coming back
// to this window is most often coming back to where they were.
func TestTheVaultOpenedLastIsCountedFirst(t *testing.T) {
	all := []domain.Vault{{ID: "one"}, {ID: "two"}, {ID: "three"}}

	for name, c := range map[string]struct {
		last string
		want []string
	}{
		"one in the middle":    {"two", []string{"two", "one", "three"}},
		"the last of them":     {"three", []string{"three", "one", "two"}},
		"the first of them":    {"one", []string{"one", "two", "three"}},
		"none opened yet":      {"", []string{"one", "two", "three"}},
		"one the list dropped": {"four", []string{"one", "two", "three"}},
	} {
		t.Run(name, func(t *testing.T) {
			api := &API{Registry: registry{held: all, last: c.last}}

			got := make([]string, 0, len(all))
			for _, v := range api.wanted(all) {
				got = append(got, v.ID)
			}
			if len(got) != len(c.want) {
				t.Fatalf("counted %v", got)
			}
			for at, want := range c.want {
				if got[at] != want {
					t.Fatalf("counted %v, want %v", got, c.want)
				}
			}
		})
	}
}
