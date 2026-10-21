package service

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The list a person sees is drawn from the first message, and counting a vault
// reads every deck in it. So the vaults stand there by name and by where they
// are, with nothing counted and nothing said about what they hold.
func TestTheVaultsStandBeforeAnyOfThemIsCounted(t *testing.T) {
	api, target := newAPI(t, deck, other)

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
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
	if len(first.GetVaults()) != len(target) {
		t.Fatalf("the front door opens on %d of %d vaults", len(first.GetVaults()), len(target))
	}
	for at, one := range first.GetVaults() {
		if one.GetId() != string(target[at].ID) || one.GetName() == "" || one.GetPath() == "" {
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
	api, target := newAPI(t, deck, other)

	// The vaults are read when the window opens them, and what a count evaluates to
	// is what is under test here.
	front(t, api)

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("the front door said nothing: %v", stream.Err())
	}

	counts := make(map[string]int, len(target))
	for stream.Receive() {
		response := stream.Msg()
		if len(response.GetVaults()) != 0 {
			t.Errorf("the vaults were listed again: %+v", response.GetVaults())
		}
		one := response.GetCounted()
		if one.GetFaces() != 1 || one.GetNew() != 1 {
			t.Errorf("%s comes to %+v", one.GetName(), one)
		}
		counts[one.GetId()]++
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	for _, v := range target {
		if counts[string(v.ID)] != 1 {
			t.Errorf("the vault %s was counted %d times", string(v.ID), counts[string(v.ID)])
		}
	}
}
