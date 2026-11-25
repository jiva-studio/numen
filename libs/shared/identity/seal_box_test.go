package identity_test

import (
	"bytes"
	"crypto/ecdh"
	"crypto/hpke"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

const boxContext = "numen-sync/share-seal/v1"

func sealToBox(tb testing.TB, account identity.AccountKeys, pluginID string, context, plain []byte) []byte {
	tb.Helper()
	sealed, err := identity.SealToAccountBox(rand.Reader, account.BoxPublicKey(), pluginID, context, plain)
	if err != nil {
		tb.Fatal(err)
	}
	return sealed
}

func TestOpenSealedBox_ReturnsThePlainSealedToTheBoxKey(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	plain := []byte("epoch 3 key of the folder Food")
	sealed := sealToBox(t, account, sealPluginID, []byte(boxContext), plain)

	got, err := account.OpenSealedBox(sealPluginID, []byte(boxContext), sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Errorf("opened = %q, want %q", got, plain)
	}
}

func TestSealToAccountBox_EpochKeyFitsSealLimit(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	plain := make([]byte, 4+32)
	sealed := sealToBox(t, account, sealPluginID, []byte(boxContext), plain)
	if len(sealed) > 256 {
		t.Errorf("sealed has %d bytes, want at most 256", len(sealed))
	}
}

func TestSealToAccountBox_IsDeterministicForAFixedRandomReader(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	fixed := allByteValues()
	seal := func() []byte {
		sealed, err := identity.SealToAccountBox(bytes.NewReader(fixed), account.BoxPublicKey(), sealPluginID, []byte(boxContext), []byte("x"))
		if err != nil {
			t.Fatal(err)
		}
		return sealed
	}
	if !bytes.Equal(seal(), seal()) {
		t.Error("the same random reader gave different boxes")
	}
}

func TestSealToAccountBox_ReturnsADifferentBoxEachTime(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	first := sealToBox(t, account, sealPluginID, []byte(boxContext), []byte("x"))
	second := sealToBox(t, account, sealPluginID, []byte(boxContext), []byte("x"))
	if bytes.Equal(first, second) {
		t.Error("two boxes of one plain are equal")
	}
}

func TestSealToAccountBox_ReturnsTheErrorOfTheRandomReader(t *testing.T) {
	t.Parallel()
	boom := errors.New("no entropy")
	_, err := identity.SealToAccountBox(failingReader{err: boom}, newAccount(t).BoxPublicKey(), sealPluginID, []byte(boxContext), []byte("x"))
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want the reader error", err)
	}
}

func TestOpenSealedBox_RefusesAnotherPlugin(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	sealed := sealToBox(t, account, sealPluginID, []byte(boxContext), []byte("x"))
	if _, err := account.OpenSealedBox("md.numen.tools.other", []byte(boxContext), sealed); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("error = %v, want ErrUnwrap", err)
	}
}

func TestOpenSealedBox_RefusesAnotherContext(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	sealed := sealToBox(t, account, sealPluginID, []byte(boxContext), []byte("x"))
	if _, err := account.OpenSealedBox(sealPluginID, []byte(boxContext+"2"), sealed); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("error = %v, want ErrUnwrap", err)
	}
}

func TestOpenSealedBox_RefusesAnotherAccount(t *testing.T) {
	t.Parallel()
	sealed := sealToBox(t, newAccount(t), sealPluginID, []byte(boxContext), []byte("x"))
	if _, err := newAccount(t).OpenSealedBox(sealPluginID, []byte(boxContext), sealed); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("error = %v, want ErrUnwrap", err)
	}
}

func TestOpenSealedBox_RefusesAnyChangedByte(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	sealed := sealToBox(t, account, sealPluginID, []byte(boxContext), []byte("x"))
	for i := range sealed {
		changed := append([]byte(nil), sealed...)
		changed[i] ^= 0x01
		if _, err := account.OpenSealedBox(sealPluginID, []byte(boxContext), changed); !errors.Is(err, identity.ErrUnwrap) {
			t.Fatalf("byte %d changed: error = %v, want ErrUnwrap", i, err)
		}
	}
}

func TestOpenSealedBox_RefusesShortInput(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	for _, size := range []int{0, 1, 31, 32, 47} {
		if _, err := account.OpenSealedBox(sealPluginID, []byte(boxContext), make([]byte, size)); !errors.Is(err, identity.ErrUnwrap) {
			t.Errorf("%d bytes: error = %v, want ErrUnwrap", size, err)
		}
	}
}

func TestOpenSealedBox_OpensABoxOfTheStandardHPKESender(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	box := account.BoxPublicKey()
	public, err := ecdh.X25519().NewPublicKey(box[:])
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := hpke.NewDHKEMPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	info := append([]byte("numen/v1/plugin-box/"+sealPluginID+"\x00"), []byte(boxContext)...)
	sealed, err := hpke.Seal(recipient, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), info, []byte("epoch key"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := account.OpenSealedBox(sealPluginID, []byte(boxContext), sealed)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "epoch key" {
		t.Errorf("opened = %q, want the plain of the standard sender", got)
	}
}

func TestSealToAccountBox_StartsWithTheEphemeralPublicKeyOfTheRandomReader(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	fixed := allByteValues()
	ephemeral, err := ecdh.X25519().NewPrivateKey(fixed[:32])
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := identity.SealToAccountBox(bytes.NewReader(fixed), account.BoxPublicKey(), sealPluginID, []byte(boxContext), []byte("x"))
	if err != nil {
		t.Fatal(err)
	}

	if want := ephemeral.PublicKey().Bytes(); !bytes.HasPrefix(sealed, want) || len(sealed) != 32+1+16 {
		t.Errorf("sealed = %x, want the ephemeral key %x and 17 bytes of ciphertext", sealed, want)
	}
}

func TestSealToAccountBox_RefusesAShortRandomReader(t *testing.T) {
	t.Parallel()
	_, err := identity.SealToAccountBox(bytes.NewReader(make([]byte, 31)), newAccount(t).BoxPublicKey(), sealPluginID, []byte(boxContext), []byte("x"))
	if err == nil {
		t.Error("a reader of 31 bytes sealed a box")
	}
}

func TestOpenSealedBox_OpensABoxSealedToAnEarlierEpoch(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	sealedAtEpoch0 := sealToBox(t, m.account.keys, sealPluginID, []byte(boxContext), []byte("epoch 0 share key"))
	revoked, current, err := revokeForSeal(m)
	if err != nil {
		t.Fatal(err)
	}
	sealedAtEpoch1 := sealToBox(t, current, sealPluginID, []byte(boxContext), []byte("epoch 1 share key"))
	wrapped, ok := revoked.GetWrappedFor(m.first.Public().Sign)
	if !ok {
		t.Fatal("the remaining device has no keys of the new epoch")
	}
	unwrapped, err := m.first.UnwrapAccountKeys(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := revoked.OpenKeyHistory(unwrapped)
	if err != nil {
		t.Fatal(err)
	}

	for name, keys := range map[string]identity.AccountKeys{"the device that revoked": current, "a device that reloaded the chain": reloaded} {
		for want, sealed := range map[string][]byte{"epoch 0 share key": sealedAtEpoch0, "epoch 1 share key": sealedAtEpoch1} {
			got, err := keys.OpenSealedBox(sealPluginID, []byte(boxContext), sealed)
			if err != nil || string(got) != want {
				t.Errorf("%s opened %q with error %v, want %q", name, got, err, want)
			}
		}
	}
}

func TestOpenSealedBox_RefusesABoxOfAnUnrelatedAccountAfterARevoke(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	_, current, err := revokeForSeal(m)
	if err != nil {
		t.Fatal(err)
	}
	foreign := sealToBox(t, newAccount(t), sealPluginID, []byte(boxContext), []byte("x"))

	if _, err := current.OpenSealedBox(sealPluginID, []byte(boxContext), foreign); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("error = %v, want ErrUnwrap", err)
	}
}

func revokeForSeal(m team) (identity.Chain, identity.AccountKeys, error) {
	return m.chain.RevokeDevice(m.first, m.firstKeys, m.second.Public().Sign, rand.Reader, epoch().Add(time.Hour))
}
