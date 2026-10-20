package file_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// openFileRead is a vault on disk and the read that works on it.
func openFileRead(t *testing.T, files map[string]string) (file.Read, domain.Vault) {
	t.Helper()
	return file.Read{Readers: filesystem.VaultReaders{}}, testsupport.NewVault(t, files)
}

func read(t *testing.T, u file.Read, v domain.Vault, path string, start, length int) file.ReadResult {
	t.Helper()
	contents, err := u.Execute(t.Context(), v, path, start, length)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return contents
}

// The tools that read a note read a note. A lecture somebody transcribed into a
// file of another kind is reached by its path and by nothing else.
func TestAFileTheVaultDoesNotHoldAsANoteIsRead(t *testing.T) {
	u, v := openFileRead(t, map[string]string{
		"lectures/kinetics.txt": "The first law, as spoken.\n",
	})

	got := read(t, u, v, "lectures/kinetics.txt", 0, 0)
	if got.Outcome != file.Ok {
		t.Fatalf("want ok, got %q", got.Outcome)
	}
	if got.Text != "The first law, as spoken.\n" {
		t.Errorf("the file came back as %q", got.Text)
	}
	if got.Whole != len(got.Text) || got.Start != 0 || got.Length != len(got.Text) {
		t.Errorf("the range was described as %+v", got)
	}
}

// A note is a file, and its bytes are what a read of a file hands over: the
// frontmatter is there, because that is what the file holds.
func TestANoteIsReadAsTheFileItIs(t *testing.T) {
	raw := "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n# Entropy\n"
	u, v := openFileRead(t, map[string]string{"Entropy.md": raw})

	if got := read(t, u, v, "Entropy.md", 0, 0); got.Text != raw {
		t.Errorf("the note came back as %q", got.Text)
	}
}

// A transcript is long, and an agent reads it a range at a time: what the answer
// says about the range is what the next call is asked with.
func TestALongFileIsReadARangeAtATime(t *testing.T) {
	whole := strings.Repeat("one line of what was said\n", 400)
	u, v := openFileRead(t, map[string]string{"lecture.txt": whole})

	first := read(t, u, v, "lecture.txt", 0, 100)
	if first.Text != whole[:100] || first.Start != 0 || first.Length != 100 {
		t.Fatalf("the first range was %+v", first)
	}
	if first.Whole != len(whole) {
		t.Errorf("the file is %d bytes and was said to be %d", len(whole), first.Whole)
	}

	next := read(t, u, v, "lecture.txt", first.Start+first.Length, 100)
	if next.Text != whole[100:200] {
		t.Errorf("reading on gave %q", next.Text)
	}
}

// A file asked for whole comes back cut to what one call carries, and says how
// much more there is.
func TestAFileLongerThanOneCallCarriesComesBackCut(t *testing.T) {
	whole := strings.Repeat("a", file.MostRead+500)
	u, v := openFileRead(t, map[string]string{"lecture.txt": whole})

	got := read(t, u, v, "lecture.txt", 0, 0)
	if got.Length != file.MostRead {
		t.Errorf("one call carried %d bytes", got.Length)
	}
	if got.Whole != len(whole) {
		t.Errorf("the file is %d bytes and was said to be %d", len(whole), got.Whole)
	}
}

// A range is held within the file, so an offset past the end is an empty range and
// not an error.
func TestARangePastTheEndOfTheFileIsEmpty(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lecture.txt": "short"})

	got := read(t, u, v, "lecture.txt", 900, 100)
	if got.Outcome != file.Ok || got.Text != "" || got.Length != 0 {
		t.Errorf("a range past the end gave %+v", got)
	}
	if got.Whole != 5 {
		t.Errorf("the file was said to be %d bytes", got.Whole)
	}
}

// A range is asked for in bytes and comes back as text, so a character the cut
// falls inside belongs to neither range.
func TestARangeIsCutAtWholeCharacters(t *testing.T) {
	// "śāstra" is two two-byte characters and four one-byte ones, and the
	// emoji that follows is four bytes.
	whole := "śāstra 🪔 and śloka"
	u, v := openFileRead(t, map[string]string{"lecture.txt": whole})

	// Opening one byte into the first character.
	opened := read(t, u, v, "lecture.txt", 1, 3)
	if opened.Text != "ā" || opened.Start != 2 || opened.Length != 2 {
		t.Errorf("a range opening inside a character came back as %+v", opened)
	}

	// Closing one byte into the second.
	closed := read(t, u, v, "lecture.txt", 0, 3)
	if closed.Text != "ś" || closed.Start != 0 || closed.Length != 2 {
		t.Errorf("a range closing inside a character came back as %+v", closed)
	}

	// The four-byte character stands at byte 9, and is closed inside at each of
	// the three places it can be.
	for _, length := range []int{10, 11, 12} {
		got := read(t, u, v, "lecture.txt", 0, length)
		if got.Text != "śāstra " {
			t.Errorf("a range of %d bytes came back as %q", length, got.Text)
		}
	}
	if got := read(t, u, v, "lecture.txt", 0, 13); got.Text != "śāstra 🪔" {
		t.Errorf("a range holding the whole character came back as %q", got.Text)
	}
}

// A range reaching the end of the file is not cut, so a file whose last character
// is cut short is named for what it is.
func TestARangeReachingTheEndOfTheFileIsNotCut(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lecture.txt": "said \xc5"})

	got := read(t, u, v, "lecture.txt", 0, 0)
	if got.Outcome != file.NotText {
		t.Errorf("want not text, got %+v", got)
	}
}

// A range goes out in a string field, so bytes that are not text are named as
// such and not carried.
func TestAFileThatIsNotTextIsSaidToBeNotText(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"scan.txt": "words \xff\xfe more"})

	got := read(t, u, v, "scan.txt", 0, 0)
	if got.Outcome != file.NotText {
		t.Errorf("want not text, got %q", got.Outcome)
	}
	if got.Text != "" {
		t.Errorf("bytes that are not text came back as %q", got.Text)
	}
}

func TestAPathWithNoFileIsMissing(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lecture.txt": "said"})

	got := read(t, u, v, "gone.txt", 0, 0)
	if got.Outcome != file.Missing {
		t.Errorf("want missing, got %q", got.Outcome)
	}
}

// The path is from the vault root, and everything that names another place is
// refused as a path outside the vault.
func TestAPathThatDoesNotStayInTheVaultIsRefused(t *testing.T) {
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("not yours"), 0o600); err != nil {
		t.Fatal(err)
	}
	u, v := openFileRead(t, map[string]string{"notes/lecture.txt": "said"})

	for _, path := range []string{
		"",
		"..",
		"../secret.txt",
		"notes/../../secret.txt",
		secret,
		"/etc/passwd",
	} {
		if _, err := u.Execute(t.Context(), v, path, 0, 0); !errors.Is(err, port.ErrOutside) {
			t.Errorf("read %q: want ErrOutside, got %v", path, err)
		}
	}
}

// A folder inside the vault may be a link to somewhere else. The text of the
// path says nothing about that; the filesystem does.
func TestAReadDoesNotFollowALinkOutOfTheVault(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("not yours"), 0o600); err != nil {
		t.Fatal(err)
	}
	u, v := openFileRead(t, map[string]string{"notes/lecture.txt": "said"})
	if err := os.Symlink(outside, filepath.Join(v.Path, "linked")); err != nil {
		t.Skipf("this filesystem does not do symlinks: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(v.Path, "secret.txt")); err != nil {
		t.Skipf("this filesystem does not do symlinks: %v", err)
	}

	for _, path := range []string{"linked/secret.txt", "secret.txt"} {
		if _, err := u.Execute(t.Context(), v, path, 0, 0); !errors.Is(err, port.ErrOutside) {
			t.Errorf("read %q: want ErrOutside, got %v", path, err)
		}
	}
}

// A name beginning with a dot belongs to a tool, and nothing that walks or
// lists a vault reports one. A read of a file goes by the same rule.
func TestWhatTheVaultPassesOverIsNotRead(t *testing.T) {
	u, v := openFileRead(t, map[string]string{
		".env":              "SECRET=1\n",
		".git/config":       "[core]\n",
		"notes/.hidden.md":  "# hidden\n",
		"notes/lecture.txt": "said",
	})

	for _, path := range []string{".env", ".git/config", "notes/.hidden.md"} {
		got := read(t, u, v, path, 0, 0)
		if got.Outcome != file.LeftAlone {
			t.Errorf("read %q: want left alone, got %+v", path, got)
		}
		if got.Text != "" {
			t.Errorf("read %q: it came back as %q", path, got.Text)
		}
	}
	if got := read(t, u, v, "notes/lecture.txt", 0, 0); got.Outcome != file.Ok {
		t.Errorf("a file the vault holds came back as %+v", got)
	}
}

// A folder is not a file, and a path holding one is answered rather than being
// opened and read.
func TestAPathHoldingAFolderIsSaidToBeOne(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lectures/kinetics.txt": "said"})

	if got := read(t, u, v, "lectures", 0, 0); got.Outcome != file.AFolder {
		t.Errorf("want a folder, got %+v", got)
	}
}

// The application's own folder inside the vault is not the person's, and a read
// of a file does not reach into it.
func TestTheApplicationsOwnFolderIsNotRead(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"notes/lecture.txt": "said"})
	kept := filepath.Join(v.Path, filesystem.DefaultServiceDir, "kept.txt")
	if err := os.MkdirAll(filepath.Dir(kept), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kept, []byte("ours"), 0o600); err != nil {
		t.Fatal(err)
	}

	path := filesystem.DefaultServiceDir + "/kept.txt"
	if _, err := u.Execute(t.Context(), v, path, 0, 0); err == nil {
		t.Error("the application's own folder was read")
	}
}

// A range beginning before the file, or longer than one call carries, is a
// mistake in the asking.
func TestARangeNamedOutsideWhatOneCallCarriesIsRefused(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lecture.txt": "said"})

	for _, asked := range []struct{ start, length int }{
		{-1, 10},
		{0, -1},
		{0, file.MostRead + 1},
	} {
		if _, err := u.Execute(t.Context(), v, "lecture.txt", asked.start, asked.length); err == nil {
			t.Errorf("a range beginning at %d for %d bytes was not refused", asked.start, asked.length)
		}
	}
}

var errOutOfReach = errors.New("the disk did not answer")

// faulty is a vault on disk whose reader can refuse to open or to list, and can
// hand over a file other than the one on disk.
type faulty struct {
	open error
	list error
	file io.ReadSeekCloser
}

func (f faulty) Open(v domain.Vault) (port.VaultReader, error) {
	if f.open != nil {
		return nil, f.open
	}
	reader, err := filesystem.VaultReaders{}.Open(v)
	if err != nil {
		return nil, err
	}
	return faultyReader{VaultReader: reader, on: f}, nil
}

type faultyReader struct {
	port.VaultReader
	on faulty
}

func (r faultyReader) List(ctx context.Context, folder string) ([]domain.Entry, error) {
	if r.on.list != nil {
		return nil, r.on.list
	}
	return r.VaultReader.List(ctx, folder)
}

func (r faultyReader) Open(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	if r.on.file != nil {
		return r.on.file, nil
	}
	return r.VaultReader.Open(ctx, path)
}

// shifting is a file changing while it is read: it says it is size bytes long
// and holds what data holds. A seek or a read can be made to fail.
type shifting struct {
	data []byte
	size int64
	at   int64
	end  error
	seek error
	read error
}

func (f *shifting) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekEnd {
		if f.end != nil {
			return 0, f.end
		}
		f.at = f.size + offset
		return f.at, nil
	}
	if f.seek != nil {
		return 0, f.seek
	}
	f.at = offset
	return f.at, nil
}

func (f *shifting) Read(p []byte) (int, error) {
	if f.read != nil {
		return 0, f.read
	}
	if f.at >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.at:])
	f.at += int64(n)
	return n, nil
}

func (f *shifting) Close() error { return nil }

// One call carries sixteen thousand bytes, whether it asks for that much or
// asks for nothing in particular.
func TestOneCallCarriesSixteenThousandBytes(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lecture.txt": strings.Repeat("a", 20000)})

	if got := read(t, u, v, "lecture.txt", 0, 0); got.Length != 16000 {
		t.Errorf("a call asking for nothing in particular carried %d bytes, want 16000", got.Length)
	}
	if got := read(t, u, v, "lecture.txt", 0, 16000); got.Length != 16000 {
		t.Errorf("a call asking for 16000 bytes carried %d", got.Length)
	}
}

// A range beginning before the file is refused before anything is looked at,
// where there is no file as much as where there is one.
func TestARangeBeginningBeforeTheFileIsRefusedWhereNoFileIs(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lecture.txt": "said"})

	if _, err := u.Execute(t.Context(), v, "gone.txt", -1, 0); err == nil {
		t.Error("a range beginning at -1 was not refused")
	}
}

// A vault or a file that will not answer is an error, and no outcome of the
// file.
func TestAReadThatDoesNotAnswerIsAnError(t *testing.T) {
	for _, c := range []struct {
		name    string
		readers faulty
	}{
		{"vault will not open", faulty{open: errOutOfReach}},
		{"folder will not list", faulty{list: errOutOfReach}},
		{"end not found", faulty{file: &shifting{data: []byte("said"), size: 4, end: errOutOfReach}}},
		{"start not found", faulty{file: &shifting{data: []byte("said"), size: 4, seek: errOutOfReach}}},
		{"bytes not read", faulty{file: &shifting{data: []byte("said"), size: 4, read: errOutOfReach}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, v := openFileRead(t, map[string]string{"lecture.txt": "said"})

			got, err := file.Read{Readers: c.readers}.Execute(t.Context(), v, "lecture.txt", 0, 0)
			if !errors.Is(err, errOutOfReach) {
				t.Errorf("read = %+v, %v, want the vault's error", got, err)
			}
		})
	}
}

// A path whose folder the vault does not list is a path it passes over.
func TestAFileInAFolderTheVaultDoesNotListIsLeftAlone(t *testing.T) {
	_, v := openFileRead(t, map[string]string{"notes/lecture.txt": "said"})

	got, err := file.Read{Readers: faulty{list: fs.ErrNotExist}}.Execute(t.Context(), v, "notes/lecture.txt", 0, 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != file.LeftAlone || got.Text != "" {
		t.Errorf("want left alone, got %+v", got)
	}
}

// Every file of a folder is read, and not only the first the vault lists.
func TestAFileListedAfterAnotherIsRead(t *testing.T) {
	u, v := openFileRead(t, map[string]string{
		"notes/a lecture.txt":      "first",
		"notes/another lecture.md": "second",
	})

	if got := read(t, u, v, "notes/another lecture.md", 0, 0); got.Outcome != file.Ok || got.Text != "second" {
		t.Errorf("the second file came back as %+v", got)
	}
}

// A file shorter by the time it is read than when it was measured comes back
// as what it holds, with nothing made up to fill the range.
func TestAFileThatShrankWhileItWasReadComesBackAsWhatItHolds(t *testing.T) {
	_, v := openFileRead(t, map[string]string{"lecture.txt": "said and said"})

	shrunk := faulty{file: &shifting{data: []byte("short"), size: 10}}
	got, err := file.Read{Readers: shrunk}.Execute(t.Context(), v, "lecture.txt", 0, 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Outcome != file.Ok || got.Text != "short" || got.Length != 5 {
		t.Errorf("a file that shrank came back as %+v", got)
	}

	gone := faulty{file: &shifting{data: []byte("ab"), size: 10}}
	got, err = file.Read{Readers: gone}.Execute(t.Context(), v, "lecture.txt", 5, 0)
	if err != nil {
		t.Fatalf("read past what is left: %v", err)
	}
	if got.Outcome != file.Ok || got.Text != "" {
		t.Errorf("a range past what is left came back as %+v", got)
	}
}

// A range opening inside a character and reaching the end of the file opens
// after that character.
func TestARangeOpeningInsideACharacterAtTheEndOpensAfterIt(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lecture.txt": "śā"})

	got := read(t, u, v, "lecture.txt", 1, 0)
	if got.Outcome != file.Ok || got.Text != "ā" || got.Start != 2 {
		t.Errorf("want ā from byte 2, got %+v", got)
	}
}

// A range lying wholly inside one character holds no character, and comes back
// empty.
func TestARangeInsideOneCharacterIsEmpty(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"lecture.txt": "śāstra"})

	for _, start := range []int{0, 1} {
		got := read(t, u, v, "lecture.txt", start, 1)
		if got.Outcome != file.Ok || got.Text != "" || got.Length != 0 {
			t.Errorf("one byte from %d came back as %+v", start, got)
		}
	}
}

// A file that begins inside a character is what the file holds, and that is not
// text.
func TestAFileBeginningInsideACharacterIsNotText(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"scan.txt": "\x80abc"})

	for _, length := range []int{0, 1} {
		if got := read(t, u, v, "scan.txt", 0, length); got.Outcome != file.NotText {
			t.Errorf("%d bytes from the start: want not text, got %+v", length, got)
		}
	}
}

// Only the character the range closes inside is left off, and one cut short
// before it is what the file holds.
func TestACharacterCutShortInsideTheRangeIsNotText(t *testing.T) {
	u, v := openFileRead(t, map[string]string{"scan.txt": "\xc5\xc5abc"})

	if got := read(t, u, v, "scan.txt", 0, 2); got.Outcome != file.NotText {
		t.Errorf("want not text, got %+v", got)
	}
}

// The size a file was measured at bounds what is read, whatever the file holds
// by the time its bytes come.
func TestAFileThatGrewWhileItWasReadIsReadOnlyToWhereItWasMeasured(t *testing.T) {
	_, v := openFileRead(t, map[string]string{"lecture.txt": "said"})
	grown := faulty{file: &shifting{data: []byte("said and said"), size: 4}}

	got, err := file.Read{Readers: grown}.Execute(t.Context(), v, "lecture.txt", 2, 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if got.Text != "id" || got.Start != 2 || got.Length != 2 || got.Whole != 4 {
		t.Errorf("a file that grew came back as %+v", got)
	}
}

func TestWholeCharactersLeavesOffACutLeadByte(t *testing.T) {
	start, raw := file.WholeCharacters(0, []byte{0xE2, 0x82}, 10)
	if start != 0 || len(raw) != 0 {
		t.Errorf("a range closing inside a character kept %x from %d", raw, start)
	}
}

func TestWholeCharactersOfOnlyContinuationBytesIsEmpty(t *testing.T) {
	start, raw := file.WholeCharacters(1, []byte{0x80}, 5)
	if start != 2 || len(raw) != 0 {
		t.Errorf("a range of continuation bytes came back as %x from %d", raw, start)
	}
}
