package identity_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

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

func TestSealToAccountBox_StaysUnderTheSealLimitForAnEpochKey(t *testing.T) {
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
