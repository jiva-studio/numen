package hostvault

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

const syncer = "md.numen.syncer"

func readDelta(t *testing.T, v domain.Vault) []byte {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(v.Path, "Delta.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func fileOpsVault(t *testing.T, files map[string]string) (*vault.Access, domain.Vault, *atomic.Int32) {
	t.Helper()
	v := vaulttest.NewVault(t, files)
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	stat := file.NewHashFile(vaultreader.Readers{}, writers)
	var indexed atomic.Int32
	index := func(_ context.Context, _ domain.Vault, paths []string) error {
		indexed.Add(int32(len(paths)))
		return nil
	}
	scan := file.NewScanFiles(index)
	count := file.NewCountFiles(vaultreader.Readers{})
	stores := derivedfile.PluginStores{}
	return NewAccess(
		Session{GetCurrent: func() (domain.Vault, bool) { return v, true }},
		Files{Stat: stat, Bytes: file.NewReadBytes(vaultreader.Readers{}, stat)},
		Writer{
			Index:  scan,
			Place:  file.NewPlaceStaged(stat, scan, writers),
			Move:   note.NewMoveChecked(note.MoveFile{}, count, stat),
			Remove: note.NewRemoveChecked(note.RemoveFile{}, count, stat),
			StoreReaders: func(v domain.Vault, pluginID string) (vault.StoreReader, error) {
				return stores.Open(v, pluginID)
			},
			StoreWriters: func(v domain.Vault, pluginID string) (vault.StoreWriter, error) {
				return stores.OpenWriter(v, pluginID)
			},
		},
		Lookups{}, Arranger{}, Stores{},
	), v, &indexed
}

func stage(t *testing.T, v domain.Vault, name, content string) {
	t.Helper()
	stores := derivedfile.PluginStores{}
	writer, err := stores.OpenWriter(v, syncer)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Write(t.Context(), name, []byte(content)); err != nil {
		t.Fatal(err)
	}
}

func isStaged(t *testing.T, v domain.Vault, name string) bool {
	t.Helper()
	reader, err := derivedfile.PluginStores{}.Open(v, syncer)
	if err != nil {
		t.Fatal(err)
	}
	_, err = reader.Read(t.Context(), name)
	return err == nil
}

func TestReadFileRangeReturnsTheBytesOfTheRangeAndTheStatWithTheHashOfTheWholeFile(t *testing.T) {
	access, _, _ := fileOpsVault(t, map[string]string{"Maps/Delta.pdf": "0123456789"})

	content, stat, err := access.Ranges.ReadFileRange(t.Context(), "Maps/Delta.pdf", 3, 4)

	if err != nil || string(content) != "3456" {
		t.Fatalf("the range was %q (%v), want 3456", content, err)
	}
	if stat.Size != 10 || stat.Hash != domain.HashContent([]byte("0123456789")) || stat.Path != "Maps/Delta.pdf" {
		t.Errorf("the stat was %+v, want the size and the hash of the whole file", stat)
	}
}

func TestReadFileRangeReturnsTheBytesThatAreThereForARangePastTheEndAndNoneForAStartAtTheEnd(t *testing.T) {
	access, _, _ := fileOpsVault(t, map[string]string{"a.bin": "0123456789"})

	past, _, err := access.Ranges.ReadFileRange(t.Context(), "a.bin", 8, 100)
	if err != nil || string(past) != "89" {
		t.Errorf("a range past the end was %q (%v), want 89", past, err)
	}
	for _, start := range []int64{10, 11, 1 << 40} {
		got, stat, err := access.Ranges.ReadFileRange(t.Context(), "a.bin", start, 4)
		if err != nil || len(got) != 0 || stat.Size != 10 {
			t.Errorf("a start at %d was %q with size %d (%v), want no bytes and the stat", start, got, stat.Size, err)
		}
	}
}

func TestReadFileRangeRefusesAFolderAPathOutsideTheVaultAndARangeOutsideTheLimits(t *testing.T) {
	access, _, _ := fileOpsVault(t, map[string]string{"Maps/Delta.pdf": "x"})
	ctx := t.Context()

	if _, _, err := access.Ranges.ReadFileRange(ctx, "Maps", 0, 1); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a folder was read with %v, want fs.ErrNotExist", err)
	}
	if _, _, err := access.Ranges.ReadFileRange(ctx, "gone.bin", 0, 1); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a missing file was read with %v, want fs.ErrNotExist", err)
	}
	if _, _, err := access.Ranges.ReadFileRange(ctx, "../outside.bin", 0, 1); err == nil {
		t.Error("a path outside the vault was read")
	}
	for _, one := range []struct{ start, length int64 }{{0, 0}, {0, -1}, {-1, 1}, {0, 4<<20 + 1}} {
		if _, _, err := access.Ranges.ReadFileRange(ctx, "Maps/Delta.pdf", one.start, one.length); !errors.Is(err, plugin.ErrInvalidRange) {
			t.Errorf("a read of %d bytes from %d was %v, want ErrInvalidRange", one.length, one.start, err)
		}
	}
	if got, _, err := access.Ranges.ReadFileRange(ctx, "Maps/Delta.pdf", 0, 4<<20); err != nil || string(got) != "x" {
		t.Errorf("a read of the most bytes was %q (%v)", got, err)
	}
}

func TestReadFileRangeHashesAgainWhenTheFileChanged(t *testing.T) {
	access, v, _ := fileOpsVault(t, map[string]string{"a.bin": "first"})
	_, before, err := access.Ranges.ReadFileRange(t.Context(), "a.bin", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Path, "a.bin"), []byte("second one"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, after, err := access.Ranges.ReadFileRange(t.Context(), "a.bin", 0, 1)

	if err != nil || before.Hash != domain.HashContent([]byte("first")) || after.Hash != domain.HashContent([]byte("second one")) {
		t.Errorf("the hashes were %q then %q (%v), want those of the content each time", before.Hash, after.Hash, err)
	}
}

func TestPlaceFileMakesTheStagedFileTheFileOfThePathAndIndexesItAndRemovesTheStagedFile(t *testing.T) {
	access, v, indexed := fileOpsVault(t, nil)
	stage(t, v, "ab.part", "the whole file")

	stat, err := access.Ops.PlaceFile(t.Context(), syncer, "ab.part", "Scans/2026/Delta.pdf", domain.HashContent([]byte("the whole file")), nil)

	if err != nil {
		t.Fatal(err)
	}
	got, readErr := os.ReadFile(filepath.Join(v.Path, "Scans", "2026", "Delta.pdf"))
	if readErr != nil || string(got) != "the whole file" {
		t.Errorf("the placed file holds %q (%v)", got, readErr)
	}
	if stat.Path != "Scans/2026/Delta.pdf" || stat.Hash != domain.HashContent([]byte("the whole file")) || stat.Size != 14 {
		t.Errorf("the stat was %+v", stat)
	}
	if indexed.Load() != 1 {
		t.Errorf("%d paths were indexed, want 1", indexed.Load())
	}
	if isStaged(t, v, "ab.part") {
		t.Error("the staged file is still in the store after the placement")
	}
}

func TestPlaceFileOfAnotherHashPlacesNothingAndKeepsTheStagedFile(t *testing.T) {
	access, v, indexed := fileOpsVault(t, nil)
	stage(t, v, "ab.part", "damaged")

	_, err := access.Ops.PlaceFile(t.Context(), syncer, "ab.part", "Delta.pdf", domain.HashContent([]byte("whole")), nil)

	if !errors.Is(err, plugin.ErrHashMismatch) {
		t.Fatalf("the placement was %v, want ErrHashMismatch", err)
	}
	if _, statErr := os.Stat(filepath.Join(v.Path, "Delta.pdf")); !errors.Is(statErr, fs.ErrNotExist) || indexed.Load() != 0 || !isStaged(t, v, "ab.part") {
		t.Errorf("a refused placement left the file %v, %d indexed, staged %v", statErr, indexed.Load(), isStaged(t, v, "ab.part"))
	}
}

func TestPlaceFileWithoutABaselineRefusesAPathThatIsTaken(t *testing.T) {
	access, v, _ := fileOpsVault(t, map[string]string{"Delta.pdf": "mine"})
	stage(t, v, "ab.part", "theirs")

	_, err := access.Ops.PlaceFile(t.Context(), syncer, "ab.part", "Delta.pdf", domain.HashContent([]byte("theirs")), nil)

	got := readDelta(t, v)
	if !errors.Is(err, vaultdomain.ErrAlreadyExists) || string(got) != "mine" {
		t.Errorf("the placement over a taken path was %v and the file holds %q, want ErrAlreadyExists and mine", err, got)
	}
}

func TestPlaceFileWithABaselineReplacesTheFileItNamesAndRefusesOneThatChanged(t *testing.T) {
	access, v, _ := fileOpsVault(t, map[string]string{"Delta.pdf": "old"})
	_, seen, err := access.Ranges.ReadFileRange(t.Context(), "Delta.pdf", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	stage(t, v, "ab.part", "new content")
	hash := domain.HashContent([]byte("new content"))
	if err := os.WriteFile(filepath.Join(v.Path, "Delta.pdf"), []byte("edited here"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = access.Ops.PlaceFile(t.Context(), syncer, "ab.part", "Delta.pdf", hash, &seen)
	if !errors.Is(err, edit.ErrStale) {
		t.Fatalf("the placement over a changed file was %v, want ErrStale", err)
	}
	if got := readDelta(t, v); string(got) != "edited here" {
		t.Errorf("a refused placement left %q", got)
	}

	_, now, err := access.Ranges.ReadFileRange(t.Context(), "Delta.pdf", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := access.Ops.PlaceFile(t.Context(), syncer, "ab.part", "Delta.pdf", hash, &now); err != nil {
		t.Fatal(err)
	}
	if got := readDelta(t, v); string(got) != "new content" {
		t.Errorf("the replaced file holds %q", got)
	}
}

func TestPlaceFileWithABaselineOfAFileThatIsMissingIsStale(t *testing.T) {
	access, v, _ := fileOpsVault(t, nil)
	stage(t, v, "ab.part", "x")
	baseline := domain.FileStat{Path: "Gone.pdf", Hash: "h", Size: 1, ModTime: time.Unix(1, 0)}

	_, err := access.Ops.PlaceFile(t.Context(), syncer, "ab.part", "Gone.pdf", domain.HashContent([]byte("x")), &baseline)

	if !errors.Is(err, edit.ErrStale) {
		t.Errorf("the placement over a file that is gone was %v, want ErrStale", err)
	}
}

func TestPlaceFileOfAFileThatIsNotStagedOrOutsideTheStoreFolderOfThePluginIsRefused(t *testing.T) {
	access, v, _ := fileOpsVault(t, map[string]string{"Secret.txt": "s"})
	stage(t, v, "ab.part", "x")
	hash := domain.HashContent([]byte("x"))

	if _, err := access.Ops.PlaceFile(t.Context(), syncer, "none.part", "A.bin", hash, nil); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not staged was %v, want fs.ErrNotExist", err)
	}
	for _, name := range []string{"../../Secret.txt", "sub/ab.part", ".hidden"} {
		if _, err := access.Ops.PlaceFile(t.Context(), syncer, name, "B.bin", hash, nil); err == nil {
			t.Errorf("the staged name %q was placed", name)
		}
	}
	if _, err := access.Ops.PlaceFile(t.Context(), "md.numen.other", "ab.part", "C.bin", hash, nil); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file staged by another plugin was placed: %v", err)
	}
}

func TestMoveFileAndRemoveFileRefuseAMissingPathAndAChangedFileBeforeTheyChangeAnything(t *testing.T) {
	access, v, _ := fileOpsVault(t, map[string]string{"Maps/Delta.pdf": "now", "Maps/Rhine.pdf": "x"})
	stale := domain.FileStat{Path: "Maps/Delta.pdf", Hash: domain.HashContent([]byte("then")), Size: 4}

	if _, err := access.Ops.MoveFile(t.Context(), "Gone.pdf", "B.pdf", nil); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a move of a missing path was %v, want fs.ErrNotExist", err)
	}
	if _, err := access.Ops.RemoveFile(t.Context(), "Gone", nil); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a removal of a missing path was %v, want fs.ErrNotExist", err)
	}
	if _, err := access.Ops.MoveFile(t.Context(), "Maps/Delta.pdf", "B.pdf", &stale); !errors.Is(err, edit.ErrStale) {
		t.Errorf("a move of a changed file was %v, want ErrStale", err)
	}
	if _, err := access.Ops.RemoveFile(t.Context(), "Maps/Delta.pdf", &stale); !errors.Is(err, edit.ErrStale) {
		t.Errorf("a removal of a changed file was %v, want ErrStale", err)
	}
	for _, name := range []string{"Maps/Delta.pdf", "Maps/Rhine.pdf"} {
		if _, err := os.Stat(filepath.Join(v.Path, name)); err != nil {
			t.Errorf("a refused call changed %s: %v", name, err)
		}
	}
}

func TestPlaceFileNeverShowsAReaderAFileThatIsNotWhole(t *testing.T) {
	access, v, _ := fileOpsVault(t, nil)
	content := strings.Repeat("0123456789abcdef", 1<<19)
	stage(t, v, "ab.part", content)
	target := filepath.Join(v.Path, "Delta.pdf")
	var partial atomic.Int64
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			default:
			}
			if got, err := os.ReadFile(target); err == nil && string(got) != content {
				partial.Add(1)
			}
		}
	}()

	_, err := access.Ops.PlaceFile(t.Context(), syncer, "ab.part", "Delta.pdf", domain.HashContent([]byte(content)), nil)
	close(done)
	<-stopped

	if err != nil {
		t.Fatal(err)
	}
	if got := partial.Load(); got != 0 {
		t.Errorf("a reader saw a file that was not whole %d times while it was placed", got)
	}
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != content {
		t.Errorf("the placed file holds %d bytes (%v), want the %d bytes of the staged file", len(got), readErr, len(content))
	}
}
