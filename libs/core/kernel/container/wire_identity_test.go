package container

import (
	"context"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

type memoryKeys map[string]string

func (m memoryKeys) GetKey(_ context.Context, name string) (string, error) {
	if value, ok := m[name]; ok {
		return value, nil
	}
	return "", plugindomain.ErrNoKey
}

func (m memoryKeys) SetKey(_ context.Context, name, key string) error { m[name] = key; return nil }

func (m memoryKeys) DeleteKey(_ context.Context, name string) error {
	if _, ok := m[name]; !ok {
		return plugindomain.ErrNoKey
	}
	delete(m, name)
	return nil
}

func TestOpenIdentityUseCases_ServesNothingWhereTheInstallationHasNoKeyStore(t *testing.T) {
	t.Parallel()
	if got := openIdentityUseCases(pluginInstall{}, vaultWiring{Clock: testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))}, nil); got != nil {
		t.Errorf("openIdentityUseCases() = %+v, want nil", got)
	}
}

func TestOpenIdentityUseCases_KeepsAnIdentityInTheKeyStoreOfTheInstallation(t *testing.T) {
	t.Parallel()
	keys := memoryKeys{}
	use := openIdentityUseCases(pluginInstall{KeyStore: keys}, vaultWiring{Clock: testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))}, nil)
	if use == nil || use.Identity.Get == nil || use.Identity.GenerateWords == nil || use.Identity.Create == nil || use.Identity.Delete == nil ||
		use.Identity.StartRestore == nil || use.Kit.Create == nil || use.Kit.Check == nil || use.Kit.Restore == nil {
		t.Fatalf("openIdentityUseCases() = %+v, want all eight use cases", use)
	}
	words, err := use.Identity.GenerateWords.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	created, err := use.Identity.Create.Execute(t.Context(), words)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := keys["identity"]; !ok {
		t.Errorf("the key store holds %v, want the entry \"identity\"", keys)
	}
	if created.DeviceName == "" {
		t.Error("the device has no name: the licensing adapter is not wired as the name of the device")
	}
}

func TestOpenIdentityUseCases_RestoresTheAccountFromTwoSheetsOfTheInstallation(t *testing.T) {
	t.Parallel()
	keys := memoryKeys{}
	clock := testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	res := &Resources{KeyStore: keys, Now: clock, restore: identity.NewPendingStore()}
	use := openIdentityUseCases(pluginInstall{KeyStore: keys}, vaultWiring{Clock: clock}, res.restore)
	if use == nil {
		t.Fatal("openIdentityUseCases() = nil")
	}
	words, err := use.Identity.GenerateWords.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	created, err := use.Identity.Create.Execute(t.Context(), words)
	if err != nil {
		t.Fatal(err)
	}
	made, err := use.Kit.Create.Execute(t.Context(), words, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(made.Sheets) != 3 {
		t.Fatalf("kit has %d sheets, want 3", len(made.Sheets))
	}
	sheets := [][]string{made.Sheets[2].Words, made.Sheets[0].Words}
	fits, err := use.Kit.Check.Execute(t.Context(), sheets)
	if err != nil || fits != created.AccountFingerprint {
		t.Fatalf("CheckKit() = %q, %v; want %q", fits, err, created.AccountFingerprint)
	}
	if err := use.Identity.Delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := use.Kit.Restore.Execute(t.Context(), sheets); err != nil {
		t.Fatal(err)
	}
	if err := openAccount(res).CompleteRestore(t.Context(), syncPluginID, nil); err != nil {
		t.Fatal(err)
	}
	restored, err := use.Identity.Get.Execute(t.Context())
	if err != nil || restored.AccountFingerprint != created.AccountFingerprint {
		t.Errorf("restored = %+v, error = %v, want the account %q", restored, err, created.AccountFingerprint)
	}
}
