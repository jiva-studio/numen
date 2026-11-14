package textversion_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion/textversiontest"
	"io"
	"io/fs"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

const atlasPath = "books/atlas.pdf"

func getAtlasTime() time.Time { return time.Date(2026, 3, 1, 9, 0, 0, 123456789, time.UTC) }

func getAtlasContent() []byte { return []byte("%PDF atlas of rivers, first edition") }

// statFile is one file of a statVault: its bytes and the stat the file system gives it.
type statFile struct {
	raw       []byte
	modTime   time.Time
	id        domain.FileID
	isIgnored bool
}

// statVault is a vault whose files have a modification time and a file id,
// and which counts the files opened and read whole.
type statVault struct {
	mu    sync.Mutex
	files map[string]*statFile
	opens int
	reads int
}

func (s *statVault) Open(domain.Vault) (vaultport.VaultReader, error) {
	return statReader{vault: s}, nil
}

func (s *statVault) change(path string, edit func(*statFile)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	edit(s.files[path])
}

func (s *statVault) getAccesses() (opens, reads int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opens, s.reads
}

type statReader struct {
	vaultport.VaultReader
	vault *statVault
}

func (r statReader) find(path string) (statFile, error) {
	r.vault.mu.Lock()
	defer r.vault.mu.Unlock()
	file, ok := r.vault.files[path]
	if !ok {
		return statFile{}, fs.ErrNotExist
	}
	return *file, nil
}

func (r statReader) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	file, err := r.find(path)
	if err != nil {
		return domain.FileStat{}, err
	}
	if file.isIgnored {
		return domain.FileStat{}, edit.ErrNotANote
	}
	return domain.FileStat{Path: path, Kind: domain.KindFile, Size: int64(len(file.raw)), ModTime: file.modTime, FileID: file.id}, nil
}

func (r statReader) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	file, err := r.find(path)
	if err != nil {
		return nil, err
	}
	r.vault.mu.Lock()
	r.vault.opens++
	r.vault.mu.Unlock()
	return memoryFile{bytes.NewReader(file.raw)}, nil
}

func (r statReader) Read(_ context.Context, path string) ([]byte, error) {
	file, err := r.find(path)
	if err != nil {
		return nil, err
	}
	r.vault.mu.Lock()
	r.vault.reads++
	r.vault.mu.Unlock()
	return file.raw, nil
}

// setClock is an instant that moves only when a test sets it.
type setClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *setClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *setClock) set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

// lookup is one vault containing the atlas, with the row a scan wrote for it, and
// the versions that look its hash up.
type lookup struct {
	vault    domain.Vault
	files    *statVault
	repo     *memoryVersions
	clock    *setClock
	versions *textversiontest.TextVersions
}

func newLookup(t *testing.T) *lookup {
	t.Helper()
	l := &lookup{
		vault: domain.Vault{ID: "v1", Name: "v1"},
		files: &statVault{files: map[string]*statFile{
			atlasPath: {raw: getAtlasContent(), modTime: getAtlasTime(), id: 4242},
		}},
		repo:  &memoryVersions{},
		clock: &setClock{now: getAtlasTime().Add(time.Minute)},
	}
	l.repo.addScannedSource(l.vault.ID, atlasPath, int64(len(getAtlasContent())), getAtlasTime())
	versions := textversiontest.NewTextVersions(l.repo, &memoryFiles{}, &plugins{}, l.files, l.clock)
	l.versions = versions
	return l
}

func (l *lookup) getHash(t *testing.T) string {
	t.Helper()
	hash, err := l.versions.Texts.GetContentHash(t.Context(), l.vault, atlasPath)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func (l *lookup) checkAccesses(t *testing.T, wantOpens int) {
	t.Helper()
	opens, reads := l.files.getAccesses()
	if opens != wantOpens || reads != 0 {
		t.Fatalf("the atlas was opened %d times and read whole %d times, want %d opens and no read", opens, reads, wantOpens)
	}
}

func TestASecondLookupOfAnUnchangedFileDoesNotReadIt(t *testing.T) {
	l := newLookup(t)
	want := domain.HashContent(getAtlasContent())

	if got := l.getHash(t); got != want {
		t.Fatalf("the first lookup gives %s, want %s", got, want)
	}
	l.checkAccesses(t, 1)
	if got := l.getHash(t); got != want {
		t.Fatalf("the second lookup gives %s, want %s", got, want)
	}
	l.checkAccesses(t, 1)
}

func TestALookupReadsAFileWhoseSizeChanged(t *testing.T) {
	l := newLookup(t)
	l.getHash(t)
	longer := []byte("%PDF atlas of rivers, first edition, with an index")
	l.files.change(atlasPath, func(f *statFile) { f.raw = longer })

	if got, want := l.getHash(t), domain.HashContent(longer); got != want {
		t.Fatalf("a file of another size gives %s, want %s", got, want)
	}
	l.checkAccesses(t, 2)
}

func TestALookupReadsAFileWhoseModificationTimeChanged(t *testing.T) {
	l := newLookup(t)
	l.getHash(t)
	l.files.change(atlasPath, func(f *statFile) { f.modTime = f.modTime.Add(5 * time.Second) })

	l.getHash(t)
	l.checkAccesses(t, 2)
}

func TestALookupReadsAFileReplacedAtTheSamePath(t *testing.T) {
	l := newLookup(t)
	l.getHash(t)
	l.files.change(atlasPath, func(f *statFile) { f.id = 4243 })

	l.getHash(t)
	l.checkAccesses(t, 2)
}

func TestARecordWithoutAHashTimeIsCheckedByContent(t *testing.T) {
	l := newLookup(t)
	l.repo.addSource(l.vault.ID, text.ContentHash{
		Path: atlasPath, Hash: domain.HashContent(getAtlasContent()), Size: int64(len(getAtlasContent())),
		ModTime: getAtlasTime(), FileID: 4242,
	})

	l.getHash(t)
	l.checkAccesses(t, 1)
}

// A file changed in the same clock tick as its record keeps its size and its
// modification time; only the content differs.
func TestAChangeWithinTheTickOfTheRecordIsFoundByContent(t *testing.T) {
	l := newLookup(t)
	l.clock.set(getAtlasTime())
	l.getHash(t)
	sameSize := []byte("%PDF atlas of rivers, fifth edition")
	if len(sameSize) != len(getAtlasContent()) {
		t.Fatal("the changed content must keep the size")
	}
	l.files.change(atlasPath, func(f *statFile) { f.raw = sameSize })

	if got, want := l.getHash(t), domain.HashContent(sameSize); got != want {
		t.Fatalf("a file changed in the tick of its record gives %s, want %s", got, want)
	}
	l.checkAccesses(t, 2)
}

func TestAfterARacyCheckTheNextLookupDoesNotReadTheFile(t *testing.T) {
	l := newLookup(t)
	l.clock.set(getAtlasTime())
	l.getHash(t)
	l.clock.set(getAtlasTime().Add(3 * time.Second))

	l.getHash(t)
	l.checkAccesses(t, 2)
	l.getHash(t)
	l.checkAccesses(t, 2)
}

// A file touched with its content unchanged has its new stat recorded, so the
// lookup after it does not read it (mypy's rule).
func TestATouchedFileWithTheSameContentHasItsNewStatRecorded(t *testing.T) {
	l := newLookup(t)
	l.getHash(t)
	touched := getAtlasTime().Add(10 * time.Second)
	l.files.change(atlasPath, func(f *statFile) { f.modTime = touched })
	l.clock.set(getAtlasTime().Add(2 * time.Minute))

	l.getHash(t)
	l.checkAccesses(t, 2)
	row, ok := l.repo.getSource(l.vault.ID, atlasPath)
	if !ok || !row.ModTime.Equal(touched) || row.Hash != domain.HashContent(getAtlasContent()) {
		t.Fatalf("the row is %+v (%v), want the hash of the atlas at %s", row, ok, touched)
	}
	l.getHash(t)
	l.checkAccesses(t, 2)
}

// A changed file gets its new hash, and its row keeps the stat it had, so the
// indexer still sees the change.
func TestAChangedFileKeepsTheRowStatAndGetsTheNewHash(t *testing.T) {
	l := newLookup(t)
	l.getHash(t)
	before, ok := l.repo.getSource(l.vault.ID, atlasPath)
	if !ok || before.Hash == "" {
		t.Fatalf("the first lookup left the row %+v (%v), want its hash recorded", before, ok)
	}
	second := []byte("%PDF atlas of rivers, second edition, revised")
	l.files.change(atlasPath, func(f *statFile) {
		f.raw = second
		f.modTime = f.modTime.Add(10 * time.Second)
	})
	l.clock.set(getAtlasTime().Add(2 * time.Minute))

	if got, want := l.getHash(t), domain.HashContent(second); got != want {
		t.Fatalf("the changed file gives %s, want %s", got, want)
	}
	after, _ := l.repo.getSource(l.vault.ID, atlasPath)
	if after.Hash != before.Hash || after.Size != before.Size || !after.ModTime.Equal(before.ModTime) ||
		after.FileID != before.FileID || !after.HashTime.Equal(before.HashTime) {
		t.Fatalf("the row became %+v, want it kept as %+v", after, before)
	}
}

func TestAFileWithNoRowIsHashedOnEveryLookupAndNothingIsSaved(t *testing.T) {
	l := newLookup(t)
	const loose = "books/loose.pdf"
	raw := []byte("%PDF loose leaf")
	l.files.files[loose] = &statFile{raw: raw, modTime: getAtlasTime(), id: 77}

	for range 2 {
		got, err := l.versions.Texts.GetContentHash(t.Context(), l.vault, loose)
		if err != nil || got != domain.HashContent(raw) {
			t.Fatalf("a file with no row gives %s (%v), want %s", got, err, domain.HashContent(raw))
		}
	}
	if opens, reads := l.files.getAccesses(); opens != 2 || reads != 0 {
		t.Fatalf("a file with no row was opened %d times and read whole %d times, want 2 opens", opens, reads)
	}
	if row, ok := l.repo.getSource(l.vault.ID, loose); ok {
		t.Fatalf("a row %+v was saved for a file the scan did not record", row)
	}
}

func TestALookupOfAMissingFileIsNotFound(t *testing.T) {
	l := newLookup(t)
	if _, err := l.versions.Texts.GetContentHash(t.Context(), l.vault, "books/gone.pdf"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a file that is not there gives %v, want fs.ErrNotExist", err)
	}
}

const gigabyte = 1 << 30

// generatedVault contains one file of a gigabyte whose bytes are made as it is
// read. Reading it whole fails the test.
type generatedVault struct {
	t     *testing.T
	path  string
	opens int
}

func (g *generatedVault) Open(domain.Vault) (vaultport.VaultReader, error) {
	return generatedReader{vault: g}, nil
}

type generatedReader struct {
	vaultport.VaultReader
	vault *generatedVault
}

func (r generatedReader) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	if path != r.vault.path {
		return domain.FileStat{}, fs.ErrNotExist
	}
	return domain.FileStat{Path: path, Kind: domain.KindFile, Size: gigabyte, ModTime: getAtlasTime(), FileID: 9001}, nil
}

func (r generatedReader) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	if path != r.vault.path {
		return nil, fs.ErrNotExist
	}
	r.vault.opens++
	return newGeneratedFile(gigabyte), nil
}

func (r generatedReader) Read(context.Context, string) ([]byte, error) {
	r.vault.t.Error("the lookup read a gigabyte file whole")
	return nil, errors.New("a gigabyte file is not read whole")
}

// generatedFile is size bytes of a repeating block, made as they are read.
type generatedFile struct {
	size   int64
	offset int64
	block  []byte
}

func newGeneratedFile(size int64) *generatedFile {
	return &generatedFile{size: size, block: getBlock()}
}

func getBlock() []byte {
	block := make([]byte, 64<<10)
	for i := range block {
		block[i] = byte(i*7 + i/251)
	}
	return block
}

func (f *generatedFile) Read(p []byte) (int, error) {
	if f.offset >= f.size {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && f.offset < f.size {
		position := f.offset % int64(len(f.block))
		k := copy(p[n:], f.block[position:min(int64(len(f.block)), position+f.size-f.offset)])
		n += k
		f.offset += int64(k)
	}
	return n, nil
}

func (f *generatedFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += f.offset
	case io.SeekEnd:
		offset += f.size
	}
	if offset < 0 {
		return 0, errors.New("seek before the start")
	}
	f.offset = offset
	return offset, nil
}

func (*generatedFile) Close() error { return nil }

func TestAOneGigabyteFileIsHashedFromAStreamWithBoundedMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("hashes a gigabyte twice")
	}
	const path = "recordings/lecture.wav"
	files := &generatedVault{t: t, path: path}
	repo := &memoryVersions{}
	repo.addScannedSource("v1", path, gigabyte, getAtlasTime())
	versions := textversiontest.NewTextVersions(repo, &memoryFiles{}, &plugins{}, files, &setClock{now: getAtlasTime().Add(time.Minute)})
	sum := sha256.New()
	if _, err := io.Copy(sum, newGeneratedFile(gigabyte)); err != nil {
		t.Fatal(err)
	}
	want := hex.EncodeToString(sum.Sum(nil))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	got, err := versions.Texts.GetContentHash(t.Context(), domain.Vault{ID: "v1", Name: "v1"}, path)
	runtime.ReadMemStats(&after)

	if err != nil || got != want {
		t.Fatalf("a gigabyte file gives %s (%v), want %s", got, err, want)
	}
	if files.opens != 1 {
		t.Errorf("the file was opened %d times, want once", files.opens)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 64<<20 {
		t.Errorf("hashing a gigabyte allocated %d MiB, want under 64 MiB", allocated>>20)
	}
}

// A file system with a coarse tick keeps the stamp of a file changed half a
// second after it was hashed, a second after the stamp; only the content differs.
func TestChangeWithinRacyWindowIsDetectedByContent(t *testing.T) {
	l := newLookup(t)
	l.clock.set(getAtlasTime().Add(time.Second))
	l.getHash(t)
	sameSize := []byte("%PDF atlas of rivers, sixth edition")
	if len(sameSize) != len(getAtlasContent()) {
		t.Fatal("the changed content must keep the size")
	}
	l.files.change(atlasPath, func(f *statFile) { f.raw = sameSize })
	l.clock.set(getAtlasTime().Add(time.Second + 500*time.Millisecond))

	if got, want := l.getHash(t), domain.HashContent(sameSize); got != want {
		t.Fatalf("a file changed half a second after its hash gives %s, want %s", got, want)
	}
	l.checkAccesses(t, 2)
}

// A file the vault ignores is hashed from a stream, as it was read before, and no record is kept.
func TestAFileTheVaultIgnoresIsHashedFromAStreamAndNotRecorded(t *testing.T) {
	l := newLookup(t)
	l.files.change(atlasPath, func(f *statFile) { f.isIgnored = true })
	before, _ := l.repo.getSource(l.vault.ID, atlasPath)

	if got, want := l.getHash(t), domain.HashContent(getAtlasContent()); got != want {
		t.Fatalf("an ignored file gives %s, want %s", got, want)
	}
	l.checkAccesses(t, 1)
	if after, _ := l.repo.getSource(l.vault.ID, atlasPath); after != before {
		t.Fatalf("the row of an ignored file became %+v, want it kept as %+v", after, before)
	}
}

// A file whose modification time is ahead of the clock cannot be changed with
// its stat kept: a later write stamps the time of the write.
func TestFutureMtimeFileIsOpenedOnceOverSixLookups(t *testing.T) {
	l := newLookup(t)
	l.clock.set(getAtlasTime().Add(-time.Hour))
	want := domain.HashContent(getAtlasContent())

	for i := range 6 {
		if got := l.getHash(t); got != want {
			t.Fatalf("lookup %d gives %s, want %s", i+1, got, want)
		}
	}
	l.checkAccesses(t, 1)
}

// A stamp a second after the hash time is within the tick of the record, so
// a change that keeps the size and the stamp is found by content.
func TestAChangeToAFileStampedASecondAfterItsHashIsFoundByContent(t *testing.T) {
	l := newLookup(t)
	l.clock.set(getAtlasTime().Add(-time.Second))
	l.getHash(t)
	sameSize := []byte("%PDF atlas of rivers, ninth edition")
	if len(sameSize) != len(getAtlasContent()) {
		t.Fatal("the changed content must keep the size")
	}
	l.files.change(atlasPath, func(f *statFile) { f.raw = sameSize })

	if got, want := l.getHash(t), domain.HashContent(sameSize); got != want {
		t.Fatalf("a file stamped a second after its hash gives %s, want %s", got, want)
	}
	l.checkAccesses(t, 2)
}
