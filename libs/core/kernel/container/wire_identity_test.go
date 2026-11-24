package container

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
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

// saveDialog answers every save dialog with one path.
type saveDialog struct{ path string }

func (saveDialog) ChooseFolder(context.Context, string, string) (string, bool, error) {
	return "", false, nil
}

func (saveDialog) ChooseFile(context.Context, string, string, []string) (string, bool, error) {
	return "", false, nil
}

func (d saveDialog) ChooseSaveFile(context.Context, string, string, []string) (string, bool, error) {
	return d.path, true, nil
}

func getWiring() vaultWiring {
	return vaultWiring{Clock: testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))}
}

func TestOpenIdentityUseCases_ServesNothingWhereTheInstallationHasNoKeyStore(t *testing.T) {
	t.Parallel()
	if got := openIdentityUseCases(pluginInstall{}, getWiring(), &Resources{}, nil); got != nil {
		t.Errorf("openIdentityUseCases() = %+v, want nil", got)
	}
}

func TestOpenIdentityUseCases_KeepsAnIdentityInTheKeyStoreOfTheInstallation(t *testing.T) {
	t.Parallel()
	keys := memoryKeys{}
	res := &Resources{ChainStore: &memoryChain{}, progress: identity.NewProgress()}
	use := openIdentityUseCases(pluginInstall{KeyStore: keys}, getWiring(), res, nil)
	if use == nil || use.Identity.Get == nil || use.Identity.GenerateWords == nil || use.Identity.Create == nil || use.Identity.Delete == nil ||
		use.Identity.StartRestore == nil || use.Kit.Save == nil || use.Kit.Restore == nil {
		t.Fatalf("openIdentityUseCases() = %+v, want all seven use cases", use)
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

func TestOpenIdentityUseCases_SavesTheKitAsAPDFInTheFileOfTheDialog(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kit.pdf")
	use := openIdentityUseCases(pluginInstall{KeyStore: memoryKeys{}}, getWiring(), &Resources{ChainStore: &memoryChain{}, progress: identity.NewProgress()}, saveDialog{path: path})
	if use == nil {
		t.Fatal("openIdentityUseCases() = nil")
	}
	words, err := use.Identity.GenerateWords.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := use.Identity.Create.Execute(t.Context(), words); err != nil {
		t.Fatal(err)
	}
	isSaved, err := use.Kit.Save.Execute(t.Context(), words, nil)
	if err != nil || !isSaved {
		t.Fatalf("SaveRecoveryKit = %v, %v", isSaved, err)
	}
	content, err := os.ReadFile(path)
	if err != nil || !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Errorf("the file of the dialog holds no PDF: %v", err)
	}
}

func TestOpenIdentityUseCases_SavesNoKitWithoutADialog(t *testing.T) {
	t.Parallel()
	use := openIdentityUseCases(pluginInstall{KeyStore: memoryKeys{}}, getWiring(), &Resources{ChainStore: &memoryChain{}, progress: identity.NewProgress()}, nil)
	if use == nil {
		t.Fatal("openIdentityUseCases() = nil")
	}
	words, err := use.Identity.GenerateWords.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := use.Identity.Create.Execute(t.Context(), words); err != nil {
		t.Fatal(err)
	}
	if _, err := use.Kit.Save.Execute(t.Context(), words, nil); !errors.Is(err, machine.ErrNoFolderDialog) {
		t.Errorf("error = %v, want ErrNoFolderDialog", err)
	}
}

// kitSaver keeps the kit it is given, as the person who saves the PDF would.
type kitSaver struct{ kit identitydomain.Kit }

func (s *kitSaver) SaveKit(_ context.Context, kit identitydomain.Kit) (bool, error) {
	s.kit = kit
	return true, nil
}

func TestOpenIdentityUseCases_RestoresTheAccountFromTwoSheetsOfTheInstallation(t *testing.T) {
	t.Parallel()
	keys := memoryKeys{}
	clock := testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	res := &Resources{KeyStore: keys, Now: clock, ChainStore: &memoryChain{}, progress: identity.NewProgress()}
	use := openIdentityUseCases(pluginInstall{KeyStore: keys}, vaultWiring{Clock: clock}, res, nil)
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
	saver := &kitSaver{}
	save := identity.NewSaveRecoveryKit(identity.NewDeps(keys, names{}, clock).WithProgress(res.progress).WithChains(res.ChainStore), saver)
	if _, err := save.Execute(t.Context(), words, nil); err != nil {
		t.Fatal(err)
	}
	if len(saver.kit.Sheets) != 3 {
		t.Fatalf("kit has %d sheets, want 3", len(saver.kit.Sheets))
	}
	sheets := [][]string{saver.kit.Sheets[2].Words, saver.kit.Sheets[0].Words}
	if err := use.Identity.Delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := use.Kit.Restore.Execute(t.Context(), sheets, ""); err != nil {
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

// memoryChain is the chain file of a device kept in memory.
type memoryChain struct {
	mu    sync.Mutex
	chain []byte
}

func (m *memoryChain) Read(context.Context) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chain == nil {
		return nil, identitydomain.ErrNoChain
	}
	return append([]byte(nil), m.chain...), nil
}

func (m *memoryChain) Write(_ context.Context, chain []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chain = append([]byte(nil), chain...)
	return nil
}

func (m *memoryChain) Delete(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chain = nil
	return nil
}

type names struct{}

func (names) GetInstanceName() string { return "Laptop" }
