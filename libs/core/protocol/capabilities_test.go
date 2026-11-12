package protocol_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/protocol"
)

func TestTheVectorsReadyEventCarriesTheVaultAndIsReadBack(t *testing.T) {
	event := protocol.GetVectorsReadyEvent("01VAULT")
	if event != "onVectorsReady:01VAULT" {
		t.Errorf("the event is %q", event)
	}
	if vaultID, ok := protocol.ReadVectorsReadyEvent(event); !ok || vaultID != "01VAULT" {
		t.Errorf("read %q, %v", vaultID, ok)
	}
}

func TestAnEventThatNamesNoVaultIsNotTheVectorsReadyEvent(t *testing.T) {
	for _, event := range []string{"", "*", "onFile:md", "onVectorsReady", "onVectorsReady:", "vectorsReady:01VAULT"} {
		if vaultID, ok := protocol.ReadVectorsReadyEvent(event); ok {
			t.Errorf("%q was read as the event of the vault %q", event, vaultID)
		}
	}
}
