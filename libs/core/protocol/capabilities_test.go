package protocol_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/protocol"
)

func TestTheSearchSourceReadyEventCarriesTheVaultAndIsReadBack(t *testing.T) {
	event := protocol.GetSearchSourceReadyEvent("01VAULT")
	if event != "onSearchSourceReady:01VAULT" {
		t.Errorf("the event is %q", event)
	}
	if vaultID, ok := protocol.ReadSearchSourceReadyEvent(event); !ok || vaultID != "01VAULT" {
		t.Errorf("read %q, %v", vaultID, ok)
	}
}

func TestAnEventThatNamesNoVaultIsNotTheSearchSourceReadyEvent(t *testing.T) {
	for _, event := range []string{"", "*", "onFile:md", "onSearchSourceReady", "onSearchSourceReady:", "searchSourceReady:01VAULT"} {
		if vaultID, ok := protocol.ReadSearchSourceReadyEvent(event); ok {
			t.Errorf("%q was read as the event of the vault %q", event, vaultID)
		}
	}
}
