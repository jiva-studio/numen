package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestASlotEntryCarriesItsCondition(t *testing.T) {
	parsed, err := readSlots(`[{"slot": "fileTree.context", "command": "ledger.open", "when": "target.isFolder && !target.isMultiple"}]`)
	if err != nil {
		t.Fatal(err)
	}

	if when := parsed.Contributes.Slots[0].When; when != "target.isFolder && !target.isMultiple" {
		t.Errorf("when = %q", when)
	}
}

// The conditions themselves are tested against libs/core/protocol/when.json in package when; this is the
// error the reader gives for an entry whose condition fails the check.
func TestAConditionWithAnUnknownKeyOrAMalformedExpressionIsAnInvalidSlotEntry(t *testing.T) {
	for _, when := range []string{"target.isRemote", "target.isFolder &&"} {
		_, err := readSlots(`[{"slot":"fileTree.context","command":"ledger.open","when":"` + when + `"}]`)

		if !errors.Is(err, manifest.ErrInvalidSlotEntry) || !strings.Contains(err.Error(), "when") {
			t.Errorf("%q: error = %v, want an invalid slot entry naming when", when, err)
		}
	}
}
