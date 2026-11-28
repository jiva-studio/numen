package identity_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const boxContext = "numen-sync/share-seal/v1"

type boxKit struct {
	kit
	open *identity.OpenSealedBox
	box  [32]byte
}

func newBoxKit(tb testing.TB, withIdentity bool) boxKit {
	tb.Helper()
	k := newKit("Laptop")
	deps := identity.NewDeps(k.keys, fakeNames{name: "Laptop"}, testsupport.NewClockAt(getNow())).WithChains(k.chains)
	var box [32]byte
	if withIdentity {
		if _, err := k.create.Execute(tb.Context(), getLegalWords()); err != nil {
			tb.Fatal(err)
		}
		public, err := identity.NewGetPublicIdentity(deps).Execute(tb.Context())
		if err != nil {
			tb.Fatal(err)
		}
		chain, err := identitylib.ParseChain(public.Chain)
		if err != nil {
			tb.Fatal(err)
		}
		box = chain.AccountBoxKey()
	}
	return boxKit{kit: k, open: identity.NewOpenSealedBox(deps), box: box}
}

func (k boxKit) seal(tb testing.TB, pluginID string, plain []byte) []byte {
	tb.Helper()
	sealed, err := identitylib.SealToAccountBox(rand.Reader, k.box, pluginID, []byte(boxContext), plain)
	if err != nil {
		tb.Fatal(err)
	}
	return sealed
}

func TestOpenSealedBox_ReturnsThePlainOfABoxSealedToTheAccountForThePlugin(t *testing.T) {
	t.Parallel()
	k := newBoxKit(t, true)
	sealed := k.seal(t, syncPlugin, []byte("epoch key"))

	opened, err := k.open.Execute(t.Context(), syncPlugin, []byte(boxContext), sealed)

	if err != nil || !bytes.Equal(opened, []byte("epoch key")) {
		t.Errorf("opened = %q, error = %v, want the plain", opened, err)
	}
}

func TestOpenSealedBox_RefusesAnotherPluginAnotherContextAndChangedBytes(t *testing.T) {
	t.Parallel()
	k := newBoxKit(t, true)
	sealed := k.seal(t, syncPlugin, []byte("epoch key"))
	changed := bytes.Clone(sealed)
	changed[len(changed)-1] ^= 1

	for name, call := range map[string]func() ([]byte, error){
		"another plugin": func() ([]byte, error) {
			return k.open.Execute(t.Context(), "md.numen.tools.link", []byte(boxContext), sealed)
		},
		"another context": func() ([]byte, error) { return k.open.Execute(t.Context(), syncPlugin, []byte("other"), sealed) },
		"changed bytes":   func() ([]byte, error) { return k.open.Execute(t.Context(), syncPlugin, []byte(boxContext), changed) },
		"no bytes":        func() ([]byte, error) { return k.open.Execute(t.Context(), syncPlugin, []byte(boxContext), nil) },
	} {
		opened, err := call()
		if !errors.Is(err, identitydomain.ErrCannotOpen) || opened != nil {
			t.Errorf("%s: opened = %q, error = %v, want nil and ErrCannotOpen", name, opened, err)
		}
	}
}

func TestOpenSealedBox_RefusesABoxSealedToAnotherAccount(t *testing.T) {
	t.Parallel()
	k := newBoxKit(t, true)
	other, err := identitylib.NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := identitylib.SealToAccountBox(rand.Reader, other.BoxPublicKey(), syncPlugin, []byte(boxContext), []byte("epoch key"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := k.open.Execute(t.Context(), syncPlugin, []byte(boxContext), sealed); !errors.Is(err, identitydomain.ErrCannotOpen) {
		t.Errorf("error = %v, want ErrCannotOpen", err)
	}
}

func TestOpenSealedBox_ReportsNoIdentityWhenTheKeyStoreHasNoRecord(t *testing.T) {
	t.Parallel()
	k := newBoxKit(t, false)

	if _, err := k.open.Execute(t.Context(), syncPlugin, []byte(boxContext), []byte("sealed")); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
}

func TestOpenSealedBox_ReportsAKeyStoreFailureAndNeverCallsItNoIdentity(t *testing.T) {
	t.Parallel()
	k := newBoxKit(t, true)
	k.keys.getFails = errBroken

	_, err := k.open.Execute(t.Context(), syncPlugin, []byte(boxContext), []byte("sealed"))

	if !errors.Is(err, errBroken) || errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("error = %v, want the key store failure", err)
	}
}
