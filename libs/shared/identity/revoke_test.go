package identity_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

const revokeContext = "numen-sync/catalog/v1"

// team is an account whose chain lists two devices, both holding the keys of epoch 0.
type team struct {
	account
	first, second identity.DeviceKey
	firstKeys     identity.AccountKeys
}

func newTeam(tb testing.TB) team {
	tb.Helper()
	a := newChainOf(tb)
	first, second := newDeviceOf(tb), newDeviceOf(tb)
	chain := a.chain
	var err error
	if chain, err = appended(chain, a.recovery, first.Public(), "Laptop", epoch()); err != nil {
		tb.Fatal(err)
	}
	if chain, err = appended(chain, first, second.Public(), "Phone", epoch().Add(time.Minute)); err != nil {
		tb.Fatal(err)
	}
	a.chain = chain
	wrapped, err := a.keys.WrapForDevice(first.Public())
	if err != nil {
		tb.Fatal(err)
	}
	firstKeys, err := first.UnwrapAccountKeys(wrapped)
	if err != nil {
		tb.Fatal(err)
	}
	return team{account: a, first: first, second: second, firstKeys: firstKeys}
}

func newChainOf(tb testing.TB) account {
	tb.Helper()
	recovery, err := parseWordsOf(tb).DeriveRecoveryKey()
	if err != nil {
		tb.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	chain, err := identity.NewChain(recovery, keys, epoch())
	if err != nil {
		tb.Fatal(err)
	}
	return account{recovery: recovery, keys: keys, chain: chain}
}

func parseWordsOf(tb testing.TB) identity.RecoveryWords {
	tb.Helper()
	words, err := identity.ParseRecoveryWords(strings.Fields(bip39Vectors()[0].words))
	if err != nil {
		tb.Fatal(err)
	}
	return words
}

func newDeviceOf(tb testing.TB) identity.DeviceKey {
	tb.Helper()
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	return device
}

func (m team) revokeSecond(tb testing.TB) (identity.Chain, identity.AccountKeys) {
	tb.Helper()
	chain, keys, err := m.chain.RevokeDevice(m.first, m.firstKeys, m.second.Public().Sign, rand.Reader, epoch().Add(time.Hour))
	if err != nil {
		tb.Fatalf("RevokeDevice() = %v", err)
	}
	return chain, keys
}

func TestRevokeDevice_RemovesTheDeviceFromTheActiveListAndKeepsItInTheRevokedList(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	if err := chain.Verify(); err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	active, revoked := chain.ListDevices(), chain.ListRevokedDevices()
	if len(active) != 1 || active[0].Name != "Laptop" {
		t.Errorf("ListDevices() = %+v, want the laptop only", active)
	}
	if len(revoked) != 1 || revoked[0].Name != "Phone" {
		t.Errorf("ListRevokedDevices() = %+v, want the phone", revoked)
	}
	if chain.IsActiveDevice(m.second.Public().Sign) || !chain.IsKnownDevice(m.second.Public().Sign) {
		t.Error("a revoked device is active or no longer known")
	}
	if !chain.IsActiveDevice(m.first.Public().Sign) {
		t.Error("the signer is not active")
	}
}

func TestRevokeDevice_DoesNotChangeTheChainItIsCalledOn(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	before := m.chain.Bytes()
	m.revokeSecond(t)
	if !bytes.Equal(m.chain.Bytes(), before) || m.chain.Epoch() != 0 {
		t.Error("RevokeDevice changed the receiver")
	}
}

func TestRevokeDevice_MakesNewKeysForTheNextEpoch(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, keys := m.revokeSecond(t)
	if chain.Epoch() != 1 || keys.Epoch() != 1 {
		t.Errorf("chain epoch = %d, keys epoch = %d, want 1 and 1", chain.Epoch(), keys.Epoch())
	}
	if keys.BoxPublicKey() == m.keys.BoxPublicKey() || chain.AccountBoxKey() != keys.BoxPublicKey() {
		t.Error("the account box key did not change, or the chain does not hold the new one")
	}
}

func TestRevokeDevice_WrapsTheNewKeysForTheRemainingDeviceAndTheRecoveryKeyOnly(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, keys := m.revokeSecond(t)

	wrapped, ok := chain.GetWrappedFor(m.first.Public().Sign)
	if !ok {
		t.Fatal("the remaining device has no wrapped keys")
	}
	opened, err := m.first.UnwrapAccountKeys(wrapped)
	if err != nil || opened.BoxPublicKey() != keys.BoxPublicKey() {
		t.Errorf("the remaining device cannot open the new keys: %v", err)
	}
	if _, ok := chain.GetWrappedFor(m.second.Public().Sign); ok {
		t.Error("the revoked device has wrapped keys of the new epoch")
	}
	recovered, err := m.recovery.UnwrapAccountKeys(chain.WrappedForRecovery())
	if err != nil || recovered.BoxPublicKey() != keys.BoxPublicKey() {
		t.Errorf("the recovery key does not open the latest keys: %v", err)
	}
}

func TestRevokeDevice_RoundTripsThroughBytes(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	parsed, err := identity.ParseChain(chain.Bytes())
	if err != nil {
		t.Fatalf("ParseChain() = %v", err)
	}
	if !bytes.Equal(parsed.Bytes(), chain.Bytes()) || parsed.Epoch() != 1 || len(parsed.ListRevokedDevices()) != 1 {
		t.Error("the parsed chain differs from the revoked chain")
	}
}

func TestRevokeDevice_RefusesToRevokeTheSigner(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	if _, _, err := m.chain.RevokeDevice(m.first, m.firstKeys, m.first.Public().Sign, rand.Reader, epoch()); !errors.Is(err, identity.ErrSelfRevoke) {
		t.Errorf("error = %v, want ErrSelfRevoke", err)
	}
}

func TestRevokeDevice_RefusesADeviceThatIsNotActive(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, keys := m.revokeSecond(t)
	if _, _, err := chain.RevokeDevice(m.first, keys, m.second.Public().Sign, rand.Reader, epoch()); !errors.Is(err, identity.ErrUnknownDevice) {
		t.Errorf("a revoked device: error = %v, want ErrUnknownDevice", err)
	}
	stranger := newDeviceOf(t)
	if _, _, err := m.chain.RevokeDevice(m.first, m.firstKeys, stranger.Public().Sign, rand.Reader, epoch()); !errors.Is(err, identity.ErrUnknownDevice) {
		t.Errorf("a stranger: error = %v, want ErrUnknownDevice", err)
	}
}

func TestRevokeDevice_AcceptsTheRecoveryKeyAsSigner(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _, err := m.chain.RevokeDevice(m.recovery, m.keys, m.second.Public().Sign, rand.Reader, epoch())
	if err != nil || len(chain.ListDevices()) != 1 {
		t.Errorf("revoke by the recovery key: %v", err)
	}
}

func TestRevokeDevice_RefusesToSignWithoutAnActiveSigner(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, keys := m.revokeSecond(t)
	if _, _, err := chain.RevokeDevice(m.second, keys, m.first.Public().Sign, rand.Reader, epoch()); !errors.Is(err, identity.ErrUnknownSigner) {
		t.Errorf("error = %v, want ErrUnknownSigner", err)
	}
}

func TestAppendDevice_RefusesAnEntryOfARevokedDevice(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	if _, err := chain.AppendDevice(m.second, newDeviceOf(t).Public(), "Tablet", epoch()); !errors.Is(err, identity.ErrUnknownSigner) {
		t.Errorf("error = %v, want ErrUnknownSigner", err)
	}
}

func TestAppendDevice_RefusesTheKeyOfARevokedDeviceAgain(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	if _, err := chain.AppendDevice(m.first, m.second.Public(), "Phone again", epoch()); !errors.Is(err, identity.ErrDuplicateDevice) {
		t.Errorf("error = %v, want ErrDuplicateDevice", err)
	}
}

func TestAppendDevice_RefusesAThirtyThirdActiveDevice(t *testing.T) {
	t.Parallel()
	a := newChainOf(t)
	chain := a.chain
	var err error
	for i := range 32 {
		if chain, err = appended(chain, a.recovery, newDeviceOf(t).Public(), "Device", epoch().Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("device %d: %v", i+1, err)
		}
	}
	if _, err = appended(chain, a.recovery, newDeviceOf(t).Public(), "One too many", epoch()); !errors.Is(err, identity.ErrTooManyDevices) {
		t.Errorf("error = %v, want ErrTooManyDevices", err)
	}
}

func TestChain_CountsEpochsAndOpensTheKeyHistory(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	oldSealed, err := m.firstKeys.SealForPlugin(rand.Reader, sealPluginID, []byte(revokeContext), []byte("eggs"))
	if err != nil {
		t.Fatal(err)
	}
	chain, keys := m.revokeSecond(t)
	third := newDeviceOf(t)
	if chain, err = appended(chain, m.first, third.Public(), "Tablet", epoch().Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	chain, _, err = chain.RevokeDevice(m.first, keys, third.Public().Sign, rand.Reader, epoch().Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if chain.Epoch() != 2 {
		t.Fatalf("Epoch() = %d, want 2", chain.Epoch())
	}
	own, ok := chain.GetWrappedFor(m.first.Public().Sign)
	if !ok {
		t.Fatal("no wrapped keys for the remaining device")
	}
	latest, err := m.first.UnwrapAccountKeys(own)
	if err != nil {
		t.Fatal(err)
	}
	history, err := chain.OpenKeyHistory(latest)
	if err != nil {
		t.Fatalf("OpenKeyHistory() = %v", err)
	}
	if history.Epoch() != 2 {
		t.Errorf("Epoch() of the keys = %d, want 2", history.Epoch())
	}
	opened, err := history.OpenForPlugin(sealPluginID, []byte(revokeContext), oldSealed)
	if err != nil || string(opened) != "eggs" {
		t.Errorf("a seal of epoch 0 opened = %q, error = %v, want eggs", opened, err)
	}
	if _, err := latest.OpenForPlugin(sealPluginID, []byte(revokeContext), oldSealed); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("keys without the history opened epoch 0: %v", err)
	}
}

func TestOpenKeyHistory_LetsADeviceLinkedAfterTheRevokeReadEveryEpoch(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	oldSealed, err := m.firstKeys.SealForPlugin(rand.Reader, sealPluginID, []byte(revokeContext), []byte("eggs"))
	if err != nil {
		t.Fatal(err)
	}
	chain, keys := m.revokeSecond(t)
	joiner := newDeviceOf(t)
	if chain, err = appended(chain, m.first, joiner.Public(), "Tablet", epoch().Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	wrapped, err := keys.WrapForDevice(joiner.Public())
	if err != nil {
		t.Fatal(err)
	}
	joinerKeys, err := joiner.UnwrapAccountKeys(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	history, err := chain.OpenKeyHistory(joinerKeys)
	if err != nil {
		t.Fatalf("OpenKeyHistory() = %v", err)
	}
	opened, err := history.OpenForPlugin(sealPluginID, []byte(revokeContext), oldSealed)
	if err != nil || string(opened) != "eggs" {
		t.Errorf("the new device read %q, error = %v, want eggs", opened, err)
	}
}

func TestOpenKeyHistory_RefusesKeysThatAreNotOfTheLatestEpoch(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	if _, err := chain.OpenKeyHistory(m.firstKeys); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("error = %v, want ErrUnwrap", err)
	}
}

func TestRecoveryKey_OpensTheHistoryOfTheLatestChain(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	oldSealed, err := m.firstKeys.SealForPlugin(rand.Reader, sealPluginID, []byte(revokeContext), []byte("eggs"))
	if err != nil {
		t.Fatal(err)
	}
	chain, _ := m.revokeSecond(t)
	latest, err := m.recovery.UnwrapAccountKeys(chain.WrappedForRecovery())
	if err != nil {
		t.Fatal(err)
	}
	history, err := chain.OpenKeyHistory(latest)
	if err != nil {
		t.Fatal(err)
	}
	if opened, err := history.OpenForPlugin(sealPluginID, []byte(revokeContext), oldSealed); err != nil || string(opened) != "eggs" {
		t.Errorf("the words did not open epoch 0: %q, %v", opened, err)
	}
}

func TestRevokedSince_NamesTheDevicesRevokedAfterAPrefix(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	revoked := chain.RevokedSince(m.chain)
	if len(revoked) != 1 || !bytes.Equal(revoked[0], m.second.Public().Sign) {
		t.Errorf("RevokedSince() = %x, want the key of the phone", revoked)
	}
	if got := chain.RevokedSince(chain); len(got) != 0 {
		t.Errorf("RevokedSince(itself) = %x, want none", got)
	}
}

func TestChain_StaysAtEpochZeroWithoutARevoke(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	if m.chain.Epoch() != 0 || len(m.chain.ListRevokedDevices()) != 0 || m.firstKeys.Epoch() != 0 {
		t.Error("a chain without a revoke is not at epoch 0")
	}
}
