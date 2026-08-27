package webui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The window asks whether a node hangs the headings of its note, and turns the
// setting from the palette.

// TestTurningTheHangingIsAnsweredByTheNextQuestion. The palette turns it, the
// file is written, and the window reads what was written. Nothing is launched
// again in between.
func TestTurningTheHangingIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	said, err := f.client.Hanging(t.Context(), connect.NewRequest(&v1.HangingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !said.Msg.GetHangPartsUnderANode() {
		t.Fatal("an installation nobody has configured hangs nothing")
	}

	turned, err := f.client.ChooseHanging(t.Context(), connect.NewRequest(&v1.ChooseHangingRequest{
		HangPartsUnderANode: false,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := turned.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the setting was refused: %v", refusal)
	}

	said, err = f.client.Hanging(t.Context(), connect.NewRequest(&v1.HangingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetHangPartsUnderANode() {
		t.Error("the setting was turned and the window still hangs the parts")
	}
}

// How many parts stand under a node is turned the same way, and a request
// naming no count leaves the one the settings hold where it was.
func TestTurningTheCountOfPartsIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	said, err := f.client.Hanging(t.Context(), connect.NewRequest(&v1.HangingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetPartsUnderANode() != 6 {
		t.Fatalf("an installation nobody has configured stands %d", said.Msg.GetPartsUnderANode())
	}

	three := int32(3)
	if _, err := f.client.ChooseHanging(t.Context(), connect.NewRequest(&v1.ChooseHangingRequest{
		HangPartsUnderANode: true,
		PartsUnderANode:     &three,
	})); err != nil {
		t.Fatal(err)
	}

	if _, err := f.client.ChooseHanging(t.Context(), connect.NewRequest(&v1.ChooseHangingRequest{
		HangPartsUnderANode: false,
	})); err != nil {
		t.Fatal(err)
	}

	said, err = f.client.Hanging(t.Context(), connect.NewRequest(&v1.HangingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetPartsUnderANode() != 3 {
		t.Errorf("a node stands %d parts", said.Msg.GetPartsUnderANode())
	}
	if said.Msg.GetHangPartsUnderANode() {
		t.Error("the setting was turned and the window still hangs the parts")
	}
}

// A count the setting does not take is not written, and the settings are left
// as they are.
func TestACountOfPartsOutsideWhatItGoesToIsNotWritten(t *testing.T) {
	f := opening(t, nil, nil, true)

	twenty := int32(20)
	if _, err := f.client.ChooseHanging(t.Context(), connect.NewRequest(&v1.ChooseHangingRequest{
		HangPartsUnderANode: true,
		PartsUnderANode:     &twenty,
	})); err == nil {
		t.Fatal("a count of twenty was written")
	}

	said, err := f.client.Hanging(t.Context(), connect.NewRequest(&v1.HangingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetPartsUnderANode() != 6 {
		t.Errorf("a node stands %d parts", said.Msg.GetPartsUnderANode())
	}
}

// The setting a person turns is written where they will read it, and every
// other byte of the file is left as they typed it.
func TestTurningTheHangingLeavesTheRestOfTheFileAlone(t *testing.T) {
	f := opening(t, nil, nil, true)
	path := filepath.Join(filepath.Dir(f.settings), "numen.json")
	if err := os.WriteFile(path, []byte("{\n  \"appearance\": {\"text_scale\": 1.5}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := f.client.ChooseHanging(t.Context(), connect.NewRequest(&v1.ChooseHangingRequest{
		HangPartsUnderANode: false,
	})); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"text_scale": 1.5`) {
		t.Errorf("what the person typed was rewritten:\n%s", raw)
	}
	if !strings.Contains(string(raw), `"hang_parts_under_a_node": false`) {
		t.Errorf("the setting is not in the file:\n%s", raw)
	}
}
