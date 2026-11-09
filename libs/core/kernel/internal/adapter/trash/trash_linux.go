//go:build linux

package trash

import (
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// suffix is the extension of the info file that records where a trashed folder
// came from.
const suffix = ".trashinfo"

// deletedAt is the form the note writes the time in, in local time.
const deletedAt = "2006-01-02T15:04:05"

func moveToTrash(path string, deletedOn time.Time) error {
	return moveToUserOrVolumeTrash(path, getUserTrashDir(), deletedOn)
}

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

func moveToUserOrVolumeTrash(path, home string, deletedOn time.Time) error {
	if home != "" {
		if err := moveIntoTrash(home, path, path, deletedOn); !errors.Is(err, syscall.EXDEV) {
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
	return moveIntoTrash(dir, path, from, deletedOn)
}

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
		isRoot, err := isVolumeRoot(parent, folder.Dev)
		if err != nil {
			return "", err
		}
		if isRoot {
			return pos, nil
		}
		pos = parent
	}
}

func isVolumeRoot(parent string, dev uint64) (bool, error) {
	var up syscall.Stat_t
	err := syscall.Lstat(parent, &up)
	// A parent nothing here can look at is as far as this goes: what it is
	// on cannot be compared, so the folder below it is the root.
	if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) || errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat parent %s: %w", parent, err)
	}
	return up.Dev != dev, nil
}

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
		return "", fmt.Errorf("%s: %w", top, machine.ErrNoTrash)
	}
	return mine, nil
}

func moveIntoTrash(dir, path, from string, deletedOn time.Time) error {
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
	_, err = note.WriteString(formatTrashInfo(from, deletedOn))
	if closed := note.Close(); err == nil {
		err = closed
	}
	if err == nil {
		err = os.Rename(path, filepath.Join(files, name))
	}
	if err != nil {
		return errors.Join(err, os.Remove(filepath.Join(info, name+suffix)))
	}
	return nil
}

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

func formatTrashInfo(from string, deletedOn time.Time) string {
	return "[Trash Info]\n" +
		"Path=" + (&url.URL{Path: from}).EscapedPath() + "\n" +
		"DeletionDate=" + deletedOn.Format(deletedAt) + "\n"
}
