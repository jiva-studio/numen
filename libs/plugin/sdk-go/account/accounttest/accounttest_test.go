package accounttest_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/identity"
)

func newClient(t *testing.T, fake *accounttest.Fake) *account.Client {
	t.Helper()
	endpoint, err := hostcall.ReadVars(fake.BuildVars("token"))
	if err != nil {
		t.Fatal(err)
	}
	return account.NewClient(endpoint)
}

func TestTwoFakesHoldTwoIdentities(t *testing.T) {
	first := accounttest.NewFake(t, "md.numen.a")
	second := accounttest.NewFake(t, "md.numen.a")

	if first.Public().AccountFingerprint == second.Public().AccountFingerprint {
		t.Error("two fakes have the same account")
	}
}

func TestTheFakeSignsWithTheKeyOfItsRecordForThePluginItWasMadeFor(t *testing.T) {
	fake := accounttest.NewFake(t, "md.numen.a")
	client := newClient(t, fake)

	signature, err := client.SignAsDevice(t.Context(), []byte("m"))
	if err != nil {
		t.Fatal(err)
	}

	if !identity.VerifyForPlugin(fake.Public().DeviceSignKey, "md.numen.a", []byte("m"), [64]byte(signature)) {
		t.Error("the signature is not the one of the key of the record")
	}
}

func TestTheFakeIsAHostAPluginReachesByTheEnvironment(t *testing.T) {
	fake := accounttest.NewFake(t, "md.numen.a")

	vars := fake.BuildVars("token")

	if vars[hostcall.PluginTokenEnv] != "token" || vars[hostcall.HostPortEnv] == "" {
		t.Errorf("vars = %v", vars)
	}
}

func TestAFakeThatStartedARestoreHasNoIdentityUntilTheRestoreCompletes(t *testing.T) {
	fake := accounttest.NewFake(t, "md.numen.a")
	fake.RemoveIdentity()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	fake.StartRestore(words.List())
	client := newClient(t, fake)

	if _, err := client.GetPublicIdentity(t.Context()); !errors.Is(err, account.ErrNoIdentity) {
		t.Fatalf("error = %v, want ErrNoIdentity while the restore is pending", err)
	}
	if err := client.CompleteRestore(t.Context(), nil); err != nil {
		t.Fatal(err)
	}

	root, err := words.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	if fake.Public().AccountFingerprint == "" || fake.Public().DeviceSignKey == nil {
		t.Errorf("public = %+v, want the identity the restore made", fake.Public())
	}
	pending, err := client.GetPendingRestore(t.Context())
	if err != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", pending, err)
	}
	chain, err := identity.ParseChain(fake.Public().Chain)
	if err != nil || !bytes.Equal(chain.Root().Sign, root.Public().Sign) {
		t.Errorf("the chain of the fake has another root than the words, error = %v", err)
	}
}
