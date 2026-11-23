package account_test

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/identity"
)

const pluginID = "md.numen.tools.sync"

func newClient(t *testing.T, fake *accounttest.Fake) *account.Client {
	t.Helper()
	endpoint, err := hostcall.ReadVars(fake.BuildVars("token-of-the-plugin"))
	if err != nil {
		t.Fatal(err)
	}
	return account.NewClient(endpoint)
}

func TestGetPublicIdentity_ReturnsWhatIsPublicOfTheIdentity(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)

	got, err := client.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := fake.Public()
	if got.AccountFingerprint != want.AccountFingerprint || got.DeviceFingerprint != want.DeviceFingerprint ||
		!bytes.Equal(got.DeviceSignKey, want.DeviceSignKey) || !bytes.Equal(got.Chain, want.Chain) {
		t.Errorf("identity = %+v, want %+v", got, want)
	}
	if len(got.DeviceSignKey) != ed25519.PublicKeySize || got.AccountFingerprint == "" {
		t.Errorf("identity = %+v, want a 32-byte sign key and an account fingerprint", got)
	}
	chain, err := identity.ParseChain(got.Chain)
	if err != nil {
		t.Fatalf("the chain does not parse: %v", err)
	}
	if devices := chain.ListDevices(); len(devices) != 1 || !bytes.Equal(devices[0].Public.Sign, got.DeviceSignKey) {
		t.Errorf("devices of the chain = %+v", devices)
	}
}

func TestSignAsDevice_ReturnsASignatureThatVerifiesForThePluginID(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)
	public, err := client.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	signature, err := client.SignAsDevice(t.Context(), []byte("challenge"))
	if err != nil {
		t.Fatal(err)
	}

	if len(signature) != 64 || !identity.VerifyForPlugin(public.DeviceSignKey, pluginID, []byte("challenge"), [64]byte(signature)) {
		t.Error("the signature does not verify for the plugin under the device key")
	}
	if identity.VerifyForPlugin(public.DeviceSignKey, "md.numen.tools.link", []byte("challenge"), [64]byte(signature)) {
		t.Error("the signature verifies for another plugin")
	}
	if fake.SignCount() != 1 {
		t.Errorf("signatures made = %d, want 1", fake.SignCount())
	}
}

func TestAHostWithoutAnIdentityAnswersErrNoIdentityToBothCalls(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	fake.NoIdentity()
	client := newClient(t, fake)

	_, signErr := client.SignAsDevice(t.Context(), []byte("challenge"))
	_, readErr := client.GetPublicIdentity(t.Context())

	if !errors.Is(signErr, account.ErrNoIdentity) || !errors.Is(readErr, account.ErrNoIdentity) {
		t.Errorf("sign = %v, read = %v, want ErrNoIdentity", signErr, readErr)
	}
}

func TestAnIdentityCreatedAfterwardsIsFoundByTheNextCall(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	fake.NoIdentity()
	client := newClient(t, fake)
	if _, err := client.GetPublicIdentity(t.Context()); !errors.Is(err, account.ErrNoIdentity) {
		t.Fatalf("error = %v, want ErrNoIdentity", err)
	}

	fake.CreateIdentity()

	public, err := client.GetPublicIdentity(t.Context())
	if err != nil || public.AccountFingerprint == "" {
		t.Errorf("identity = %+v, error = %v, want the created identity", public, err)
	}
}

func TestAFailureOfTheHostIsNotErrNoIdentity(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	broken := errors.New("the keychain refuses")
	fake.Fail(broken)
	client := newClient(t, fake)

	_, signErr := client.SignAsDevice(t.Context(), []byte("challenge"))
	_, readErr := client.GetPublicIdentity(t.Context())

	for name, err := range map[string]error{"sign": signErr, "read": readErr} {
		if err == nil || errors.Is(err, account.ErrNoIdentity) {
			t.Errorf("%s: error = %v, want a failure that is not ErrNoIdentity", name, err)
		}
	}
}

func TestSignAsDevice_RefusesAMessageOfNoBytesAndOneOverTheLimit(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)

	for name, message := range map[string][]byte{"empty": nil, "over the limit": bytes.Repeat([]byte{1}, 4097)} {
		if _, err := client.SignAsDevice(t.Context(), message); err == nil {
			t.Errorf("%s: a message was signed", name)
		}
	}
	if _, err := client.SignAsDevice(t.Context(), bytes.Repeat([]byte{1}, 4096)); err != nil {
		t.Errorf("a message of 4096 bytes: %v", err)
	}
}
