package container_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func openPluginVault(t *testing.T, notes map[string]string) (domain.Vault, func() (domain.Vault, bool), windowVaults) {
	t.Helper()
	cfg := container.NewConfig(container.Config{IndexPath: indexfile.Path(t), PluginCachePath: t.TempDir(), PluginHost: container.NewPluginHost("md.numen.test", t.TempDir()), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")})
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	v := testsupport.NewVault(t, notes)
	if err := registry.Saver.Save(v); err != nil {
		t.Fatal(err)
	}
	return v, func() (domain.Vault, bool) { return v, true }, windowVaults{cfg: cfg, registry: registry, db: db}
}

func TestAPluginNoteIsInTheIndexOnceItsFileIsScanned(t *testing.T) {
	v, current, assembly := openPluginVault(t, getNote())
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{})
	if err != nil {
		t.Fatal(err)
	}

	if err := shared.CreateFile(t.Context(), "Made.md", []byte("---\ntitle: Made\n---\n\n# Made\n")); err != nil {
		t.Fatal(err)
	}
	if err := shared.Scan(t.Context(), []string{"Made.md"}); err != nil {
		t.Fatal(err)
	}

	got, err := shared.GetNotes(t.Context(), []string{"Made.md"})
	if err != nil || got["Made.md"].Title != "Made" {
		t.Errorf("got %+v (%v) in %s", got, err, v.Name)
	}
}

func TestAPluginKeepsItsFolderAndItsCacheApart(t *testing.T) {
	_, current, assembly := openPluginVault(t, getNote())
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := shared.OpenStore("org.example.sample")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := shared.OpenStoreWriter("org.example.sample")
	if err != nil {
		t.Fatal(err)
	}

	if err := writer.Append(t.Context(), "log", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := shared.WriteCache(t.Context(), "org.example.sample", "due", []byte("3")); err != nil {
		t.Fatal(err)
	}

	got, err := data.Read(t.Context(), "log")
	cached, found, cacheErr := shared.ReadCache(t.Context(), "org.example.sample", "due")
	_, missing := data.Read(t.Context(), "due")
	if err != nil || string(got) != "a" || cacheErr != nil || !found || string(cached) != "3" || !errors.Is(missing, fs.ErrNotExist) {
		t.Errorf("data %q (%v), cache %q %v (%v), missing %v", got, err, cached, found, cacheErr, missing)
	}
}

func TestAnotherVaultOfTheInstallationIsWorkedByItsIdentity(t *testing.T) {
	v, _, assembly := openPluginVault(t, getNote())
	nothing := func() (domain.Vault, bool) { return domain.Vault{}, false }
	shared, err := assembly.OpenPluginVault(nothing, container.PluginVaultHooks{})
	if err != nil {
		t.Fatal(err)
	}

	recognized, listErr := shared.GetVaults()
	linked, withErr := shared.With(v.ID)
	_, unknown := shared.With("nobody")

	if listErr != nil || len(recognized) != 1 || recognized[0].ID != v.ID {
		t.Errorf("the vaults were %v (%v)", recognized, listErr)
	}
	if withErr != nil {
		t.Fatal(withErr)
	}
	if got, showing := linked.GetVault(); !showing || got.ID != v.ID {
		t.Errorf("the bound vault was %v", got)
	}
	if !errors.Is(unknown, fs.ErrNotExist) {
		t.Errorf("an unknown identity was answered %v", unknown)
	}
}

func TestAWriteByAPluginIsReportedToTheWindow(t *testing.T) {
	_, current, assembly := openPluginVault(t, getNote())
	notifyCount := 0
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{Written: func() { notifyCount++ }})
	if err != nil {
		t.Fatal(err)
	}

	if err := shared.CreateFile(t.Context(), "Made.md", []byte("# Made\n")); err != nil {
		t.Fatal(err)
	}

	if notifyCount != 1 {
		t.Errorf("the window received %d times", notifyCount)
	}
}
