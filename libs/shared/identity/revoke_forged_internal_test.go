package identity

import (
	"crypto/rand"
	"errors"
	"slices"
	"testing"
	"time"
)

type forgery struct {
	signer  DeviceKey
	chain   Chain
	revoked DevicePublicKey
	other   DeviceKey
	keys    AccountKeys
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
	revoked, _, err := chain.RevokeDevice(devices[0], keys, devices[2].Public().Sign, rand.Reader, time.Unix(1_700_000_100, 0))
	if err != nil {
		tb.Fatalf("RevokeDevice() = %v", err)
	}
	return forgery{signer: devices[0], chain: revoked, revoked: devices[2].Public(), other: devices[1], keys: keys}
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
