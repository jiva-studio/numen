package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func TestOneViewPlacedInTwoBarSlotsIsRefusedNamingBothSlots(t *testing.T) {
	_, err := readSlots(`[{"slot":"titleBar.status","view":"balance"},{"slot":"statusBar.right","view":"balance"}]`)

	if !errors.Is(err, manifest.ErrInvalidSlotEntry) {
		t.Fatalf("error = %v, want an invalid slot entry", err)
	}
	for _, slot := range []string{"titleBar.status", "statusBar.right"} {
		if !strings.Contains(err.Error(), slot) {
			t.Errorf("error %q does not name %s", err, slot)
		}
	}
}

func TestOneCommandPlacedInTwoBarSlotsIsRefusedNamingBothSlots(t *testing.T) {
	_, err := readSlots(`[{"slot":"titleBar.commands","command":"ledger.open"},{"slot":"statusBar.left","command":"ledger.open"}]`)

	if !errors.Is(err, manifest.ErrInvalidSlotEntry) {
		t.Fatalf("error = %v, want an invalid slot entry", err)
	}
	for _, slot := range []string{"titleBar.commands", "statusBar.left"} {
		if !strings.Contains(err.Error(), slot) {
			t.Errorf("error %q does not name %s", err, slot)
		}
	}
}

func TestAViewAndACommandWithOneIdInTheBarSlotsAreRefused(t *testing.T) {
	_, err := readSlots(`[{"slot":"titleBar.commands","command":"ledger.open"},{"slot":"titleBar.status","view":"ledger.open"}]`)

	if !errors.Is(err, manifest.ErrInvalidSlotEntry) {
		t.Fatalf("error = %v, want an invalid slot entry", err)
	}
	if !strings.Contains(err.Error(), `"ledger.open"`) {
		t.Errorf("error %q does not name the shared id", err)
	}
}

func TestACommandInABarSlotMayRepeatInMenusAndSubmenus(t *testing.T) {
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
