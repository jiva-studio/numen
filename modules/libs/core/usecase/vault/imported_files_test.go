package vault_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// handedOver is a machine whose handles are not paths and which has no
// filesystem behind them: a phone hands a file over as a content URI, and
// nothing about one can be given to os.Open.
type handedOver struct {
	names   map[string]string
	bodies  map[string]string
	holding map[string][]string
}

func (h handedOver) Named(handle string) string { return h.names[handle] }

func (h handedOver) Stat(_ context.Context, handle string) (port.ImportedFile, error) {
	one := port.ImportedFile{Name: h.names[handle], Handle: handle}
	if _, held := h.holding[handle]; held {
		one.Folder = true
		return one, nil
	}
	if _, held := h.bodies[handle]; held {
		one.File = true
		return one, nil
	}
	return port.ImportedFile{}, io.ErrUnexpectedEOF
}

func (h handedOver) List(_ context.Context, handle string) ([]port.ImportedFile, error) {
	var out []port.ImportedFile
	for _, under := range h.holding[handle] {
		out = append(out, port.ImportedFile{Name: h.names[under], Handle: under})
	}
	return out, nil
}

func (h handedOver) Open(_ context.Context, handle string) (io.ReadCloser, error) {
	body, held := h.bodies[handle]
	if !held {
		return nil, io.ErrUnexpectedEOF
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

func (handedOver) Holds(string, string) bool { return false }

// What a person hands over is read through the port, so a machine that names
// its files anything but paths can bring them in. A use case reaching the
// filesystem itself cannot be bound on a phone at all.
func TestFilesAreBroughtInFromAMachineWhoseHandlesAreNotPaths(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, nil)
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
	bring := vaults.Import{Writers: filesystem.VaultWriters{}, Files: handed}

	brought, err := bring.Execute(t.Context(), v, "", []string{"content://held/1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brought.Errors) != 0 {
		t.Fatalf("what stayed outside: %v", brought.Errors)
	}
	if len(brought.Landed) != 3 {
		t.Fatalf("what landed: %v", brought.Landed)
	}
	if body := readArrived(t, v.Path, "scans/Cover.png"); body != "PNG" {
		t.Errorf("the picture arrived as %q", body)
	}
	if body := readArrived(t, v.Path, "scans/Kelvin.md"); body != "# Kelvin\n" {
		t.Errorf("the note arrived as %q", body)
	}
}
