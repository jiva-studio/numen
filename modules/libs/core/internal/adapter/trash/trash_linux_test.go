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

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// folder is a vault of one note, somewhere of its own.
func folder(t *testing.T, name string) string {
	t.Helper()
	at := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(at, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(at, "note.md"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	return at
}

// note is one trashinfo file read as the fields it carries.
func note(t *testing.T, at string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) == 0 || lines[0] != "[Trash Info]" {
		t.Fatalf("%s does not begin with [Trash Info]:\n%s", at, raw)
	}
	fields := map[string]string{}
	for _, line := range lines[1:] {
		name, value, found := strings.Cut(line, "=")
		if !found {
			t.Fatalf("%s has a line that is not a field: %q", at, line)
		}
		fields[name] = value
	}
	return fields
}

func TestAFolderGoesToTheTrash(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	at := folder(t, "Vault")

	if err := New().Trash(at); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(at); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s is still where it was: %v", at, err)
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
	if came != at {
		t.Errorf("Path is %q, want %q", came, at)
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
	at := folder(t, "Notes & Papers 100% #1")

	if err := New().Trash(at); err != nil {
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
	if came != at {
		t.Errorf("Path is %q, want %q", came, at)
	}
}

// TestANameAlreadyTaken is what the trash does with the second vault a person
// called the same thing: both are in there, each under its own name, and each
// note says where its own folder came from.
func TestANameAlreadyTaken(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	first, second := folder(t, "Vault"), folder(t, "Vault")

	for _, at := range []string{first, second} {
		if err := New().Trash(at); err != nil {
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

// TestAVolumeThatTakesNoTrash is the rule that a folder is never deleted
// because there is nowhere to put it: a volume whose root refuses a trash
// directory has none, and the answer says so.
func TestAVolumeThatTakesNoTrash(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes into a directory whose mode refuses it")
	}
	top := t.TempDir()
	if err := os.Chmod(top, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(top, 0o700) })

	if _, err := volume(top, os.Getuid()); !errors.Is(err, port.ErrNoTrash) {
		t.Errorf("a volume root that takes no trash gave %v, want %v", err, port.ErrNoTrash)
	}
}

// TestNowhereToPutIt is the whole answer for a folder this machine cannot trash:
// the sentinel, and the folder still where the person left it.
func TestNowhereToPutIt(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root makes a trash directory at the root of any volume")
	}
	at := folder(t, "Vault")
	top, err := topdir(at)
	if err != nil {
		t.Fatal(err)
	}
	if syscall.Access(top, 0o2) == nil {
		t.Skipf("%s takes a trash directory on this machine", top)
	}

	// No trash of this login, so the volume's own is the only one there could
	// be, and its root refuses one.
	if err := sendTo(at, ""); !errors.Is(err, port.ErrNoTrash) {
		t.Errorf("trashing %s gave %v, want %v", at, err, port.ErrNoTrash)
	}
	if _, err := os.Lstat(filepath.Join(at, "note.md")); err != nil {
		t.Errorf("%s did not stay where it was: %v", at, err)
	}
}
