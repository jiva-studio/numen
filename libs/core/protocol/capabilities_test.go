package protocol_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/protocol"
)

func TestTheIndexChangedEventCarriesTheVaultAndIsReadBack(t *testing.T) {
	event := protocol.GetIndexChangedEvent("01VAULT")
	if event != "onIndexChanged:01VAULT" {
		t.Errorf("the event is %q", event)
	}
	if vaultID, ok := protocol.ReadIndexChangedEvent(event); !ok || vaultID != "01VAULT" {
		t.Errorf("read %q, %v", vaultID, ok)
	}
}

func TestAnEventThatNamesNoVaultIsNotTheIndexChangedEvent(t *testing.T) {
	for _, event := range []string{"", "*", "onFile:md", "onIndexChanged", "onIndexChanged:", "indexChanged:01VAULT"} {
		if vaultID, ok := protocol.ReadIndexChangedEvent(event); ok {
			t.Errorf("%q was read as the event of the vault %q", event, vaultID)
		}
	}
}
