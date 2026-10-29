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

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// suffix is the extension of the info file that records where a trashed folder
// came from.
const suffix = ".trashinfo"

// deletedAt is the form the note writes the time in, in local time.
const deletedAt = "2006-01-02T15:04:05"

// moveToTrash puts the folder in the trash the desktop reads.
func moveToTrash(path string) error {
	return moveToUserOrVolumeTrash(path, getUserTrashDir())
}

// getUserTrashDir is the trash of this login. It is empty where the machine will not say
// where this person's data lives.
func getUserTrashDir() string {
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

// moveToUserOrVolumeTrash puts the folder in the trash of this login, and in the trash at the root
// of its own volume when a rename cannot reach the first. A volume that has no
// trash and takes none has nowhere to put it.
func moveToUserOrVolumeTrash(path, home string) error {
	if home != "" {
		if err := moveIntoTrash(home, path, path); !errors.Is(err, syscall.EXDEV) {
			return err
		}
	}
	top, err := getVolumeRoot(path)
	if err != nil {
		return err
	}
	dir, err := getVolumeTrashDir(top, os.Getuid())
	if err != nil {
		return err
	}
	from, err := filepath.Rel(top, path)
	if err != nil {
		return err
	}
	return moveIntoTrash(dir, path, from)
}

// getVolumeRoot is the root of the volume the folder is on: the last directory on the
// way up that is still on the folder's device.
func getVolumeRoot(path string) (string, error) {
	var folder syscall.Stat_t
	if err := syscall.Lstat(path, &folder); err != nil {
		return "", err
	}
	pos := path
	for {
		parent := filepath.Dir(pos)
		if parent == pos {
			return pos, nil
		}
		var up syscall.Stat_t
		// A parent nothing here can look at is as far as this goes: what it is
		// on cannot be compared, so the folder below it is the root.
		if err := syscall.Lstat(parent, &up); errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) || errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
			return pos, nil
		} else if err != nil {
			return "", fmt.Errorf("stat parent %s: %w", parent, err)
		} else if up.Dev != folder.Dev {
			return pos, nil
		}
		pos = parent
	}
}

// getVolumeTrashDir returns the trash directory at the root of a mounted
// volume: the per-user directory inside the shared `.Trash` when that is a
// sticky directory, and the user's own `.Trash-<uid>` otherwise.
func getVolumeTrashDir(top string, uid int) (string, error) {
	shared := filepath.Join(top, ".Trash")
	if info, err := os.Lstat(shared); err == nil && info.IsDir() && info.Mode()&os.ModeSticky != 0 {
		mine := filepath.Join(shared, strconv.Itoa(uid))
		if err := os.MkdirAll(mine, 0o700); err == nil {
			return mine, nil
		}
	}
	mine := filepath.Join(top, ".Trash-"+strconv.Itoa(uid))
	if err := os.MkdirAll(mine, 0o700); err != nil {
		return "", fmt.Errorf("%s: %w", top, machineport.ErrNoTrash)
	}
	return mine, nil
}

// moveIntoTrash moves the folder into one trash directory under a name nothing
// there holds, and records where it came from beside it.
//
// The note is written first: the name is this deletion's before anything moves
// under it. A move that fails takes the note back with it.
func moveIntoTrash(dir, path, from string) error {
	files := filepath.Join(dir, "files")
	info := filepath.Join(dir, "info")
	for _, item := range []string{files, info} {
		if err := os.MkdirAll(item, 0o700); err != nil {
			return err
		}
	}

	name, note, err := reserveTrashName(files, info, filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = note.WriteString(formatTrashInfo(from))
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

// reserveTrashName is a free name in one trash directory and the open note that holds
// it. Two deletions of folders called the same thing race for the note, and the
// one that creates it has the name.
func reserveTrashName(files, notes, name string) (string, *os.File, error) {
	for n := 1; ; n++ {
		targetName := name
		if n > 1 {
			targetName = name + "." + strconv.Itoa(n)
		}
		if _, err := os.Lstat(filepath.Join(files, targetName)); err == nil {
			continue
		}
		note, err := os.OpenFile(filepath.Join(notes, targetName+suffix), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return targetName, note, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, err
		}
	}
}

// formatTrashInfo returns the info file content for a trashed folder: its
// original path and deletion time. The path is URL-encoded, and is relative to
// the volume root for a trash at that root.
func formatTrashInfo(from string) string {
	return "[Trash Info]\n" +
		"Path=" + (&url.URL{Path: from}).EscapedPath() + "\n" +
		"DeletionDate=" + time.Now().Format(deletedAt) + "\n"
}
