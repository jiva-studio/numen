//go:build linux

package trash

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// suffix names the note that says where a trashed folder came from.
const suffix = ".trashinfo"

// deletedAt is the form the note writes the time in, in local time.
const deletedAt = "2006-01-02T15:04:05"

// send puts the folder in the trash the desktop reads.
func send(path string) error {
	return sent(path, home())
}

// home is the trash of this login. It is empty where the machine will not say
// where this person's data lives.
func home() string {
	data := os.Getenv("XDG_DATA_HOME")
	if !filepath.IsAbs(data) {
		dir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		data = filepath.Join(dir, ".local", "share")
	}
	return filepath.Join(data, "Trash")
}

// sent puts the folder in the trash of this login, and in the trash at the root
// of its own volume when a rename cannot reach the first. A volume that has no
// trash and takes none has nowhere to put it.
func sent(path, home string) error {
	if home != "" {
		if err := into(home, path, path); !errors.Is(err, syscall.EXDEV) {
			return err
		}
	}
	top, err := topdir(path)
	if err != nil {
		return err
	}
	dir, err := volume(top, os.Getuid())
	if err != nil {
		return err
	}
	from, err := filepath.Rel(top, path)
	if err != nil {
		return err
	}
	return into(dir, path, from)
}

// topdir is the root of the volume the folder is on: the last directory on the
// way up that is still on the folder's device.
func topdir(path string) (string, error) {
	var folder syscall.Stat_t
	if err := syscall.Lstat(path, &folder); err != nil {
		return "", err
	}
	at := path
	for {
		parent := filepath.Dir(at)
		if parent == at {
			return at, nil
		}
		var up syscall.Stat_t
		if err := syscall.Lstat(parent, &up); err != nil || up.Dev != folder.Dev {
			return at, nil
		}
		at = parent
	}
}

// volume is the trash at the root of a mounted volume: the one the volume
// carries for everybody where it is a sticky directory of its own, and this
// login's otherwise.
func volume(top string, uid int) (string, error) {
	shared := filepath.Join(top, ".Trash")
	if info, err := os.Lstat(shared); err == nil && info.IsDir() && info.Mode()&os.ModeSticky != 0 {
		mine := filepath.Join(shared, strconv.Itoa(uid))
		if err := os.MkdirAll(mine, 0o700); err == nil {
			return mine, nil
		}
	}
	mine := filepath.Join(top, ".Trash-"+strconv.Itoa(uid))
	if err := os.MkdirAll(mine, 0o700); err != nil {
		return "", fmt.Errorf("%s: %w", top, port.ErrNoTrash)
	}
	return mine, nil
}

// into moves the folder into one trash directory under a name nothing there
// holds, and records where it came from beside it.
//
// The note is written first: the name is this deletion's before anything moves
// under it. A move that fails takes the note back with it.
func into(dir, path, from string) error {
	files := filepath.Join(dir, "files")
	info := filepath.Join(dir, "info")
	for _, at := range []string{files, info} {
		if err := os.MkdirAll(at, 0o700); err != nil {
			return err
		}
	}

	name, note, err := reserve(files, info, filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = note.WriteString(record(from))
	if closed := note.Close(); err == nil {
		err = closed
	}
	if err == nil {
		err = os.Rename(path, filepath.Join(files, name))
	}
	if err != nil {
		os.Remove(filepath.Join(info, name+suffix))
		return err
	}
	return nil
}

// reserve is a free name in one trash directory and the open note that holds
// it. Two deletions of folders called the same thing race for the note, and the
// one that creates it has the name.
func reserve(files, info, name string) (string, *os.File, error) {
	for n := 1; ; n++ {
		taken := name
		if n > 1 {
			taken = name + "." + strconv.Itoa(n)
		}
		if _, err := os.Lstat(filepath.Join(files, taken)); err == nil {
			continue
		}
		note, err := os.OpenFile(filepath.Join(info, taken+suffix), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return taken, note, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, err
		}
	}
}

// record is the note beside a trashed folder: where it was and when it was
// deleted. The path is URL-encoded, and is relative to the root of the volume
// for a trash that sits at that root.
func record(from string) string {
	return "[Trash Info]\n" +
		"Path=" + (&url.URL{Path: from}).EscapedPath() + "\n" +
		"DeletionDate=" + time.Now().Format(deletedAt) + "\n"
}
