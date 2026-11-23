package fakeserver_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

func TestAppendChain_ARevokeEndsTheSessionsOfTheRevokedDeviceAtOnce(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	first := account.addDevice(t, "Laptop")
	second := account.addDevice(t, "Phone")
	laptop := newEnvOn(t, address, &account, first)
	phone := newEnvOn(t, address, &account, second)
	if _, err := phone.accounts.GetAccountChain(t.Context()); err != nil {
		t.Fatalf("the phone before the revoke: %v", err)
	}
	revoked := account.revoke(t, first, second)

	if _, err := laptop.accounts.AppendChain(t.Context(), revoked.Bytes()); err != nil {
		t.Fatal(err)
	}

	if _, err := phone.accounts.GetAccountChain(t.Context()); !errors.Is(err, syncprotocol.ErrUnauthenticated) {
		t.Errorf("the revoked phone: error = %v, want ErrUnauthenticated", err)
	}
	if _, err := laptop.accounts.GetAccountChain(t.Context()); err != nil {
		t.Errorf("the remaining laptop: %v", err)
	}
}

func TestSignIn_RevokedDeviceReceivesRevocationWithStoredChain(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	first := account.addDevice(t, "Laptop")
	second := account.addDevice(t, "Phone")
	old := account.chain
	laptop := newEnvOn(t, address, &account, first)
	revoked := account.revoke(t, first, second)
	if _, err := laptop.accounts.AppendChain(t.Context(), revoked.Bytes()); err != nil {
		t.Fatal(err)
	}

	_, err := trySignIn(t, address, second, old)

	told, ok := errors.AsType[syncprotocol.DeviceRevokedError](err)
	if !ok || !bytes.Equal(told.Chain, revoked.Bytes()) {
		t.Errorf("error = %v, want DeviceRevokedError with the stored chain", err)
	}
}

func TestSignIn_ADeviceThatWasNeverInTheChainGetsNoDetail(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	account.addDevice(t, "Laptop")
	stranger, _ := account.extend(t, account.root, "Stranger")
	_, err := trySignIn(t, address, stranger, account.chain)
	var revoked syncprotocol.DeviceRevokedError
	if errors.As(err, &revoked) {
		t.Error("a stranger received the stored chain")
	}
	if err == nil {
		t.Error("a stranger signed in")
	}
}

func TestAppendCatalogEntry_ARevokedAuthorIsRefusedAndARemainingDeviceIsNot(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	first := account.addDevice(t, "Laptop")
	second := account.addDevice(t, "Phone")
	laptop := newEnvOn(t, address, &account, first)
	revoked := account.revoke(t, first, second)
	if _, err := laptop.accounts.AppendChain(t.Context(), revoked.Bytes()); err != nil {
		t.Fatal(err)
	}

	byRevoked := laptop.newCatalogEntry(newID(16), true, "vault")
	byRevoked.AuthorDeviceSignKey = second.Public().Sign
	signature := second.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetCatalogEntryMessage(byRevoked))
	byRevoked.Signature = signature[:]
	if _, err := laptop.catalog.AppendCatalogEntry(t.Context(), byRevoked); !errors.Is(err, syncprotocol.ErrWriteNotAllowed) {
		t.Errorf("entry of the revoked author: error = %v, want ErrWriteNotAllowed", err)
	}
	laptop.appendEntry(newID(16), true, "vault")
}
