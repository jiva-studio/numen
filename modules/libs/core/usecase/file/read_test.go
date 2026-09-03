package file_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
)

// readable is a vault on disk and the read that works on it.
func readable(t *testing.T, files map[string]string) (file.Read, domain.Vault) {
	t.Helper()
	return file.Read{Readers: filesystem.Readers{}}, testsupport.NewVault(t, files)
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
	u, v := readable(t, map[string]string{
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
		t.Errorf("the run was described as %+v", got)
	}
}

// A note is a file, and its bytes are what a read of a file hands over: the
// frontmatter is there, because that is what the file holds.
func TestANoteIsReadAsTheFileItIs(t *testing.T) {
	raw := "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n# Entropy\n"
	u, v := readable(t, map[string]string{"Entropy.md": raw})

	if got := read(t, u, v, "Entropy.md", 0, 0); got.Text != raw {
		t.Errorf("the note came back as %q", got.Text)
	}
}

// A transcript is long, and an agent reads it a run at a time: what the answer
// says about the run is what the next call is asked with.
func TestALongFileIsReadARunAtATime(t *testing.T) {
	whole := strings.Repeat("one line of what was said\n", 400)
	u, v := readable(t, map[string]string{"lecture.txt": whole})

	first := read(t, u, v, "lecture.txt", 0, 100)
	if first.Text != whole[:100] || first.Start != 0 || first.Length != 100 {
		t.Fatalf("the first run was %+v", first)
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
	u, v := readable(t, map[string]string{"lecture.txt": whole})

	got := read(t, u, v, "lecture.txt", 0, 0)
	if got.Length != file.MostRead {
		t.Errorf("one call carried %d bytes", got.Length)
	}
	if got.Whole != len(whole) {
		t.Errorf("the file is %d bytes and was said to be %d", len(whole), got.Whole)
	}
}

// A run is held within the file, so an offset past the end is an empty run and
// not an error.
func TestARunPastTheEndOfTheFileIsEmpty(t *testing.T) {
	u, v := readable(t, map[string]string{"lecture.txt": "short"})

	got := read(t, u, v, "lecture.txt", 900, 100)
	if got.Outcome != file.Ok || got.Text != "" || got.Length != 0 {
		t.Errorf("a run past the end gave %+v", got)
	}
	if got.Whole != 5 {
		t.Errorf("the file was said to be %d bytes", got.Whole)
	}
}

// A run is asked for in bytes and comes back as text, so a character the cut
// falls inside belongs to neither run.
func TestARunIsCutAtWholeCharacters(t *testing.T) {
	// "śāstra" is two two-byte characters and four one-byte ones, and the
	// emoji that follows is four bytes.
	whole := "śāstra 🪔 and śloka"
	u, v := readable(t, map[string]string{"lecture.txt": whole})

	// Opening one byte into the first character.
	opened := read(t, u, v, "lecture.txt", 1, 3)
	if opened.Text != "ā" || opened.Start != 2 || opened.Length != 2 {
		t.Errorf("a run opening inside a character came back as %+v", opened)
	}

	// Closing one byte into the second.
	closed := read(t, u, v, "lecture.txt", 0, 3)
	if closed.Text != "ś" || closed.Start != 0 || closed.Length != 2 {
		t.Errorf("a run closing inside a character came back as %+v", closed)
	}

	// The four-byte character stands at byte 9, and is closed inside at each of
	// the three places it can be.
	for _, length := range []int{10, 11, 12} {
		got := read(t, u, v, "lecture.txt", 0, length)
		if got.Text != "śāstra " {
			t.Errorf("a run of %d bytes came back as %q", length, got.Text)
		}
	}
	if got := read(t, u, v, "lecture.txt", 0, 13); got.Text != "śāstra 🪔" {
		t.Errorf("a run holding the whole character came back as %q", got.Text)
	}
}

// A run reaching the end of the file is not cut, so a file whose last character
// is cut short is named for what it is.
func TestARunReachingTheEndOfTheFileIsNotCut(t *testing.T) {
	u, v := readable(t, map[string]string{"lecture.txt": "said \xc5"})

	got := read(t, u, v, "lecture.txt", 0, 0)
	if got.Outcome != file.NotText {
		t.Errorf("want not text, got %+v", got)
	}
}

// A run goes out in a string field, so bytes that are not text are named as
// such and not carried.
func TestAFileThatIsNotTextIsSaidToBeNotText(t *testing.T) {
	u, v := readable(t, map[string]string{"scan.txt": "words \xff\xfe more"})

	got := read(t, u, v, "scan.txt", 0, 0)
	if got.Outcome != file.NotText {
		t.Errorf("want not text, got %q", got.Outcome)
	}
	if got.Text != "" {
		t.Errorf("bytes that are not text came back as %q", got.Text)
	}
}

func TestAPathWithNoFileIsMissing(t *testing.T) {
	u, v := readable(t, map[string]string{"lecture.txt": "said"})

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
	u, v := readable(t, map[string]string{"notes/lecture.txt": "said"})

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
	u, v := readable(t, map[string]string{"notes/lecture.txt": "said"})
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
	u, v := readable(t, map[string]string{
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
	u, v := readable(t, map[string]string{"lectures/kinetics.txt": "said"})

	if got := read(t, u, v, "lectures", 0, 0); got.Outcome != file.AFolder {
		t.Errorf("want a folder, got %+v", got)
	}
}

// The application's own folder inside the vault is not the person's, and a read
// of a file does not reach into it.
func TestTheApplicationsOwnFolderIsNotRead(t *testing.T) {
	u, v := readable(t, map[string]string{"notes/lecture.txt": "said"})
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

// A run beginning before the file, or longer than one call carries, is a
// mistake in the asking.
func TestARunNamedOutsideWhatOneCallCarriesIsRefused(t *testing.T) {
	u, v := readable(t, map[string]string{"lecture.txt": "said"})

	for _, run := range []struct{ start, length int }{
		{-1, 10},
		{0, -1},
		{0, file.MostRead + 1},
	} {
		if _, err := u.Execute(t.Context(), v, "lecture.txt", run.start, run.length); err == nil {
			t.Errorf("a run beginning at %d for %d bytes was not refused", run.start, run.length)
		}
	}
}
