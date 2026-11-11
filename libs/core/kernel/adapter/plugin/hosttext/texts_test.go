package hosttext_test

import (
	"bytes"
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/textfake"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

const (
	mapsPath   = "library/maps.pdf"
	pdfReader  = "org.numen.readers-pdf"
	textOfMaps = "text-of-maps"
)

func getMapsContent() []byte { return []byte("%PDF maps of the coast") }

func getMapsTime() time.Time { return time.Date(2026, 5, 4, 10, 0, 0, 987654321, time.UTC) }

// openFailure fails to open every vault, and records the vaults it received open requests for.
type openFailure struct {
	opened *[]domain.VaultID
}

func (o openFailure) Open(v domain.Vault) (vaultport.VaultReader, error) {
	*o.opened = append(*o.opened, v.ID)
	return nil, errors.New("vault is gone")
}

// mapsVault has the maps PDF, and counts the times it is opened or read whole.
type mapsVault struct {
	opens int
}

func (m *mapsVault) Open(domain.Vault) (vaultport.VaultReader, error) {
	return mapsReader{vault: m}, nil
}

type mapsReader struct {
	vaultport.VaultReader
	vault *mapsVault
}

func (r mapsReader) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	if path != mapsPath {
		return domain.FileStat{}, fs.ErrNotExist
	}
	return domain.FileStat{Path: path, Kind: domain.KindFile, Size: int64(len(getMapsContent())), ModTime: getMapsTime(), FileID: 31}, nil
}

func (r mapsReader) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	if path != mapsPath {
		return nil, fs.ErrNotExist
	}
	r.vault.opens++
	return memoryFile{bytes.NewReader(getMapsContent())}, nil
}

func (r mapsReader) Read(_ context.Context, path string) ([]byte, error) {
	if path != mapsPath {
		return nil, fs.ErrNotExist
	}
	r.vault.opens++
	return getMapsContent(), nil
}

type memoryFile struct{ *bytes.Reader }

func (memoryFile) Close() error { return nil }

// mapsVersions is the one version of the maps and the row a scan wrote for them.
type mapsVersions struct {
	source      text.ContentHash
	hasRow      bool
	findFailure error
	saveFailure error
}

func (m *mapsVersions) OpenVersionReader(domain.Vault) (textport.VersionReader, error) { return m, nil }

func (m *mapsVersions) OpenVersionWriter(domain.Vault) (textport.VersionWriter, error) { return m, nil }

func (m *mapsVersions) OpenContentHashes(domain.Vault) (textport.ContentHashes, error) { return m, nil }

func (m *mapsVersions) GetVersions(_ context.Context, path string) ([]domain.TextVersion, error) {
	if path != mapsPath {
		return nil, nil
	}
	return []domain.TextVersion{{
		Path: mapsPath, PluginID: pdfReader, Key: "pages", Kind: "page-text", File: "pages.jsonl",
		InputHash: domain.HashContent(getMapsContent()), TextHash: textOfMaps, CreateTime: getMapsTime(),
	}}, nil
}

func (m *mapsVersions) SaveVersion(context.Context, domain.TextVersion) error { return nil }

func (m *mapsVersions) RemoveVersion(context.Context, string, domain.TextVersionID) error { return nil }

func (m *mapsVersions) RemovePluginVersions(context.Context, string) ([]domain.TextVersion, error) {
	return nil, nil
}

func (m *mapsVersions) FindContentHash(_ context.Context, path string) (text.ContentHash, bool, error) {
	if m.findFailure != nil {
		return text.ContentHash{}, false, m.findFailure
	}
	if path != mapsPath || !m.hasRow {
		return text.ContentHash{}, false, nil
	}
	return m.source, true, nil
}

func (m *mapsVersions) SaveContentHash(_ context.Context, record text.ContentHash) error {
	if m.saveFailure != nil {
		return m.saveFailure
	}
	if record.Path != mapsPath || !m.hasRow {
		return nil
	}
	isSameStat := m.source.Size == record.Size && m.source.ModTime.Equal(record.ModTime)
	if isSameStat || (m.source.Hash != "" && m.source.Hash == record.Hash) {
		m.source = record
	}
	return nil
}

// textFiles contains the text file of every version, unchanged since it was recorded.
type textFiles struct{}

func (textFiles) Open(domain.Vault) (textport.TextFiles, error) { return textFiles{}, nil }

func (textFiles) DeleteTextFile(context.Context, string, string) error { return nil }

func (textFiles) ReadTextFile(context.Context, string, string) (text.File, error) {
	return text.File{TextHash: textOfMaps}, nil
}

func (textFiles) WriteTextFile(context.Context, string, string, []domain.TextSegment) (string, error) {
	return textOfMaps, nil
}

type noPlugins struct{}

func (noPlugins) IsEnabled(string) bool { return false }

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func newTexts(t *testing.T, readers vaultport.VaultReaders, repo *mapsVersions) hosttext.Texts {
	t.Helper()
	versions := textfake.NewTextVersions(repo, textFiles{}, noPlugins{}, readers, fixedClock{now: getMapsTime().Add(time.Hour)})
	return hosttext.NewTexts(versions.Versions, versions.Texts)
}

func TestListVersionsReturnsTheReaderFailure(t *testing.T) {
	var opened []domain.VaultID
	texts := newTexts(t, openFailure{opened: &opened}, &mapsVersions{})
	if _, err := texts.ListVersions(t.Context(), domain.Vault{ID: "v1"}, "talk.mp3"); err == nil || errors.Is(err, text.ErrNoText) {
		t.Fatalf("got %v, want the reader's failure", err)
	}
}

func TestListVersionsReadsTheVaultItIsGiven(t *testing.T) {
	var opened []domain.VaultID
	texts := newTexts(t, openFailure{opened: &opened}, &mapsVersions{})
	if _, err := texts.ListVersions(t.Context(), domain.Vault{ID: "v2"}, "talk.mp3"); err == nil {
		t.Error("expected the reader's failure")
	}
	if len(opened) != 1 || opened[0] != "v2" {
		t.Errorf("opened %v, want v2", opened)
	}
}

// library_read lists the versions of a book on every call; an unchanged book is
// opened for its hash once.
func TestListingTheVersionsOfAnUnchangedPDFTwiceOpensItOnce(t *testing.T) {
	files := &mapsVault{}
	repo := &mapsVersions{hasRow: true, source: text.ContentHash{
		Path: mapsPath, Size: int64(len(getMapsContent())), ModTime: getMapsTime(),
	}}
	texts := newTexts(t, files, repo)
	vault := domain.Vault{ID: "v1", Name: "v1"}

	for i := range 2 {
		listing, err := texts.ListVersions(t.Context(), vault, mapsPath)
		if err != nil || len(listing.Versions) != 1 || listing.Live != listing.Versions[0].GetID() {
			t.Fatalf("listing %d is %+v (%v), want the one version live", i+1, listing, err)
		}
	}
	if files.opens != 1 {
		t.Fatalf("the PDF was opened or read %d times over two listings, want once", files.opens)
	}
}

func getScannedMaps() text.ContentHash {
	return text.ContentHash{Path: mapsPath, Size: int64(len(getMapsContent())), ModTime: getMapsTime()}
}

func checkLive(t *testing.T, listing plugin.TextListing, err error) {
	t.Helper()
	if err != nil || len(listing.Versions) != 1 || listing.Live != listing.Versions[0].GetID() {
		t.Fatalf("the listing is %+v (%v), want the one version live", listing, err)
	}
}

// The recorded hash is a cache: a hash that cannot be recorded still decides the live version.
func TestListVersionsListsLiveVersionWhenHashCannotBeRecorded(t *testing.T) {
	repo := &mapsVersions{hasRow: true, source: getScannedMaps(), saveFailure: errors.New("database is locked")}
	texts := newTexts(t, &mapsVault{}, repo)

	listing, err := texts.ListVersions(t.Context(), domain.Vault{ID: "v1", Name: "v1"}, mapsPath)
	checkLive(t, listing, err)
}

func TestListVersionsReturnsAFailureOfTheIndexAsItIs(t *testing.T) {
	failure := errors.New("database is locked")
	repo := &mapsVersions{hasRow: true, source: getScannedMaps(), findFailure: failure}
	texts := newTexts(t, &mapsVault{}, repo)

	_, err := texts.ListVersions(t.Context(), domain.Vault{ID: "v1", Name: "v1"}, mapsPath)
	if !errors.Is(err, failure) || errors.Is(err, text.ErrNoText) {
		t.Fatalf("got %v, want the index's failure, not ErrNoText", err)
	}
}

// dirReaders opens every vault on the folder, read by the filesystem reader with the options.
type dirReaders struct {
	root string
	opts vaultconfig.Options
}

func (d dirReaders) Open(domain.Vault) (vaultport.VaultReader, error) {
	return vaultreader.Open(d.root, d.opts)
}

// A file of a folder the vault ignores has no index row; its versions still have a live one.
func TestListVersionsOfAPDFInAnIgnoredFolderListsItsLiveVersion(t *testing.T) {
	root := vaultfixture.CopyVault(t)
	target := filepath.Join(root, filepath.FromSlash(mapsPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, getMapsContent(), 0o600); err != nil {
		t.Fatal(err)
	}
	texts := newTexts(t, dirReaders{root: root, opts: vaultconfig.Options{Ignore: []string{"library/"}}}, &mapsVersions{})

	listing, err := texts.ListVersions(t.Context(), domain.Vault{ID: "v1", Name: "v1"}, mapsPath)
	checkLive(t, listing, err)
}
