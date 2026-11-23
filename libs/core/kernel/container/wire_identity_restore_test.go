package container

import (
	"errors"
	"strings"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const syncPluginID = "md.numen.tools.sync"

func TestTheAccountOfThePluginsSeesTheRestoreTheWindowStarted(t *testing.T) {
	t.Parallel()
	keys := memoryKeys{}
	clock := testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	chains := &memoryChain{}
	res := &Resources{KeyStore: keys, ChainStore: chains, Now: clock, progress: identity.NewProgress()}
	use := openIdentityUseCases(pluginInstall{KeyStore: keys}, vaultWiring{Clock: clock}, res)
	account := openAccount(res)
	words := strings.Fields("legal winner thank year wave sausage worth useful legal winner thank yellow")
	if use == nil || use.Identity.StartRestore == nil {
		t.Fatalf("openIdentityUseCases() = %+v, want a use case that starts a restore", use)
	}

	if err := use.Identity.StartRestore.Execute(t.Context(), words); err != nil {
		t.Fatal(err)
	}

	pending, err := account.GetPendingRestore(t.Context())
	if err != nil || !pending.IsPending || len(pending.RootSignKey) != 32 || len(pending.DeviceSignKey) != 32 {
		t.Fatalf("pending = %+v, error = %v, want the restore the window started", pending, err)
	}
	signature, err := account.SignAsRecovery(t.Context(), syncPluginID, []byte("chain read"))
	if err != nil || !identitylib.VerifyForRecovery(pending.RootSignKey, syncPluginID, []byte("chain read"), [64]byte(signature)) {
		t.Errorf("signature = %x, error = %v, want a signature of the recovery key of the words", signature, err)
	}
	if len(keys) != 0 {
		t.Errorf("the key store holds %v, want nothing before the restore completes", keys)
	}
}

func TestACompleteRestoreOfThePluginsMakesTheIdentityOfTheWindowAndTheKeysOfThePlugins(t *testing.T) {
	t.Parallel()
	keys := memoryKeys{}
	clock := testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	chains := &memoryChain{}
	res := &Resources{KeyStore: keys, ChainStore: chains, Now: clock, progress: identity.NewProgress()}
	use := openIdentityUseCases(pluginInstall{KeyStore: keys}, vaultWiring{Clock: clock}, res)
	account := openAccount(res)
	words := strings.Fields("legal winner thank year wave sausage worth useful legal winner thank yellow")
	if err := use.Identity.StartRestore.Execute(t.Context(), words); err != nil {
		t.Fatal(err)
	}

	if err := account.CompleteRestore(t.Context(), syncPluginID, nil); err != nil {
		t.Fatal(err)
	}

	if shown, err := use.Identity.Get.Execute(t.Context()); err != nil || shown.AccountFingerprint == "" {
		t.Fatalf("identity = %+v, error = %v, want the identity the plugin completed", shown, err)
	}
	sealed, err := account.SealData(t.Context(), syncPluginID, []byte("context"), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	if opened, err := account.OpenData(t.Context(), syncPluginID, []byte("context"), sealed); err != nil || string(opened) != "milk" {
		t.Errorf("opened = %q, error = %v, want milk", opened, err)
	}
	if _, err := account.OpenData(t.Context(), "md.numen.tools.link", []byte("context"), sealed); !errors.Is(err, identitydomain.ErrCannotOpen) {
		t.Errorf("another plugin: error = %v, want ErrCannotOpen", err)
	}
	if _, err := account.SignAsRecovery(t.Context(), syncPluginID, []byte("m")); !errors.Is(err, identitydomain.ErrNoPendingRestore) {
		t.Errorf("SignAsRecovery() after completion: error = %v, want ErrNoPendingRestore", err)
	}
}

func TestTheAccountOfThePluginsRefusesTheRestoreCallsWhereTheInstallationHasNoKeyStore(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	account := openAccount(&Resources{Now: clock})

	if _, err := account.SealData(t.Context(), syncPluginID, nil, []byte("m")); err == nil {
		t.Error("SealData() ran with no key store")
	}
	if err := account.CompleteRestore(t.Context(), syncPluginID, nil); err == nil {
		t.Error("CompleteRestore() ran with no key store")
	}
}
