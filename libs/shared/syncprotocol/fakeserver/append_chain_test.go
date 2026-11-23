package fakeserver_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

func (a *ownedAccount) extend(tb testing.TB, by identity.Signer, name string) (identity.DeviceKey, identity.Chain) {
	tb.Helper()
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	chain, err := a.chain.AppendDevice(by, device.Public(), name, machineClock{}.Now())
	if err != nil {
		tb.Fatal(err)
	}
	return device, chain
}

func (a *ownedAccount) revoke(tb testing.TB, by identity.Signer, victim identity.DeviceKey) identity.Chain {
	tb.Helper()
	chain, keys, err := a.chain.RevokeDevice(by, a.keys, victim.Public().Sign, identity.Lost, rand.Reader, machineClock{}.Now())
	if err != nil {
		tb.Fatal(err)
	}
	a.keys = keys
	return chain
}

func TestAppendChain_StoresALongerChainOfTheAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	_, longer := e.account.extend(t, e.device, "Phone")

	head, err := e.accounts.AppendChain(t.Context(), longer.Bytes())

	if err != nil || !bytes.Equal(head, longer.Bytes()) {
		t.Fatalf("head = %d bytes, error = %v, want the presented chain", len(head), err)
	}
	stored, err := e.accounts.GetAccountChain(t.Context())
	if err != nil || !bytes.Equal(stored, longer.Bytes()) {
		t.Errorf("GetAccountChain() = %d bytes, error = %v, want the appended chain", len(stored), err)
	}
}

func TestAppendChain_AcceptsTheStoredChainAgainAsIdempotent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	_, longer := e.account.extend(t, e.device, "Phone")
	if _, err := e.accounts.AppendChain(t.Context(), longer.Bytes()); err != nil {
		t.Fatal(err)
	}
	head, err := e.accounts.AppendChain(t.Context(), longer.Bytes())
	if err != nil || !bytes.Equal(head, longer.Bytes()) {
		t.Errorf("the second append: %d bytes, %v, want the same head", len(head), err)
	}
}

func TestAppendChain_TheSecondOfTwoCompetingChainsGetsTheHead(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	first := account.addDevice(t, "Laptop")
	one := newEnvOn(t, address, &account, first)
	_, forOne := account.extend(t, first, "Phone")
	_, forOther := account.extend(t, first, "Tablet")
	if _, err := one.accounts.AppendChain(t.Context(), forOne.Bytes()); err != nil {
		t.Fatal(err)
	}

	_, err := one.accounts.AppendChain(t.Context(), forOther.Bytes())

	stale, ok := errors.AsType[syncprotocol.ChainStaleError](err)
	if !ok || !bytes.Equal(stale.Head, forOne.Bytes()) || connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("error = %v, want ChainStaleError with the head of the first chain from FAILED_PRECONDITION", err)
	}
}

func TestAppendChain_RefusesAChainThatIsAPrefixOfTheStoredOne(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	short := e.account.chain
	_, longer := e.account.extend(t, e.device, "Phone")
	if _, err := e.accounts.AppendChain(t.Context(), longer.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err := e.accounts.AppendChain(t.Context(), short.Bytes()); !errors.As(err, &syncprotocol.ChainStaleError{}) {
		t.Errorf("error = %v, want ChainStaleError", err)
	}
}

func TestAppendChain_RefusesAChainOfAnotherAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	other := newOwnedAccount(t)
	other.addDevice(t, "Laptop")
	if _, err := e.accounts.AppendChain(t.Context(), other.chain.Bytes()); err == nil {
		t.Error("a chain of another root was accepted")
	}
	stored, err := e.accounts.GetAccountChain(t.Context())
	if err != nil || !bytes.Equal(stored, e.account.chain.Bytes()) {
		t.Error("the stored chain changed")
	}
}

func TestAppendChain_RefusesAChainWhereTheDeviceOfTheSessionIsNotActive(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	first := account.addDevice(t, "Laptop")
	second := account.addDevice(t, "Phone")
	secondEnv := newEnvOn(t, address, &account, second)
	revoked := account.revoke(t, first, second)
	if _, err := secondEnv.accounts.AppendChain(t.Context(), revoked.Bytes()); connect.CodeOf(err) != connect.CodePermissionDenied && !errors.Is(err, syncprotocol.ErrWriteNotAllowed) {
		t.Errorf("error = %v, want PERMISSION_DENIED", err)
	}
}

func TestAppendChain_RefusesAnEntryThatAnotherDeviceThanThePublisherSigned(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	first := account.addDevice(t, "Laptop")
	second := account.addDevice(t, "Phone")
	firstEnv := newEnvOn(t, address, &account, first)
	_, signedBySecond := account.extend(t, second, "Tablet")
	if _, err := firstEnv.accounts.AppendChain(t.Context(), signedBySecond.Bytes()); err == nil {
		t.Error("an entry signed by a device other than the publisher was accepted")
	}
}

func TestAppendChain_AcceptsAnEntryTheRootSigned(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	_, byRoot := e.account.extend(t, e.account.root, "Restored")
	if _, err := e.accounts.AppendChain(t.Context(), byRoot.Bytes()); err != nil {
		t.Errorf("error = %v, want the entry of the root accepted", err)
	}
}

func TestAppendChain_TheRevokeOfTheRecoveryKeyReplacesTheEntriesOfADeviceAtAFork(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	laptop := account.addDevice(t, "Laptop")
	phone := account.addDevice(t, "Phone")
	laptopEnv := newEnvOn(t, address, &account, laptop)
	phoneEnv := newEnvOn(t, address, &account, phone)
	_, withThiefDevice := account.extend(t, laptop, "Thief phone")
	if _, err := laptopEnv.accounts.AppendChain(t.Context(), withThiefDevice.Bytes()); err != nil {
		t.Fatal(err)
	}
	rotated, _, err := account.chain.RevokeDevice(account.root, account.keys, laptop.Public().Sign, identity.Lost, rand.Reader, machineClock{}.Now())
	if err != nil {
		t.Fatal(err)
	}

	head, err := phoneEnv.accounts.AppendChain(t.Context(), rotated.Bytes())

	if err != nil || !bytes.Equal(head, rotated.Bytes()) {
		t.Fatalf("head = %d bytes, error = %v, want the chain of the recovery key", len(head), err)
	}
	stored, err := phoneEnv.accounts.GetAccountChain(t.Context())
	if err != nil || !bytes.Equal(stored, rotated.Bytes()) {
		t.Errorf("GetAccountChain() = %d bytes, error = %v, want the chain of the recovery key", len(stored), err)
	}
}

func TestAppendChain_AChainOfADeviceDoesNotReplaceTheEntriesOfAnotherDeviceAtAFork(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	laptop := account.addDevice(t, "Laptop")
	phone := account.addDevice(t, "Phone")
	laptopEnv := newEnvOn(t, address, &account, laptop)
	phoneEnv := newEnvOn(t, address, &account, phone)
	_, withTablet := account.extend(t, laptop, "Tablet")
	if _, err := laptopEnv.accounts.AppendChain(t.Context(), withTablet.Bytes()); err != nil {
		t.Fatal(err)
	}
	_, withDesktop := account.extend(t, phone, "Desktop")
	if _, err := phoneEnv.accounts.AppendChain(t.Context(), withDesktop.Bytes()); !errors.As(err, &syncprotocol.ChainStaleError{}) {
		t.Errorf("error = %v, want ChainStaleError", err)
	}
}
