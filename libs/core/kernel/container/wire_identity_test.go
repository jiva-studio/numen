package container

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	if got := openIdentityUseCases(pluginInstall{}, getWiring(), nil); got != nil {
		t.Errorf("openIdentityUseCases() = %+v, want nil", got)
	}
}

func TestOpenIdentityUseCases_KeepsAnIdentityInTheKeyStoreOfTheInstallation(t *testing.T) {
	t.Parallel()
	keys := memoryKeys{}
	use := openIdentityUseCases(pluginInstall{Keys: identity.SharedKeys{Store: keys}}, getWiring(), nil)
	if use == nil || use.Identity.Get == nil || use.Identity.GenerateWords == nil || use.Identity.Create == nil || use.Identity.Delete == nil ||
		use.Kit.Save == nil || use.Kit.Restore == nil {
		t.Fatalf("openIdentityUseCases() = %+v, want all six use cases", use)
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
	use := openIdentityUseCases(pluginInstall{Keys: identity.SharedKeys{Store: memoryKeys{}}}, getWiring(), saveDialog{path: path})
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
	use := openIdentityUseCases(pluginInstall{Keys: identity.SharedKeys{Store: memoryKeys{}}}, getWiring(), nil)
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
