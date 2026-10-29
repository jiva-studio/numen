//go:build linux

package trash

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// folder is a vault of one note, somewhere of its own.
func folder(t *testing.T, name string) string {
	t.Helper()
	pos := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(pos, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pos, "note.md"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	return pos
}

// note is one trashinfo file read as the fields it carries.
func note(t *testing.T, pos string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(pos)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) == 0 || lines[0] != "[Trash Info]" {
		t.Fatalf("%s does not begin with [Trash Info]:\n%s", pos, raw)
	}
	fields := map[string]string{}
	for _, line := range lines[1:] {
		name, value, found := strings.Cut(line, "=")
		if !found {
			t.Fatalf("%s has a line that is not a field: %q", pos, line)
		}
		fields[name] = value
	}
	return fields
}

func TestAFolderGoesToTheTrash(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	pos := folder(t, "Vault")

	if err := New().Trash(pos); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(pos); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s is still where it was: %v", pos, err)
	}
	moved := filepath.Join(data, "Trash", "files", "Vault")
	text, err := os.ReadFile(filepath.Join(moved, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(text) != "hello" {
		t.Errorf("the note in the trash reads %q, want %q", text, "hello")
	}

	fields := note(t, filepath.Join(data, "Trash", "info", "Vault.trashinfo"))
	came, err := url.PathUnescape(fields["Path"])
	if err != nil {
		t.Fatalf("Path=%q does not decode: %v", fields["Path"], err)
	}
	if came != pos {
		t.Errorf("Path is %q, want %q", came, pos)
	}
	if _, err := time.ParseInLocation(deletedAt, fields["DeletionDate"], time.Local); err != nil {
		t.Errorf("DeletionDate=%q does not parse: %v", fields["DeletionDate"], err)
	}
}

// TestAFolderWithAPunctuatedName is the encoding of the recorded path: a name a
// person may well choose has characters a URL does not carry as they are, and
// what is written has to come back as what was deleted.
func TestAFolderWithAPunctuatedName(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	pos := folder(t, "Notes & Papers 100% #1")

	if err := New().Trash(pos); err != nil {
		t.Fatal(err)
	}

	fields := note(t, filepath.Join(data, "Trash", "info", "Notes & Papers 100% #1.trashinfo"))
	if strings.ContainsAny(fields["Path"], " #") {
		t.Errorf("Path=%q is not encoded", fields["Path"])
	}
	came, err := url.PathUnescape(fields["Path"])
	if err != nil {
		t.Fatalf("Path=%q does not decode: %v", fields["Path"], err)
	}
	if came != pos {
		t.Errorf("Path is %q, want %q", came, pos)
	}
}

// TestANameAlreadyTaken verifies the trash handling of a second vault with the
// same name: both are in the trash, each under its own name, and each note
// records the origin folder.
func TestANameAlreadyTaken(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	first, second := folder(t, "Vault"), folder(t, "Vault")

	for _, item := range []string{first, second} {
		if err := New().Trash(item); err != nil {
			t.Fatal(err)
		}
	}

	for _, name := range []string{"Vault", "Vault.2"} {
		if _, err := os.Lstat(filepath.Join(data, "Trash", "files", name)); err != nil {
			t.Fatalf("%s is not in the trash: %v", name, err)
		}
	}
	came := map[string]bool{}
	for _, name := range []string{"Vault", "Vault.2"} {
		path, err := url.PathUnescape(note(t, filepath.Join(data, "Trash", "info", name+".trashinfo"))["Path"])
		if err != nil {
			t.Fatal(err)
		}
		came[path] = true
	}
	if !came[first] || !came[second] {
		t.Errorf("the notes name %v, want %q and %q", came, first, second)
	}
}

// TestAVolumeThatConsumesNoTrash verifies that a folder is never deleted when
// there is nowhere to move it: a volume whose root rejects a trash directory
// has none, and the returned error reports that.
func TestAVolumeThatConsumesNoTrash(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes into a directory whose mode refuses it")
	}
	top := t.TempDir()
	if err := os.Chmod(top, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(top, 0o700) })

	if _, err := getVolumeTrashDir(top, os.Getuid()); !errors.Is(err, machineport.ErrNoTrash) {
		t.Errorf("a volume root that takes no trash gave %v, want %v", err, machineport.ErrNoTrash)
	}
}

// TestNowhereToInsertIt verifies the result for a folder this machine cannot
// trash: the sentinel error, and the folder remains in place.
func TestNowhereToInsertIt(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root makes a trash directory at the root of any volume")
	}
	pos := folder(t, "Vault")
	top, err := getVolumeRoot(pos)
	if err != nil {
		t.Fatal(err)
	}
	if syscall.Access(top, 0o2) == nil {
		t.Skipf("%s takes a trash directory on this machine", top)
	}

	// The user has no trash, so the volume's own trash is the only candidate, and
	// its root rejects one.
	if err := moveToUserOrVolumeTrash(pos, ""); !errors.Is(err, machineport.ErrNoTrash) {
		t.Errorf("trashing %s gave %v, want %v", pos, err, machineport.ErrNoTrash)
	}
	if _, err := os.Lstat(filepath.Join(pos, "note.md")); err != nil {
		t.Errorf("%s did not stay where it was: %v", pos, err)
	}
}
