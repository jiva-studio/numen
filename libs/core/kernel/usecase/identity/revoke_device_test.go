package identity_test

import (
	"encoding/binary"
	"errors"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func (l linked) fingerprintOf(tb testing.TB, d *linkDevice) string {
	tb.Helper()
	shown, err := d.get().Execute(tb.Context())
	if err != nil {
		tb.Fatal(err)
	}
	return shown.DeviceFingerprint
}

func TestListDevices_ListsActiveDevicesFirstAndFlagsThisDevice(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	devices, err := l.joiner.devices().Execute(t.Context())
	if err != nil || len(devices) != 2 {
		t.Fatalf("ListDevices() = %+v, %v, want 2 devices", devices, err)
	}
	for _, device := range devices {
		isThis := device.Fingerprint == l.fingerprintOf(t, l.joiner)
		if device.IsThis != isThis || device.IsRevoked || !fingerprint.MatchString(device.Fingerprint) {
			t.Errorf("device = %+v, want IsThis = %v and not revoked", device, isThis)
		}
	}
}

func TestRevokeDevice_TheNewKeysReplaceTheRecordOnlyAfterTheServerAcceptedTheChain(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	before := l.host.readRecord(t).Wrapped.Bytes()
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.joiner), identitydomain.RevokeLost, time.Time{}); err != nil {
		t.Fatalf("RevokeDevice() = %v", err)
	}
	pending, err := l.host.publish().Execute(t.Context())
	if err != nil || !pending.IsPending {
		t.Fatalf("GetChainPublish() = %+v, %v, want a pending chain", pending, err)
	}
	chain, err := identitylib.ParseChain(pending.Chain)
	if err != nil || chain.Epoch() != 1 || len(chain.ListRevokedDevices()) != 1 {
		t.Fatalf("pending chain: epoch = %d, revoked = %d, error = %v", chain.Epoch(), len(chain.ListRevokedDevices()), err)
	}
	if string(l.host.readRecord(t).Wrapped.Bytes()) != string(before) || l.host.readChain(t).Epoch() != 0 {
		t.Error("the host kept the new keys or the new chain before the server accepted the chain")
	}
	if err := l.host.answered().Execute(t.Context(), true, nil); err != nil {
		t.Fatal(err)
	}
	if string(l.host.readRecord(t).Wrapped.Bytes()) == string(before) || l.host.readChain(t).Epoch() != 1 {
		t.Error("the host did not keep the new keys and the new chain after the accept")
	}
}

func TestRevokeDevice_RemovesDeviceFromListAndNextSealUsesNewEpoch(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.joiner), identitydomain.RevokeLost, time.Time{}); err != nil {
		t.Fatal(err)
	}
	l.relay.settle(t, l.host, l.joiner)

	devices, err := l.host.devices().Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var active, revoked int
	for _, device := range devices {
		if device.IsRevoked {
			revoked++
		} else {
			active++
		}
	}
	if active != 1 || revoked != 1 {
		t.Errorf("devices = %+v, want 1 active and 1 revoked", devices)
	}
	sealed, err := l.host.seal().Execute(t.Context(), "md.numen.tools.sync", []byte("c"), []byte("milk"))
	if err != nil || binary.BigEndian.Uint32(sealed[1:5]) != 1 {
		t.Errorf("SealData() = %x, %v, want a seal of epoch 1", sealed, err)
	}
	shown, err := l.host.get().Execute(t.Context())
	if err != nil || shown.KeyEpoch != 1 || shown.IsRevoked {
		t.Errorf("identity = %+v, %v, want epoch 1 and not revoked", shown, err)
	}
}

func TestRevokeDevice_RefusesThisDeviceAndAnUnknownDevice(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.host), identitydomain.RevokeLost, time.Time{}); !errors.Is(err, identitylib.ErrSelfRevoke) {
		t.Errorf("this device: error = %v, want ErrSelfRevoke", err)
	}
	if err := l.host.revoke().Execute(t.Context(), "0000 0000 0000 0000 0000 0000 0000 0000", identitydomain.RevokeLost, time.Time{}); !errors.Is(err, identitylib.ErrUnknownDevice) {
		t.Errorf("an unknown device: error = %v, want ErrUnknownDevice", err)
	}
	if pending, err := l.host.publish().Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("a refused revoke left %+v pending: %v", pending, err)
	}
}

func TestRevokeDevice_RefusesToLeaveTheAccountWithoutADevice(t *testing.T) {
	t.Parallel()
	device := newLinkDevice("Laptop", newClock())
	device.createAccount(t)
	shown, err := device.get().Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := device.revoke().Execute(t.Context(), shown.DeviceFingerprint, identitydomain.RevokeLost, time.Time{}); !errors.Is(err, identitydomain.ErrLastDevice) {
		t.Errorf("error = %v, want ErrLastDevice", err)
	}
}

func TestChainPublished_RevokeAbortsWhenRevokedDeviceAddedDeviceMeanwhile(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	l.relay.staleOnce = func(head identitylib.Chain) identitylib.Chain {
		next, err := head.AppendDevice(l.joiner.readRecord(t).Device, newDeviceKey(t).Public(), "Stolen", getNow().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.joiner), identitydomain.RevokeLost, time.Time{}); err != nil {
		t.Fatal(err)
	}
	l.relay.settle(t, l.host, l.joiner)

	if len(l.relay.published) != 1 || !errors.Is(l.relay.published[0], identitydomain.ErrRevokeRaced) {
		t.Fatalf("answers to the courier = %v, want one ErrRevokeRaced", l.relay.published)
	}
	devices, err := l.host.devices().Execute(t.Context())
	if err != nil || len(devices) != 3 {
		t.Errorf("devices = %+v, %v, want the list of the new head with 3 devices", devices, err)
	}
	for _, device := range devices {
		if device.IsRevoked {
			t.Errorf("%+v is revoked: the revoke was aborted", device)
		}
	}
	if pending, err := l.host.publish().Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("an aborted revoke is still pending: %+v, %v", pending, err)
	}
}

func TestChainPublished_RevokeRebuildsOnNewHeadWhenAnotherDeviceAppended(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	l.relay.staleOnce = func(head identitylib.Chain) identitylib.Chain {
		next, err := head.AppendDevice(l.host.readRecord(t).Device, newDeviceKey(t).Public(), "Tablet", getNow().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.joiner), identitydomain.RevokeLost, time.Time{}); err != nil {
		t.Fatal(err)
	}
	l.relay.settle(t, l.host, l.joiner)
	if l.relay.head.Epoch() != 1 || len(l.relay.head.ListDevices()) != 2 || len(l.relay.head.ListRevokedDevices()) != 1 {
		t.Errorf("server head: epoch %d, %d active, %d revoked, want epoch 1, the host and the tablet active, the phone revoked",
			l.relay.head.Epoch(), len(l.relay.head.ListDevices()), len(l.relay.head.ListRevokedDevices()))
	}
}

func TestAdoptChain_ARevokedDeviceLearnsItWasRevokedAndKeepsItsRecord(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.joiner), identitydomain.RevokeLost, time.Time{}); err != nil {
		t.Fatal(err)
	}
	l.relay.settle(t, l.host, l.joiner)
	before := l.joiner.readRecord(t).Bytes()

	if err := l.joiner.adopt().Execute(t.Context(), l.relay.head.Bytes()); err != nil {
		t.Fatalf("AdoptChain() = %v", err)
	}
	shown, err := l.joiner.get().Execute(t.Context())
	if err != nil || !shown.IsRevoked {
		t.Errorf("identity = %+v, %v, want revoked", shown, err)
	}
	if string(l.joiner.readRecord(t).Bytes()) != string(before) {
		t.Error("the revoked device lost or changed its record")
	}
	if err := l.joiner.adopt().Execute(t.Context(), l.relay.head.Bytes()); err != nil {
		t.Errorf("the second AdoptChain() = %v, want it idempotent", err)
	}
	public, err := l.joiner.public().Execute(t.Context())
	if err != nil || !public.IsRevoked || public.ChainLength != l.relay.head.Len() {
		t.Errorf("public identity = %+v, %v, want revoked over the head", public, err)
	}
}

func TestAdoptChain_RemainingDeviceAdoptsNewKeysOfEpoch(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.joiner), identitydomain.RevokeLost, time.Time{}); err != nil {
		t.Fatal(err)
	}
	l.relay.settle(t, l.host, l.joiner)
	other := newLinkDevice("Desktop", l.clock)
	other.createAccount(t)
	if err := l.host.adopt().Execute(t.Context(), l.relay.head.Bytes()); err != nil {
		t.Fatalf("AdoptChain() = %v", err)
	}
	if err := l.host.adopt().Execute(t.Context(), other.readChain(t).Bytes()); !errors.Is(err, identitylib.ErrNotExtension) {
		t.Errorf("a chain of another account: error = %v, want ErrNotExtension", err)
	}
	public, err := l.host.public().Execute(t.Context())
	if err != nil || public.KeyEpoch != 1 || public.ChainLength != l.relay.head.Len() || public.IsRevoked {
		t.Errorf("public identity = %+v, %v, want epoch 1 over the head, not revoked", public, err)
	}
}

func TestAdoptChain_RefusesBytesThatAreNotAChain(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	if err := l.host.adopt().Execute(t.Context(), []byte("not a chain")); !errors.Is(err, identitylib.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestRevokeDevice_RefusesAnUnknownReasonAndAStolenDeviceWithoutAUsableDate(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	fingerprint := l.fingerprintOf(t, l.joiner)
	cases := map[string]struct {
		reason identitydomain.RevokeReason
		since  time.Time
	}{
		"no reason":            {"", time.Time{}},
		"another reason":       {"misplaced", getNow().Add(-time.Hour)},
		"stolen with no date":  {identitydomain.RevokeStolen, time.Time{}},
		"stolen in the future": {identitydomain.RevokeStolen, getNow().Add(time.Hour)},
	}
	for name, c := range cases {
		if err := l.host.revoke().Execute(t.Context(), fingerprint, c.reason, c.since); !errors.Is(err, identitydomain.ErrInvalidRevokeReason) {
			t.Errorf("%s: error = %v, want ErrInvalidRevokeReason", name, err)
		}
	}
	if pending, err := l.host.publish().Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("a refused revoke left %+v pending: %v", pending, err)
	}
}

func TestListDevices_NamesWhoRevokedADeviceWhenAndWhyOnBothDevices(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	since := getNow().Add(-48 * time.Hour)
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.joiner), identitydomain.RevokeStolen, since); err != nil {
		t.Fatal(err)
	}
	l.relay.settle(t, l.host, l.joiner)
	if err := l.joiner.adopt().Execute(t.Context(), l.relay.head.Bytes()); err != nil {
		t.Fatal(err)
	}

	for name, device := range map[string]*linkDevice{"the laptop that revoked": l.host, "the phone that was revoked": l.joiner} {
		devices, err := device.devices().Execute(t.Context())
		if err != nil || len(devices) != 2 {
			t.Fatalf("%s: ListDevices() = %+v, %v, want 2 devices", name, devices, err)
		}
		revocation := devices[1].Revocation
		if !devices[1].IsRevoked || devices[1].Name != "Phone" || revocation.ByName != "Laptop" || revocation.IsByRecovery ||
			revocation.Reason != identitydomain.RevokeStolen || !revocation.Since.Equal(since) || revocation.At.IsZero() {
			t.Errorf("%s: revoked device = %+v, want the phone revoked by the laptop as stolen since %v", name, devices[1], since)
		}
		if devices[0].IsRevoked || devices[0].Revocation != (identitydomain.Revocation{}) {
			t.Errorf("%s: the active device = %+v, want no revocation", name, devices[0])
		}
	}
}

func TestRevokeDevice_ALostDeviceIsListedAsLostWithNoDate(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	if err := l.host.revoke().Execute(t.Context(), l.fingerprintOf(t, l.joiner), identitydomain.RevokeLost, time.Time{}); err != nil {
		t.Fatal(err)
	}
	l.relay.settle(t, l.host, l.joiner)

	devices, err := l.host.devices().Execute(t.Context())
	if err != nil || devices[1].Revocation.Reason != identitydomain.RevokeLost || !devices[1].Revocation.Since.IsZero() {
		t.Errorf("devices = %+v, %v, want the phone lost with no date", devices, err)
	}
}
