package account_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/shared/identity"
)

const boxContext = "numen-sync/share-seal/v1"

func sealToIdentityOf(t *testing.T, client *account.Client, plugin string, plain []byte) []byte {
	t.Helper()
	public, err := client.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chain, err := identity.ParseChain(public.Chain)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := identity.SealToAccountBox(rand.Reader, chain.AccountBoxKey(), plugin, []byte(boxContext), plain)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func TestOpenSealedBox_ReturnsThePlainOfABoxSealedToTheAccountOfTheHost(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)
	sealed := sealToIdentityOf(t, client, pluginID, []byte("epoch key"))

	opened, err := client.OpenSealedBox(t.Context(), []byte(boxContext), sealed)

	if err != nil || !bytes.Equal(opened, []byte("epoch key")) {
		t.Errorf("opened = %q, error = %v, want the plain", opened, err)
	}
}

func TestOpenSealedBox_ReturnsErrCannotOpenForAnotherPluginContextAndChangedBytes(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)
	sealed := sealToIdentityOf(t, client, pluginID, []byte("epoch key"))
	other := sealToIdentityOf(t, client, "md.numen.tools.link", []byte("epoch key"))
	changed := bytes.Clone(sealed)
	changed[0] ^= 1

	for name, call := range map[string]func() ([]byte, error){
		"another plugin":  func() ([]byte, error) { return client.OpenSealedBox(t.Context(), []byte(boxContext), other) },
		"another context": func() ([]byte, error) { return client.OpenSealedBox(t.Context(), []byte("other"), sealed) },
		"changed bytes":   func() ([]byte, error) { return client.OpenSealedBox(t.Context(), []byte(boxContext), changed) },
	} {
		if opened, err := call(); !errors.Is(err, account.ErrCannotOpen) || opened != nil {
			t.Errorf("%s: opened = %q, error = %v, want nil and ErrCannotOpen", name, opened, err)
		}
	}
}

func TestOpenSealedBox_ReturnsErrNoIdentityWhenDeviceHasNoIdentity(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	fake.RemoveIdentity()
	client := newClient(t, fake)

	if _, err := client.OpenSealedBox(t.Context(), []byte(boxContext), []byte("sealed")); !errors.Is(err, account.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
}

func TestOpenSealedBox_RefusesABoxOfMoreThan1KiBWithoutCallingTheHost(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)

	if _, err := client.OpenSealedBox(t.Context(), []byte(boxContext), make([]byte, 1<<10+1)); err == nil {
		t.Error("a box of 1 KiB and a byte was sent to the host")
	}
}
