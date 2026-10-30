package hosttest

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// store is where the files of a vault are kept.
type store interface {
	fileReader
	fileWriter
}

// fileReader reads the files of a vault.
type fileReader interface {
	// stat returns what the file is, or fs.ErrNotExist.
	stat(filePath string) (host.FileStat, error)
	read(filePath string) ([]byte, error)
	// paths lists every file, with forward slashes.
	paths() ([]string, error)
}

// fileWriter changes the files of a vault.
type fileWriter interface {
	// write makes the file or replaces it, and returns what it is now.
	write(filePath string, content []byte) (host.FileStat, error)
	// create makes a file that is not there, else it returns host.ErrAlreadyExists.
	create(filePath string, content []byte) (host.FileStat, error)
	remove(filePath string) error
}

// memory keeps the files in memory. Every write moves the modification time on by a second.
type memory struct {
	files map[string][]byte
	stats map[string]host.FileStat
	clock time.Time
}

func newMemory() *memory {
	return &memory{files: map[string][]byte{}, stats: map[string]host.FileStat{}, clock: time.Unix(1_700_000_000, 0)}
}

func (m *memory) stat(filePath string) (host.FileStat, error) {
	fp, there := m.stats[filePath]
	if !there {
		return host.FileStat{}, fs.ErrNotExist
	}
	fp.Kind = getKind(filePath)
	return fp, nil
}

func (m *memory) read(filePath string) ([]byte, error) {
	content, there := m.files[filePath]
	if !there {
		return nil, fs.ErrNotExist
	}
	return append([]byte(nil), content...), nil
}

func (m *memory) write(filePath string, content []byte) (host.FileStat, error) {
	m.clock = m.clock.Add(time.Second)
	m.files[filePath] = append([]byte(nil), content...)
	m.stats[filePath] = host.FileStat{Path: filePath, Kind: getKind(filePath), Size: int64(len(content)), ModTime: m.clock}
	return m.stats[filePath], nil
}

func (m *memory) create(filePath string, content []byte) (host.FileStat, error) {
	if _, there := m.files[filePath]; there {
		return host.FileStat{}, host.ErrAlreadyExists
	}
	return m.write(filePath, content)
}

func (m *memory) remove(filePath string) error {
	if _, there := m.files[filePath]; !there {
		return fs.ErrNotExist
	}
	delete(m.files, filePath)
	delete(m.stats, filePath)
	return nil
}

func (m *memory) paths() ([]string, error) {
	out := make([]string, 0, len(m.files))
	for p := range m.files {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// disk stores the files in a folder. A test can modify and read them as another program
// does, and make writes fail through the permissions of the folder.
type disk struct {
	root string
}

func (d disk) at(filePath string) string { return filepath.Join(d.root, filepath.FromSlash(filePath)) }

func (d disk) stat(filePath string) (host.FileStat, error) {
	info, err := os.Stat(d.at(filePath))
	if err != nil {
		return host.FileStat{}, err
	}
	return host.FileStat{Path: filePath, Kind: getKind(filePath), Size: info.Size(), ModTime: info.ModTime()}, nil
}

func (d disk) read(filePath string) ([]byte, error) { return os.ReadFile(d.at(filePath)) }

func (d disk) write(filePath string, content []byte) (host.FileStat, error) {
	target := d.at(filePath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return host.FileStat{}, err
	}
	// A save is a rename over the target name, so permission to write is decided by the folder containing the name.
	temp, err := os.CreateTemp(filepath.Dir(target), ".write-*")
	if err != nil {
		return host.FileStat{}, err
	}
	_, werr := temp.Write(content)
	cerr := temp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		return host.FileStat{}, removeAfterFailure(temp.Name(), err)
	}
	if err := os.Chmod(temp.Name(), 0o644); err != nil {
		return host.FileStat{}, removeAfterFailure(temp.Name(), err)
	}
	if err := os.Rename(temp.Name(), target); err != nil {
		return host.FileStat{}, removeAfterFailure(temp.Name(), err)
	}
	return d.stat(filePath)
}

func (d disk) create(filePath string, content []byte) (host.FileStat, error) {
	target := d.at(filePath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return host.FileStat{}, err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return host.FileStat{}, host.ErrAlreadyExists
	}
	if err != nil {
		return host.FileStat{}, err
	}
	_, werr := file.Write(content)
	if err := errors.Join(werr, file.Close()); err != nil {
		return host.FileStat{}, err
	}
	return d.stat(filePath)
}

// removeAfterFailure removes the temporary file left by the failure cause and returns cause, joined
// with the removal error when removing fails.
func removeAfterFailure(name string, cause error) error {
	if err := os.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return errors.Join(cause, err)
	}
	return cause
}

func (d disk) remove(filePath string) error { return os.Remove(d.at(filePath)) }

func (d disk) paths() ([]string, error) {
	var out []string
	err := filepath.WalkDir(d.root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(d.root, p)
		if relErr != nil {
			return relErr
		}
		if strings.HasPrefix(path.Base(rel), ".") {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out, err
}
