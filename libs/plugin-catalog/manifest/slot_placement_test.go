package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

func TestOneViewMayBePlacedInTwoSlots(t *testing.T) {
	if _, err := readSlots(`[{"slot":"titleBar.status","view":"balance"},{"slot":"statusBar.right","view":"balance"}]`); err != nil {
		t.Errorf("error = %v, want none", err)
	}
}

func TestOneCommandMayBePlacedInTwoBarsAndInAMenu(t *testing.T) {
	_, err := readSlots(`[
		{"slot":"titleBar.commands","command":"ledger.open"},
		{"slot":"statusBar.left","command":"ledger.open"},
		{"slot":"fileTree.context","command":"ledger.open"}
	]`)

	if err != nil {
		t.Errorf("error = %v, want none", err)
	}
}

func TestACommandMayRepeatInMenusAndSubmenus(t *testing.T) {
	_, err := readSlots(`[
		{"slot":"titleBar.commands","command":"ledger.open"},
		{"slot":"fileTree.context","command":"ledger.open"},
		{"slot":"palette.searchResult","command":"ledger.open"},
		{"slot":"tabStrip.context","submenu":"ledger.more","title":"More"},
		{"slot":"ledger.more","command":"ledger.open"}
	]`)

	if err != nil {
		t.Errorf("error = %v, want none", err)
	}
}

func TestAViewAndACommandWithOneIdInOneSlotAreRefusedNamingIdAndSlot(t *testing.T) {
	for _, slot := range []string{"titleBar.commands", "statusBar.left"} {
		_, err := readSlots(`[{"slot":"` + slot + `","command":"ledger.open"},{"slot":"` + slot + `","view":"ledger.open"}]`)

		if !errors.Is(err, manifest.ErrInvalidSlotEntry) {
			t.Fatalf("%s: error = %v, want an invalid slot entry", slot, err)
		}
		for _, part := range []string{`"ledger.open"`, slot} {
			if !strings.Contains(err.Error(), part) {
				t.Errorf("%s: error %q does not name %s", slot, err, part)
			}
		}
	}
}

func TestAViewAndACommandWithOneIdInDifferentSlotsAreAccepted(t *testing.T) {
	_, err := readSlots(`[{"slot":"titleBar.commands","command":"ledger.open"},{"slot":"titleBar.status","view":"ledger.open"}]`)

	if err != nil {
		t.Errorf("error = %v, want none", err)
	}
}

func TestTwoPluginsMayPlaceEntriesWithOneIdInOneSlot(t *testing.T) {
	for _, id := range []string{"org.example.ledger", "org.example.vault"} {
		_, err := manifestjson.Read(strings.NewReader(`{"id":"` + id + `","name":"P","version":"1.0.0","frontend":{},"contributes":{"slots":[{"slot":"titleBar.status","view":"open.it"}]}}`))

		if err != nil {
			t.Errorf("%s: error = %v, want none", id, err)
		}
	}
}
