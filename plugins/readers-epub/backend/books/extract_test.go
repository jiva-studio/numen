package books

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/testsupport"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/vault"
)

func writeBook(t *testing.T) (files *hosttest.Vault, name string, size, mtime int64) {
	t.Helper()
	name = "pictured.epub"
	files = hosttest.OnDisk(t, "vault", map[string]string{name: string(testsupport.CreateImageEpub(t))})
	info, err := files.StatFile(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	return files, name, info.Size, info.ModTime.UnixNano()
}

func readEntry(t *testing.T, files *hosttest.Vault, path, entry string, baseline *api.FileStat) (api.ReadBookEntryResponse, error) {
	t.Helper()
	books := NewReader(func(id domain.VaultID) host.FileReader { return files.With(id) })
	t.Cleanup(books.Close)
	return books.ReadBookEntry(t.Context(),
		api.ReadBookEntryRequest{Path: path, Entry: entry, Baseline: baseline})
}

func TestReadBookEntryAnswersTheBytesOfTheEntry(t *testing.T) {
	files, name, size, mtime := writeBook(t)
	resp, err := readEntry(t, files, name, "EPUB/cover.png", &api.FileStat{Path: name, Size: size, Mtime: mtime})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resp.Body, testsupport.GetPNGBytes()) {
		t.Errorf("the entry is %q, want the picture's bytes", resp.Body)
	}
}

func TestReadBookEntryAnswersWithoutABaseline(t *testing.T) {
	files, name, _, _ := writeBook(t)
	if _, err := readEntry(t, files, name, "EPUB/cover.png", nil); err != nil {
		t.Fatal(err)
	}
}

func TestReadBookEntry_ErrorsOnChangedFile(t *testing.T) {
	files, name, size, mtime := writeBook(t)
	cases := map[string]*api.FileStat{
		"another size":  {Path: name, Size: size + 1, Mtime: mtime},
		"another stamp": {Path: name, Size: size, Mtime: mtime + 1},
	}
	for label, baseline := range cases {
		t.Run(label, func(t *testing.T) {
			_, err := readEntry(t, files, name, "EPUB/cover.png", baseline)
			if codeOf(err) != server.CodeFailedPrecondition {
				t.Fatalf("code = %v (%v), want failed precondition", codeOf(err), err)
			}
		})
	}
}

func TestReadBookEntry_ErrorsOnMissingArchiveEntry(t *testing.T) {
	files, name, _, _ := writeBook(t)
	_, err := readEntry(t, files, name, "EPUB/missing.png", nil)
	if codeOf(err) != server.CodeNotFound {
		t.Fatalf("code = %v (%v), want not found", codeOf(err), err)
	}
}

func TestReadBookEntry_ErrorsOnPathOutsideVault(t *testing.T) {
	files, _, _, _ := writeBook(t)
	outside := filepath.Join(t.TempDir(), "outside.epub")
	if err := os.WriteFile(outside, testsupport.CreateImageEpub(t), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readEntry(t, files, outside, "EPUB/cover.png", nil)
	if codeOf(err) != server.CodePermissionDenied {
		t.Fatalf("code = %v (%v), want permission denied", codeOf(err), err)
	}
	if !strings.Contains(err.Error(), vault.ErrOutsideVault.Error()) {
		t.Fatalf("err = %v, want the outside-vault message", err)
	}
}

func TestReadBookEntry_ErrorsOnInvalidBook(t *testing.T) {
	files, _, _, _ := writeBook(t)
	files.File("junk.epub", "junk")
	_, err := readEntry(t, files, "junk.epub", "EPUB/cover.png", nil)
	if codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", codeOf(err), err)
	}
}

func TestReadBookEntry_ErrorsOnEmptyPath(t *testing.T) {
	files, _, _, _ := writeBook(t)
	_, err := readEntry(t, files, "", "EPUB/cover.png", nil)
	if codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", codeOf(err), err)
	}
}

func TestReadBookEntry_ErrorsOnEmptyEntry(t *testing.T) {
	files, name, _, _ := writeBook(t)
	_, err := readEntry(t, files, name, "", nil)
	if codeOf(err) != server.CodeInvalidArgument {
		t.Fatalf("code = %v (%v), want invalid argument", codeOf(err), err)
	}
}

func TestReadBookEntry_ErrorsOnMissingBook(t *testing.T) {
	files, _, _, _ := writeBook(t)
	subDir := filepath.Join(files.Vault().Path, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"nonexistent.epub", "sub"} {
		t.Run(target, func(t *testing.T) {
			_, err := readEntry(t, files, target, "EPUB/cover.png", nil)
			if codeOf(err) != server.CodeNotFound {
				t.Fatalf("code = %v (%v), want not found", codeOf(err), err)
			}
		})
	}
}

func TestReadBookEntryAnswersWithZeroMtimeBaseline(t *testing.T) {
	files, name, size, _ := writeBook(t)
	if _, err := readEntry(t, files, name, "EPUB/cover.png", &api.FileStat{Path: name, Size: size, Mtime: 0}); err != nil {
		t.Fatalf("expected success with Mtime == 0, got %v", err)
	}
}
