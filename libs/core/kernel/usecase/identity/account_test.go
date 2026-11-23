package identity_test

import (
	"crypto/ed25519"
	"errors"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const syncPlugin = "md.numen.tools.sync"

type accountKit struct {
	kit
	sign *identity.SignAsDevice
	read *identity.GetPublicIdentity
}

func newAccountKit(t *testing.T, withIdentity bool) accountKit {
	t.Helper()
	k := newKit("Laptop")
	deps := identity.NewDeps(k.keys, fakeNames{name: "Laptop"}, testsupport.NewClockAt(getNow())).WithChains(k.chains)
	if withIdentity {
		if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
			t.Fatal(err)
		}
	}
	return accountKit{kit: k, sign: identity.NewSignAsDevice(deps), read: identity.NewGetPublicIdentity(deps)}
}

func TestSignAsDevice_SignsForThePluginIDWithTheKeyOfThePublicIdentity(t *testing.T) {
	t.Parallel()
	k := newAccountKit(t, true)

	signature, err := k.sign.Execute(t.Context(), syncPlugin, []byte("challenge"))
	if err != nil {
		t.Fatal(err)
	}
	public, err := k.read.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(signature) != ed25519.SignatureSize {
		t.Fatalf("signature has %d bytes, want 64", len(signature))
	}
	if !identitylib.VerifyForPlugin(public.DeviceSignKey, syncPlugin, []byte("challenge"), [64]byte(signature)) {
		t.Error("the signature does not verify for the plugin under the device key")
	}
	if identitylib.VerifyForPlugin(public.DeviceSignKey, "md.numen.tools.link", []byte("challenge"), [64]byte(signature)) {
		t.Error("the signature verifies for another plugin")
	}
}

func TestSignAsDevice_ReportsNoIdentityWhenTheKeyStoreHasNoRecord(t *testing.T) {
	t.Parallel()
	k := newAccountKit(t, false)

	if _, err := k.sign.Execute(t.Context(), syncPlugin, []byte("challenge")); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
}

func TestSignAsDevice_ReportsAKeyStoreFailureAndNeverCallsItNoIdentity(t *testing.T) {
	t.Parallel()
	k := newAccountKit(t, true)
	k.keys.getFails = errBroken

	_, err := k.sign.Execute(t.Context(), syncPlugin, []byte("challenge"))

	if !errors.Is(err, errBroken) || errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("error = %v, want the key store failure and not ErrNoIdentity", err)
	}
}

func TestSignAsDevice_ReportsARecordThatDoesNotParse(t *testing.T) {
	t.Parallel()
	k := newAccountKit(t, false)
	k.keys.keys[keyName] = "not-a-record"

	_, err := k.sign.Execute(t.Context(), syncPlugin, []byte("challenge"))

	if !errors.Is(err, identitylib.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestSignAsDevice_SignsNothingForAnEmptyPluginID(t *testing.T) {
	t.Parallel()
	k := newAccountKit(t, true)

	if _, err := k.sign.Execute(t.Context(), "", []byte("challenge")); err == nil {
		t.Error("a signature was made with no plugin id")
	}
}

func TestGetPublicIdentity_ReportsTheFingerprintsOfTheIdentityOfThisDevice(t *testing.T) {
	t.Parallel()
	k := newAccountKit(t, true)

	public, err := k.read.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	seen, err := k.get.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if public.AccountFingerprint != seen.AccountFingerprint || public.DeviceFingerprint != seen.DeviceFingerprint {
		t.Errorf("fingerprints = %q, %q, want %q, %q", public.AccountFingerprint, public.DeviceFingerprint, seen.AccountFingerprint, seen.DeviceFingerprint)
	}
	if !fingerprint.MatchString(public.AccountFingerprint) {
		t.Errorf("account fingerprint %q is not in the form of a fingerprint", public.AccountFingerprint)
	}
}

func TestGetPublicIdentity_ReportsTheSignedChainThatContainsTheDeviceKey(t *testing.T) {
	t.Parallel()
	k := newAccountKit(t, true)

	public, err := k.read.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chain, err := identitylib.ParseChain(public.Chain)
	if err != nil {
		t.Fatalf("the chain does not parse and verify: %v", err)
	}

	if len(public.DeviceSignKey) != ed25519.PublicKeySize {
		t.Fatalf("device sign key has %d bytes, want 32", len(public.DeviceSignKey))
	}
	devices := chain.ListDevices()
	if len(devices) != 1 || string(devices[0].Public.Sign) != string(public.DeviceSignKey) {
		t.Errorf("devices of the chain = %+v, want the device of the sign key", devices)
	}
	if chain.Root().Fingerprint().String() != public.AccountFingerprint {
		t.Error("the root of the chain is not the account of the fingerprint")
	}
}

func TestGetPublicIdentity_ReportsNoIdentityAKeyStoreFailureAndABrokenRecord(t *testing.T) {
	t.Parallel()
	none := newAccountKit(t, false)
	if _, err := none.read.Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("no record: error = %v, want ErrNoIdentity", err)
	}

	broken := newAccountKit(t, true)
	broken.keys.getFails = errBroken
	if _, err := broken.read.Execute(t.Context()); !errors.Is(err, errBroken) || errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("key store failure: error = %v", err)
	}

	garbled := newAccountKit(t, false)
	garbled.keys.keys[keyName] = "not-a-record"
	if _, err := garbled.read.Execute(t.Context()); !errors.Is(err, identitylib.ErrMalformed) {
		t.Errorf("garbled record: error = %v, want ErrMalformed", err)
	}
}
