package vault_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// isolateHome points every directory a test could reach through the
// environment at a folder of its own.
func isolateHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// unwritable is a set of writers that cannot open any vault.
type unwritable struct {
	vaultport.VaultWriters
	err error
}

func (u unwritable) Open(domain.Vault) (vaultport.VaultWriter, error) { return nil, u.err }

// flawed is a machine handing files over, with the failures it is told to have.
type flawed struct {
	handedOver
	statFails map[string]error
	listFails map[string]error
	openFails map[string]error
	onList    func()
	neither   map[string]bool
}

func (f flawed) GetImportedFile(ctx context.Context, handle string) (sourceport.ImportedFile, error) {
	if f.neither[handle] {
		return sourceport.ImportedFile{Name: f.names[handle], Handle: handle}, nil
	}
	if err := f.statFails[handle]; err != nil {
		return sourceport.ImportedFile{}, err
	}
	return f.handedOver.GetImportedFile(ctx, handle)
}

func (f flawed) List(ctx context.Context, handle string) ([]sourceport.ImportedFile, error) {
	if f.onList != nil {
		f.onList()
	}
	if err := f.listFails[handle]; err != nil {
		return nil, err
	}
	return f.handedOver.List(ctx, handle)
}

func (f flawed) Open(ctx context.Context, handle string) (io.ReadCloser, error) {
	if err := f.openFails[handle]; err != nil {
		return nil, err
	}
	return f.handedOver.Open(ctx, handle)
}

// aFolderOfTwo is a folder holding a picture and a note, by handle.
func aFolderOfTwo() handedOver {
	return handedOver{
		names: map[string]string{
			"content://held/1": "scans",
			"content://held/2": "Cover.png",
			"content://held/3": "Kelvin.md",
		},
		bodies: map[string]string{
			"content://held/2": "PNG",
			"content://held/3": "# Kelvin\n",
		},
		holding: map[string][]string{
			"content://held/1": {"content://held/2", "content://held/3"},
		},
	}
}

// Nothing handed over is nothing done, whether or not the vault can be written.
func TestNothingHandedOverIsNothingCompleted(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	bring := vaults.NewImport(unwritable{err: errors.New("the vault cannot be written")}, filesystem.ImportedFiles{})

	brought, err := bring.Execute(t.Context(), v, "", nil)
	if err != nil {
		t.Fatalf("an empty drop failed: %v", err)
	}
	if len(brought.Landed) != 0 || len(brought.Errors) != 0 {
		t.Errorf("an empty drop came to %+v", brought)
	}
}

// A vault that cannot be written takes nothing, and the caller is told why.
func TestAVaultThatCannotBeSavedIsAnError(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	cannot := errors.New("the vault cannot be written")
	bring := vaults.NewImport(unwritable{err: cannot}, aFolderOfTwo())

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/2"})
	if !errors.Is(err, cannot) {
		t.Fatalf("got %v", err)
	}
	if len(brought.Errors) != 0 {
		t.Errorf("the rejection was reported as a file: %v", brought.Errors)
	}
}

// A drop that was cancelled ends, and nothing is reported as a file that failed.
func TestACancelledDropEndsWithTheCancellation(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, aFolderOfTwo())

	brought, err := bring.Execute(ctx, v, "", []string{"content://held/2"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if len(brought.Errors) != 0 || len(brought.Landed) != 0 {
		t.Errorf("a cancelled drop came to %+v", brought)
	}
}

// A drop cancelled while a folder is being read leaves what is under it
// outside.
func TestADropCancelledInsideAFolderRestoresNothingBelowIt(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	machine := flawed{handedOver: aFolderOfTwo(), onList: cancel}
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, machine)

	brought, broughtErr := bring.Execute(ctx, v, "", []string{"content://held/1"})
	if broughtErr != nil {
		t.Fatal(broughtErr)
	}

	if len(brought.Landed) != 1 || brought.Landed[0] != "scans" {
		t.Errorf("what landed: %v", brought.Landed)
	}
	for _, name := range []string{"Cover.png", "Kelvin.md"} {
		if _, err := os.Stat(filepath.Join(v.Path, "scans", name)); err == nil {
			t.Errorf("%s arrived after the drop was cancelled", name)
		}
	}
	if len(brought.Errors) != 2 {
		t.Errorf("what stayed outside: %v", brought.Errors)
	}
}

// A handle that cannot be read stays outside under the error it gave.
func TestAHandleThatCannotBeReadStaysOutsideWithItsError(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	removed := errors.New("the file is gone")
	machine := flawed{
		handedOver: handedOver{names: map[string]string{"content://held/9": "Gone.png"}},
		statFails:  map[string]error{"content://held/9": removed},
	}
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, machine)

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/9"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 1 || brought.Errors[0].Name != "Gone.png" || !errors.Is(brought.Errors[0].Why, removed) {
		t.Errorf("what stayed outside: %+v", brought.Errors)
	}
	if len(brought.Landed) != 0 {
		t.Errorf("what landed: %v", brought.Landed)
	}
}

// A folder whose name a file of the vault already carries stays outside whole.
func TestAFolderOntoAFileOfTheSameNameStaysOutside(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, map[string]string{"scans": "a file"})
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, aFolderOfTwo())

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 1 || !errors.Is(brought.Errors[0].Why, vaultport.ErrAlreadyExists) {
		t.Fatalf("what stayed outside: %+v", brought.Errors)
	}
	if len(brought.Landed) != 0 {
		t.Errorf("what landed: %v", brought.Landed)
	}
}

// A folder that cannot be listed has arrived and holds nothing, and it is
// named with the error.
func TestAFolderThatCannotBeListedIsNamedWithTheError(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	unlistable := errors.New("the folder cannot be listed")
	machine := flawed{
		handedOver: aFolderOfTwo(),
		listFails:  map[string]error{"content://held/1": unlistable},
	}
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, machine)

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 1 || brought.Errors[0].Name != "scans" || !errors.Is(brought.Errors[0].Why, unlistable) {
		t.Errorf("what stayed outside: %+v", brought.Errors)
	}
}

// One file of a folder that cannot be read leaves the others to arrive, and is
// named under its own name.
func TestOneFileOfAFolderThatCannotBeReadIsNamedAndTheRestArrive(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	denied := errors.New("the file is not readable")
	machine := flawed{
		handedOver: aFolderOfTwo(),
		openFails:  map[string]error{"content://held/2": denied},
	}
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, machine)

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 1 || brought.Errors[0].Name != "Cover.png" || !errors.Is(brought.Errors[0].Why, denied) {
		t.Fatalf("what stayed outside: %+v", brought.Errors)
	}
	if _, err := os.Stat(filepath.Join(v.Path, "scans", "Cover.png")); err == nil {
		t.Error("the file that could not be read arrived")
	}
	if body := readArrived(t, v.Path, "scans/Kelvin.md"); body != "# Kelvin\n" {
		t.Errorf("the note beside it arrived as %q", body)
	}
}

// A file entry of a folder that cannot be examined is named under its own name.
func TestAnEntryOfAFolderThatCannotBeExaminedIsNamed(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	removed := errors.New("the entry is gone")
	machine := flawed{
		handedOver: aFolderOfTwo(),
		statFails:  map[string]error{"content://held/3": removed},
	}
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, machine)

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 1 || brought.Errors[0].Name != "Kelvin.md" || !errors.Is(brought.Errors[0].Why, removed) {
		t.Errorf("what stayed outside: %+v", brought.Errors)
	}
}

// ignoringContext is a set of writers that finish a copy whatever the context
// says.
type ignoringContext struct{ vaultport.VaultWriters }

func (i ignoringContext) Open(v domain.Vault) (vaultport.VaultWriter, error) {
	w, err := i.VaultWriters.Open(v)
	return deaf{w}, err
}

type deaf struct{ vaultport.VaultWriter }

func (d deaf) Import(ctx context.Context, to string, r io.Reader) error {
	return d.VaultWriter.Import(context.WithoutCancel(ctx), to, r)
}

// Files under a folder are not brought once the drop is cancelled, whatever the
// vault would do with them.
func TestNothingBelowAFolderIsBroughtOnceTheDropIsCancelled(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	machine := flawed{handedOver: aFolderOfTwo(), onList: cancel}
	bring := vaults.NewImport(ignoringContext{vaultwriter.VaultWriters{}}, machine)

	brought, broughtErr := bring.Execute(ctx, v, "", []string{"content://held/1"})
	if broughtErr != nil {
		t.Fatal(broughtErr)
	}

	if len(brought.Landed) != 1 {
		t.Errorf("what landed: %v", brought.Landed)
	}
}

// A drop cancelled after one handle ended keeps what that one brought, and the
// rest is not tried.
func TestADropCancelledAfterOneHandleReportsWhatArrived(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	machine := flawed{handedOver: aFolderOfTwo(), onList: cancel}
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, machine)

	brought, err := bring.Execute(ctx, v, "", []string{"content://held/1", "content://held/3"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if len(brought.Landed) != 1 || brought.Landed[0] != "scans" {
		t.Errorf("what landed: %v", brought.Landed)
	}
}

// Something that is neither a file nor a folder stays outside, and says so.
func TestSomethingNeitherFileNorFolderStaysOutside(t *testing.T) {
	isolateHome(t)
	v := testsupport.NewVault(t, nil)
	machine := flawed{
		handedOver: handedOver{names: map[string]string{"content://held/7": "Socket"}},
		neither:    map[string]bool{"content://held/7": true},
	}
	bring := vaults.NewImport(vaultwriter.VaultWriters{}, machine)

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/7"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 1 || brought.Errors[0].Why == nil ||
		!strings.Contains(brought.Errors[0].Why.Error(), "neither a file nor a folder") {
		t.Errorf("what stayed outside: %+v", brought.Errors)
	}
	if len(brought.Landed) != 0 {
		t.Errorf("what landed: %v", brought.Landed)
	}
}
