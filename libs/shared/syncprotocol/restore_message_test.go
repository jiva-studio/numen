package syncprotocol_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func newEpochKey(tb testing.TB, seedByte byte) ed25519.PrivateKey {
	tb.Helper()
	return ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seedByte}, ed25519.SeedSize))
}

func newSignedEpoch(key ed25519.PrivateKey) *syncv1.RestoreEpoch {
	id := bytes.Repeat([]byte{0x21}, 16)
	const issuedAt = 1_800_000_000
	return &syncv1.RestoreEpoch{
		Id: id, IssuedAt: issuedAt, Signature: ed25519.Sign(key, syncprotocol.GetRestoreEpochMessage(id, issuedAt)),
	}
}

func TestGetRestoreEpochMessage_IsTheLabelAZeroByteTheIDAndTheTimeBigEndian(t *testing.T) {
	t.Parallel()
	id := bytes.Repeat([]byte{0x21}, 16)
	want := append([]byte("numen-sync/restore-epoch/v1"), 0)
	want = append(want, id...)
	want = binary.BigEndian.AppendUint64(want, 1_800_000_000)

	got := syncprotocol.GetRestoreEpochMessage(id, 1_800_000_000)

	if !bytes.Equal(got, want) {
		t.Errorf("message = %x, want %x", got, want)
	}
}

func TestGetRestoreEpochMessage_ChangesWhenTheTimeChanges(t *testing.T) {
	t.Parallel()
	id := bytes.Repeat([]byte{0x21}, 16)

	one := syncprotocol.GetRestoreEpochMessage(id, 1)
	other := syncprotocol.GetRestoreEpochMessage(id, 2)

	if bytes.Equal(one, other) {
		t.Error("two issue times give the same message")
	}
}

func TestCheckRestoreEpoch_AcceptsAnEpochSignedByTheKey(t *testing.T) {
	t.Parallel()
	key := newEpochKey(t, 1)

	err := syncprotocol.CheckRestoreEpoch(key.Public().(ed25519.PublicKey), newSignedEpoch(key))

	if err != nil {
		t.Errorf("CheckRestoreEpoch = %v, want nil", err)
	}
}

func TestCheckRestoreEpoch_RefusesWhatDoesNotVerify(t *testing.T) {
	t.Parallel()
	key := newEpochKey(t, 1)
	public, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatal("not an Ed25519 key")
	}
	tamperedID := newSignedEpoch(key)
	tamperedID.Id[0] ^= 1
	tamperedTime := newSignedEpoch(key)
	tamperedTime.IssuedAt++
	tamperedSignature := newSignedEpoch(key)
	tamperedSignature.Signature[0] ^= 1
	shortID := newSignedEpoch(key)
	shortID.Id = shortID.Id[:15]
	shortSignature := newSignedEpoch(key)
	shortSignature.Signature = shortSignature.Signature[:63]
	otherKey, ok := newEpochKey(t, 2).Public().(ed25519.PublicKey)
	if !ok {
		t.Fatal("not an Ed25519 key")
	}
	tests := []struct {
		name  string
		key   []byte
		epoch *syncv1.RestoreEpoch
	}{
		{"changed id", public, tamperedID},
		{"changed time", public, tamperedTime},
		{"changed signature", public, tamperedSignature},
		{"id of 15 bytes", public, shortID},
		{"signature of 63 bytes", public, shortSignature},
		{"another key", otherKey, newSignedEpoch(key)},
		{"key of 31 bytes", public[:31], newSignedEpoch(key)},
		{"no epoch", public, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := syncprotocol.CheckRestoreEpoch(tc.key, tc.epoch); !errors.Is(err, syncprotocol.ErrInvalidRestoreEpoch) {
				t.Errorf("CheckRestoreEpoch = %v, want ErrInvalidRestoreEpoch", err)
			}
		})
	}
}
