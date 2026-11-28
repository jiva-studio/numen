package account_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	dataContext = "numen-sync/catalog/v1"
	maxData     = 32 << 10
)

type firstDevice struct {
	words []string
	root  identity.RecoveryKey
	keys  identity.AccountKeys
	chain identity.Chain
}

func newFirstDevice(tb testing.TB) firstDevice {
	tb.Helper()
	recovery, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	root, err := recovery.DeriveRecoveryKey()
	if err != nil {
		tb.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	chain, err := identity.NewChain(root, keys, clock.System{}.Now())
	if err != nil {
		tb.Fatal(err)
	}
	chain, err = chain.AppendDevice(root, device.Public(), "Desktop", clock.System{}.Now())
	if err != nil {
		tb.Fatal(err)
	}
	return firstDevice{words: recovery.List(), root: root, keys: keys, chain: chain}
}

func newRestoringClient(t *testing.T, words []string) (*account.Client, *accounttest.Fake) {
	t.Helper()
	fake := accounttest.NewFake(t, pluginID)
	fake.RemoveIdentity()
	fake.StartRestore(words)
	return newClient(t, fake), fake
}

func TestSealData_ReturnsBytesThatOpenDataOpensForTheSameContext(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)

	sealed, err := client.SealData(t.Context(), []byte(dataContext), []byte("Shopping list"))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := client.OpenData(t.Context(), []byte(dataContext), sealed)

	if err != nil || string(opened) != "Shopping list" {
		t.Errorf("opened = %q, error = %v, want the plain text", opened, err)
	}
	if bytes.Contains(sealed, []byte("Shopping")) {
		t.Error("the sealed bytes hold the plain text")
	}
}

func TestSealData_SealsWithTheAccountKeysOfTheIdentityOfTheHost(t *testing.T) {
	first := newFirstDevice(t)
	fake := accounttest.NewFake(t, pluginID)
	fake.RemoveIdentity()
	fake.StartRestore(first.words)
	client := newClient(t, fake)
	if err := client.CompleteRestore(t.Context(), first.chain.Bytes()); err != nil {
		t.Fatal(err)
	}
	sealedByFirst, err := first.keys.SealForPlugin(rand.Reader, pluginID, []byte(dataContext), []byte("catalog"))
	if err != nil {
		t.Fatal(err)
	}

	opened, err := client.OpenData(t.Context(), []byte(dataContext), sealedByFirst)

	if err != nil || string(opened) != "catalog" {
		t.Errorf("opened = %q, error = %v, want the catalog sealed by the first device", opened, err)
	}
	sealed, err := client.SealData(t.Context(), []byte(dataContext), []byte("renamed"))
	if err != nil {
		t.Fatal(err)
	}
	if back, err := first.keys.OpenForPlugin(pluginID, []byte(dataContext), sealed); err != nil || string(back) != "renamed" {
		t.Errorf("the first device opened %q with error %v, want renamed", back, err)
	}
}

func TestOpenData_ReportsErrCannotOpenForAnotherContextAnotherAccountAndChangedBytes(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)
	sealed, err := client.SealData(t.Context(), []byte(dataContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Clone(sealed)
	changed[len(changed)-1] ^= 1
	otherAccount := newClient(t, accounttest.NewFake(t, pluginID))

	for name, call := range map[string]func() ([]byte, error){
		"another context": func() ([]byte, error) { return client.OpenData(t.Context(), []byte("other"), sealed) },
		"another account": func() ([]byte, error) { return otherAccount.OpenData(t.Context(), []byte(dataContext), sealed) },
		"changed bytes":   func() ([]byte, error) { return client.OpenData(t.Context(), []byte(dataContext), changed) },
		"no bytes":        func() ([]byte, error) { return client.OpenData(t.Context(), []byte(dataContext), nil) },
	} {
		opened, err := call()
		if !errors.Is(err, account.ErrCannotOpen) || opened != nil {
			t.Errorf("%s: opened = %q, error = %v, want nil and ErrCannotOpen", name, opened, err)
		}
	}
}

func TestSealDataAndOpenData_AnswerErrNoIdentityWhereTheHostHasNone(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	fake.RemoveIdentity()
	client := newClient(t, fake)

	_, sealErr := client.SealData(t.Context(), []byte(dataContext), []byte("milk"))
	_, openErr := client.OpenData(t.Context(), []byte(dataContext), []byte("sealed"))

	if !errors.Is(sealErr, account.ErrNoIdentity) || !errors.Is(openErr, account.ErrNoIdentity) {
		t.Errorf("seal = %v, open = %v, want ErrNoIdentity", sealErr, openErr)
	}
}

func TestSealDataAndOpenData_ReportAFailureOfTheHostThatIsNotAnAnswerAboutTheData(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	fake.Fail(errors.New("the keychain refuses"))
	client := newClient(t, fake)

	_, sealErr := client.SealData(t.Context(), []byte(dataContext), []byte("milk"))
	_, openErr := client.OpenData(t.Context(), []byte(dataContext), []byte("sealed"))

	for name, err := range map[string]error{"seal": sealErr, "open": openErr} {
		if err == nil || errors.Is(err, account.ErrNoIdentity) || errors.Is(err, account.ErrCannotOpen) {
			t.Errorf("%s: error = %v, want a failure that is neither ErrNoIdentity nor ErrCannotOpen", name, err)
		}
	}
}

func TestSealDataAndOpenData_RefuseDataOver32KiBAndAcceptExactly32KiB(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)

	if _, err := client.SealData(t.Context(), []byte(dataContext), bytes.Repeat([]byte{1}, maxData+1)); err == nil {
		t.Error("data over 32 KiB was sealed")
	}
	if _, err := client.OpenData(t.Context(), []byte(dataContext), bytes.Repeat([]byte{1}, 2*maxData)); err == nil || errors.Is(err, account.ErrCannotOpen) {
		t.Errorf("sealed data far over 32 KiB: error = %v, want a refusal of the size", err)
	}
	sealed, err := client.SealData(t.Context(), []byte(dataContext), bytes.Repeat([]byte{1}, maxData))
	if err != nil {
		t.Fatalf("32 KiB: %v", err)
	}
	if opened, err := client.OpenData(t.Context(), []byte(dataContext), sealed); err != nil || len(opened) != maxData {
		t.Errorf("opened %d bytes, error = %v, want 32 KiB back", len(opened), err)
	}
}

func TestGetPendingRestore_ReportsNothingPendingWhereNoRestoreStarted(t *testing.T) {
	client := newClient(t, accounttest.NewFake(t, pluginID))

	pending, err := client.GetPendingRestore(t.Context())

	if err != nil || pending.IsPending || len(pending.RootSignKey) != 0 || len(pending.DeviceSignKey) != 0 {
		t.Errorf("pending = %+v, error = %v, want nothing pending", pending, err)
	}
}

func TestGetPendingRestore_ReportsThePublicKeysOfTheRestoreTheHostStarted(t *testing.T) {
	first := newFirstDevice(t)
	client, _ := newRestoringClient(t, first.words)

	pending, err := client.GetPendingRestore(t.Context())

	if err != nil || !pending.IsPending || !bytes.Equal(pending.RootSignKey, first.root.Public().Sign) || len(pending.DeviceSignKey) != 32 {
		t.Errorf("pending = %+v, error = %v, want the root key of the words and a device key", pending, err)
	}
}

func TestSignAsRecovery_ReturnsASignatureOfTheRecoveryKeyForThePluginID(t *testing.T) {
	first := newFirstDevice(t)
	client, _ := newRestoringClient(t, first.words)

	signature, err := client.SignAsRecovery(t.Context(), []byte("chain read"))
	if err != nil {
		t.Fatal(err)
	}

	root := first.root.Public().Sign
	if len(signature) != 64 || !identity.VerifyForRecovery(root, pluginID, []byte("chain read"), [64]byte(signature)) {
		t.Error("the signature does not verify for the plugin under the recovery key of the words")
	}
	if identity.VerifyForRecovery(root, "md.numen.tools.link", []byte("chain read"), [64]byte(signature)) {
		t.Error("the signature verifies for another plugin")
	}
}

func TestSignAsRecovery_RefusesAMessageOfNoBytesAndOneOverTheLimit(t *testing.T) {
	client, _ := newRestoringClient(t, newFirstDevice(t).words)

	for name, message := range map[string][]byte{"empty": nil, "over the limit": bytes.Repeat([]byte{1}, 4097)} {
		if _, err := client.SignAsRecovery(t.Context(), message); err == nil {
			t.Errorf("%s: a message was signed", name)
		}
	}
	if _, err := client.SignAsRecovery(t.Context(), bytes.Repeat([]byte{1}, 4096)); err != nil {
		t.Errorf("a message of 4096 bytes: %v", err)
	}
}

func TestSignAsRecoveryAndCompleteRestore_AnswerErrNoPendingRestoreWhereNothingIsPending(t *testing.T) {
	client := newClient(t, accounttest.NewFake(t, pluginID))

	_, signErr := client.SignAsRecovery(t.Context(), []byte("chain read"))
	completeErr := client.CompleteRestore(t.Context(), nil)

	if !errors.Is(signErr, account.ErrNoPendingRestore) || !errors.Is(completeErr, account.ErrNoPendingRestore) {
		t.Errorf("sign = %v, complete = %v, want ErrNoPendingRestore", signErr, completeErr)
	}
}

func TestCompleteRestore_WithTheChainOfTheServerJoinsTheAccountOfTheFirstDevice(t *testing.T) {
	first := newFirstDevice(t)
	client, fake := newRestoringClient(t, first.words)
	pending, err := client.GetPendingRestore(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if err := client.CompleteRestore(t.Context(), first.chain.Bytes()); err != nil {
		t.Fatal(err)
	}

	public, err := client.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chain, err := identity.ParseChain(public.Chain)
	if err != nil {
		t.Fatalf("the chain does not verify: %v", err)
	}
	devices := chain.ListDevices()
	if public.AccountFingerprint != first.chain.Root().Fingerprint().String() || !chain.IsExtensionOf(first.chain) ||
		len(devices) != 2 || !bytes.Equal(devices[1].Public.Sign, pending.DeviceSignKey) || !bytes.Equal(public.DeviceSignKey, pending.DeviceSignKey) {
		t.Errorf("identity = %+v with %d devices, want the account of the first device and the pending device added", public, len(devices))
	}
	if after, err := client.GetPendingRestore(t.Context()); err != nil || after.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", after, err)
	}
	if fake.Public().AccountFingerprint != public.AccountFingerprint {
		t.Error("the fake and the client report different accounts")
	}
}

func TestCompleteRestore_WithNoChainMakesANewIdentityOfTheWords(t *testing.T) {
	first := newFirstDevice(t)
	client, _ := newRestoringClient(t, first.words)

	if err := client.CompleteRestore(t.Context(), nil); err != nil {
		t.Fatal(err)
	}

	public, err := client.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chain, err := identity.ParseChain(public.Chain)
	if err != nil || len(chain.ListDevices()) != 1 || public.AccountFingerprint != first.chain.Root().Fingerprint().String() {
		t.Errorf("identity = %+v, chain error = %v, want one device on the account of the words", public, err)
	}
}

func TestCompleteRestore_RefusesAChainOfAnotherRootAndABrokenChain(t *testing.T) {
	first := newFirstDevice(t)
	stranger := newFirstDevice(t)
	broken := bytes.Clone(first.chain.Bytes())
	broken[len(broken)/2] ^= 1

	for name, chain := range map[string][]byte{
		"another root":            stranger.chain.Bytes(),
		"a broken chain":          broken,
		"bytes that are no chain": []byte("not a chain"),
	} {
		client, _ := newRestoringClient(t, first.words)
		if err := client.CompleteRestore(t.Context(), chain); !errors.Is(err, account.ErrInvalidChain) {
			t.Errorf("%s: error = %v, want ErrInvalidChain", name, err)
		}
		if _, err := client.GetPublicIdentity(t.Context()); !errors.Is(err, account.ErrNoIdentity) {
			t.Errorf("%s: GetPublicIdentity() error = %v, want ErrNoIdentity: the refusal wrote nothing", name, err)
		}
	}
}

func TestRestoreCalls_AnswerErrNoIdentityNeverForAPendingRestore(t *testing.T) {
	client, _ := newRestoringClient(t, newFirstDevice(t).words)

	if _, err := client.GetPublicIdentity(t.Context()); !errors.Is(err, account.ErrNoIdentity) {
		t.Errorf("GetPublicIdentity() error = %v, want ErrNoIdentity while the restore is pending", err)
	}
	if _, err := client.SignAsRecovery(t.Context(), []byte("m")); errors.Is(err, account.ErrNoIdentity) {
		t.Errorf("SignAsRecovery() error = %v, want the call to work while the restore is pending", err)
	}
}
