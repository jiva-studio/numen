package vault_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// handedOver is a machine whose handles are not paths and which has no
// filesystem behind them: a phone hands a file over as a content URI, and
// nothing about one can be given to os.Open.
type handedOver struct {
	names   map[string]string
	bodies  map[string]string
	holding map[string][]string
}

func (h handedOver) GetName(handle string) string { return h.names[handle] }

func (h handedOver) GetImportedFile(_ context.Context, handle string) (source.ImportedFile, error) {
	one := source.ImportedFile{Name: h.names[handle], Handle: handle}
	if _, target := h.holding[handle]; target {
		one.IsFolder = true
		return one, nil
	}
	if _, targetVal := h.bodies[handle]; targetVal {
		one.IsFile = true
		return one, nil
	}
	return source.ImportedFile{}, io.ErrUnexpectedEOF
}

func (h handedOver) List(_ context.Context, handle string) ([]source.ImportedFile, error) {
	var out []source.ImportedFile
	for _, underVal := range h.holding[handle] {
		out = append(out, source.ImportedFile{Name: h.names[underVal], Handle: underVal})
	}
	return out, nil
}

func (h handedOver) Open(_ context.Context, handle string) (io.ReadCloser, error) {
	body, target := h.bodies[handle]
	if !target {
		return nil, io.ErrUnexpectedEOF
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

func (handedOver) HasHandle(string, string) bool { return false }

// User-supplied files are read through the port, so a machine that identifies
// its files by anything but paths can still import them. A use case that
// accesses the filesystem directly cannot be bound on a phone at all.
func TestFilesAreBroughtInFromAMachineWhoseHandlesAreNotPaths(t *testing.T) {
	t.Parallel()
	v := vaultfixture.NewVault(t, nil)
	handed := handedOver{
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
	bring := vaults.NewImport(vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, handed)

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 0 {
		t.Fatalf("what stayed outside: %v", brought.Errors)
	}
	if len(brought.Imported) != 3 {
		t.Fatalf("what landed: %v", brought.Imported)
	}
	if body := readArrived(t, v.Path, "scans/Cover.png"); body != "PNG" {
		t.Errorf("the picture arrived as %q", body)
	}
	if body := readArrived(t, v.Path, "scans/Kelvin.md"); body != "# Kelvin\n" {
		t.Errorf("the note arrived as %q", body)
	}
}
