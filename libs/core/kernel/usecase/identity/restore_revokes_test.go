package identity_test

import (
	"bytes"
	"testing"
	"time"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func recoveryKeyOf(tb testing.TB) identitylib.RecoveryKey {
	tb.Helper()
	words, err := identitylib.ParseRecoveryWords(getLegalWords())
	if err != nil {
		tb.Fatal(err)
	}
	key, err := words.DeriveRecoveryKey()
	if err != nil {
		tb.Fatal(err)
	}
	return key
}

func TestCompleteRestore_RevokesADeviceThatTheChainStillListsAsActive(t *testing.T) {
	t.Parallel()
	desktop := newRestoreKit(t, "Desktop")
	if _, err := desktop.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	thief, err := identitylib.NewDeviceKey(bytes.NewReader(bytes.Repeat([]byte{9}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	stale := mustAppend(t, desktop.getChain(t), thief.Public())
	phone := newRestoreKit(t, "Phone")
	if err := phone.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	pending, err := phone.pending.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if err := phone.complete.Execute(t.Context(), syncPlugin, stale); err != nil {
		t.Fatal(err)
	}

	chain, err := identitylib.ParseChain(phone.getChain(t))
	if err != nil {
		t.Fatal(err)
	}
	active := chain.ListDevices()
	if len(active) != 1 || !bytes.Equal(active[0].Public.Sign, pending.DeviceSignKey) {
		t.Errorf("active devices = %d, want only the restored phone", len(active))
	}
	if chain.IsActiveDevice(thief.Public().Sign) {
		t.Error("the device that was never revoked is still active after the restore")
	}
	if _, ok := chain.GetWrappedFor(thief.Public().Sign); ok {
		t.Error("the device that was never revoked holds the keys of the new epoch")
	}
	if len(chain.ListRevokedDevices()) != 2 {
		t.Errorf("revoked devices = %d, want the desktop and the thief", len(chain.ListRevokedDevices()))
	}
}

func mustAppend(tb testing.TB, chainBytes []byte, device identitylib.DevicePublicKey) []byte {
	tb.Helper()
	chain, err := identitylib.ParseChain(chainBytes)
	if err != nil {
		tb.Fatal(err)
	}
	next, err := chain.AppendDevice(recoveryKeyOf(tb), device, "Unknown", getNow().Add(time.Minute))
	if err != nil {
		tb.Fatal(err)
	}
	return next.Bytes()
}
