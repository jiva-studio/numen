package identity

import (
	"bytes"
	"crypto/rand"
	"errors"
	"slices"
	"testing"
	"time"
)

type forgery struct {
	signer   DeviceKey
	devices  [3]DeviceKey
	recovery RecoveryKey
	keys     AccountKeys
	before   Chain
	chain    Chain
	revoked  DevicePublicKey
}

func newForgery(tb testing.TB) forgery {
	tb.Helper()
	recovery, err := newRecoveryKey([32]byte{1}, [32]byte{2})
	if err != nil {
		tb.Fatal(err)
	}
	keys, err := NewAccountKeys(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	chain, err := NewChain(recovery, keys, time.Unix(1_700_000_000, 0))
	if err != nil {
		tb.Fatal(err)
	}
	var devices [3]DeviceKey
	for i := range devices {
		if devices[i], err = NewDeviceKey(rand.Reader); err != nil {
			tb.Fatal(err)
		}
		if chain, err = chain.AppendDevice(recovery, devices[i].Public(), "Device", time.Unix(1_700_000_000, 0)); err != nil {
			tb.Fatal(err)
		}
	}
	before := chain
	revoked, _, err := chain.RevokeDevice(devices[0], keys, devices[2].Public().Sign, Lost, rand.Reader, time.Unix(1_700_000_100, 0))
	if err != nil {
		tb.Fatalf("RevokeDevice() = %v", err)
	}
	return forgery{signer: devices[0], devices: devices, recovery: recovery, keys: keys, before: before, chain: revoked, revoked: devices[2].Public()}
}

func (f forgery) resign(change func(*Entry)) Chain {
	entries := slices.Clone(f.chain.entries)
	last := entries[len(entries)-1]
	revoke := *last.Revoke
	last.Revoke = &revoke
	change(&last)
	sign(&last, f.signer)
	entries[len(entries)-1] = last
	return Chain{entries: entries}
}

func TestVerify_RefusesARevokeThatDoesNotRaiseTheEpochByOne(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	for _, epoch := range []uint32{0, 2, 7} {
		chain := f.resign(func(e *Entry) { e.Revoke.Epoch = epoch })
		if err := chain.Verify(); !errors.Is(err, ErrBrokenLink) {
			t.Errorf("epoch %d: error = %v, want ErrBrokenLink", epoch, err)
		}
	}
}

func TestVerify_RefusesARevokeWhoseWrappedSetIsNotTheActiveDevices(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	cases := map[string]func(*Entry){
		"a device missing": func(e *Entry) { e.Revoke.Wrapped = e.Revoke.Wrapped[:1] },
		"a device twice":   func(e *Entry) { e.Revoke.Wrapped = []DeviceWrap{e.Revoke.Wrapped[0], e.Revoke.Wrapped[0]} },
		"the revoked device included": func(e *Entry) {
			e.Revoke.Wrapped = append(slices.Clone(e.Revoke.Wrapped), DeviceWrap{Device: f.revoked.Sign, Wrapped: e.Revoke.Wrapped[0].Wrapped})
		},
		"another order": func(e *Entry) { slices.Reverse(e.Revoke.Wrapped) },
	}
	for name, change := range cases {
		if err := f.resign(change).Verify(); !errors.Is(err, ErrWrappedSet) {
			t.Errorf("%s: error = %v, want ErrWrappedSet", name, err)
		}
	}
}

func (f forgery) resignLast(by Signer, change func(*Entry)) Chain {
	entries := slices.Clone(f.chain.entries)
	last := entries[len(entries)-1]
	revoke := *last.Revoke
	last.Revoke = &revoke
	change(&last)
	last.Signer = by.SignerKey()
	sign(&last, by)
	entries[len(entries)-1] = last
	return Chain{entries: entries}
}

func (f forgery) rotated(tb testing.TB) forgery {
	tb.Helper()
	chain, _, err := f.before.RotateKeys(f.recovery, f.keys, rand.Reader, time.Unix(1_700_000_200, 0))
	if err != nil {
		tb.Fatalf("RotateKeys() = %v", err)
	}
	f.chain = chain
	return f
}

func (f forgery) recordOf(tb testing.TB, device DeviceKey) Record {
	tb.Helper()
	wrapped, err := f.keys.WrapForDevice(device.Public())
	if err != nil {
		tb.Fatal(err)
	}
	return Record{Device: device, Wrapped: wrapped}
}

func TestVerify_RefusesARevokeWithAnInconsistentReasonOrDate(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	at := time.Unix(1_700_000_100, 0)
	cases := map[string]func(*Entry){
		"stolen with no date":    func(e *Entry) { e.Revoke.Reason = ReasonStolen },
		"stolen after the entry": func(e *Entry) { e.Revoke.Reason, e.Revoke.Since = ReasonStolen, at.Add(time.Second) },
		"lost with a date":       func(e *Entry) { e.Revoke.Since = at.Add(-time.Hour) },
		"no reason":              func(e *Entry) { e.Revoke.Reason = 0 },
	}
	for name, change := range cases {
		if err := f.resign(change).Verify(); !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: error = %v, want ErrMalformed", name, err)
		}
	}
}

func TestVerify_AcceptsAStolenRevokeWithADateBeforeTheEntry(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	since := time.Unix(1_700_000_050, 0).UTC()
	chain := f.resign(func(e *Entry) { e.Revoke.Reason, e.Revoke.Since = ReasonStolen, since })
	if err := chain.Verify(); err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	parsed, err := ParseChain(chain.Bytes())
	if err != nil {
		t.Fatalf("ParseChain() = %v", err)
	}
	removal := parsed.ListRevokedDevices()[0].Removal
	if removal.Reason != ReasonStolen || !removal.Since.Equal(since) || !bytes.Equal(removal.By, f.devices[0].Public().Sign) {
		t.Errorf("removal = %+v, want stolen since %v by the laptop", removal, since)
	}
}

func TestVerify_RefusesARotationNotSignedByTheRecoveryKey(t *testing.T) {
	t.Parallel()
	f := newForgery(t).rotated(t)
	chain := f.resignLast(f.devices[0], func(*Entry) {})
	if err := chain.Verify(); !errors.Is(err, ErrUnknownSigner) {
		t.Errorf("error = %v, want ErrUnknownSigner", err)
	}
}

func TestVerify_RefusesARotationThatNamesADeviceOrDoesNotWrapForEveryActiveDevice(t *testing.T) {
	t.Parallel()
	f := newForgery(t).rotated(t)
	cases := map[string]struct {
		change func(*Entry)
		want   error
	}{
		"a device named":   {func(e *Entry) { e.Revoke.Device = f.devices[2].Public().Sign }, ErrMalformed},
		"a reason given":   {func(e *Entry) { e.Revoke.Reason = ReasonLost }, ErrMalformed},
		"a device missing": {func(e *Entry) { e.Revoke.Wrapped = e.Revoke.Wrapped[:1] }, ErrWrappedSet},
		"a device twice": {func(e *Entry) {
			e.Revoke.Wrapped = []DeviceWrap{e.Revoke.Wrapped[0], e.Revoke.Wrapped[0], e.Revoke.Wrapped[2]}
		}, ErrWrappedSet},
	}
	for name, c := range cases {
		if err := f.resignLast(f.recovery, c.change).Verify(); !errors.Is(err, c.want) {
			t.Errorf("%s: error = %v, want %v", name, err, c.want)
		}
	}
}

func TestAdopt_RefusesAWrapOfOtherKeysThanTheAccountBoxOfTheChain(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	other, err := NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := other.WrapForDevice(f.devices[1].Public())
	if err != nil {
		t.Fatal(err)
	}
	forged := f.resign(func(e *Entry) {
		wraps := slices.Clone(e.Revoke.Wrapped)
		wraps[1].Wrapped = foreign
		e.Revoke.Wrapped = wraps
	})
	state, err := NewState(f.recordOf(t, f.devices[1]), f.before)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.Adopt(forged); !errors.Is(err, ErrWrapMismatch) {
		t.Errorf("error = %v, want ErrWrapMismatch", err)
	}
	if _, err := state.Adopt(f.chain); err != nil {
		t.Errorf("the honest chain: error = %v, want none", err)
	}
}

func TestAdopt_RefusesAKeyHistoryThatDoesNotOpenWithTheWrappedKeys(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	forged := f.resign(func(e *Entry) { e.Revoke.PreviousKeys = bytes.Repeat([]byte{7}, previousKeysSize) })
	state, err := NewState(f.recordOf(t, f.devices[1]), f.before)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.Adopt(forged); !errors.Is(err, ErrWrapMismatch) {
		t.Errorf("error = %v, want ErrWrapMismatch", err)
	}
}
