package recognition_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// paddingSize is a trailing run large enough that a copy of the file shows
// against the allocator's own movement.
const paddingSize = 64 << 20

const pdfTestdata = "../../internal/testsupport/testdata/pdf/"

// newDiskVault is a vault folder holding one file, and the vault over it.
func newDiskVault(t *testing.T, raw []byte) domain.Vault {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "scan.pdf"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return domain.Vault{ID: "disk", Path: dir}
}

func newDiskRecognise(onPage func()) (recognition.Recognise, func(domain.VaultID, string) string) {
	return recognition.NewMemoryRecognise(filesystem.VaultReaders{}, testsupport.Documents{}, onPage)
}

func TestRecognisingAFileOnDiskContainsNoCopyOfIt(t *testing.T) {
	tiny, err := os.ReadFile(pdfTestdata + "tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	padded := append(bytes.Clone(tiny), bytes.Repeat([]byte("%padding\n"), paddingSize/len("%padding\n"))...)
	v := newDiskVault(t, padded)
	size := uint64(len(padded))
	want := text.Fingerprint(padded)

	var peak uint64
	u, hashOf := newDiskRecognise(func() {
		runtime.GC()
		var during runtime.MemStats
		runtime.ReadMemStats(&during)
		peak = max(peak, during.HeapAlloc)
	})

	// The first recognition starts the pdfium worker, so the baseline holds it.
	warmVault := newDiskVault(t, tiny)
	warmUse, _ := newDiskRecognise(nil)
	if _, err := warmUse.Execute(t.Context(), warmVault, "scan.pdf"); err != nil {
		t.Fatal(err)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	res, err := u.Execute(t.Context(), v, "scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if res.Pages == 0 || res.Read != res.Pages {
		t.Fatalf("read %d of %d pages", res.Read, res.Pages)
	}
	if peak > before.HeapAlloc && peak-before.HeapAlloc > size/2 {
		t.Errorf("the recognition retained %d bytes of a %d byte file", peak-before.HeapAlloc, size)
	}
	if got := hashOf(v.ID, "scan.pdf"); got != want {
		t.Errorf("the source is named %s, want the hash of the bytes %s", got, want)
	}
}

func TestRecognisingAFileOnDiskRefusesWhatIsNotADocument(t *testing.T) {
	for name, raw := range map[string][]byte{
		"empty":   {},
		"not PDF": []byte("this is not a document"),
	} {
		t.Run(name, func(t *testing.T) {
			v := newDiskVault(t, raw)
			u, _ := newDiskRecognise(nil)
			_, err := u.Execute(t.Context(), v, "scan.pdf")
			if !errors.Is(err, port.ErrNotADocument) {
				t.Errorf("recognising it gave %v, want %v", err, port.ErrNotADocument)
			}
		})
	}
}

func TestRecognisingAMissingFileIsAnError(t *testing.T) {
	v := newDiskVault(t, []byte("x"))
	u, _ := newDiskRecognise(nil)
	if _, err := u.Execute(t.Context(), v, "gone.pdf"); err == nil {
		t.Error("a file that is not there was recognised")
	}
}
