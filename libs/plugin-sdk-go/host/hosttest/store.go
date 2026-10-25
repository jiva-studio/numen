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

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// store is where the files of a vault are kept.
type store interface {
	// stat returns what the file is, or fs.ErrNotExist.
	stat(filePath string) (domain.Fingerprint, error)
	read(filePath string) ([]byte, error)
	// write makes the file or replaces it, and returns what it is now.
	write(filePath string, content []byte) (domain.Fingerprint, error)
	// create makes a file that is not there, else it returns host.ErrOccupied.
	create(filePath string, content []byte) (domain.Fingerprint, error)
	remove(filePath string) error
	// paths lists every file, with forward slashes.
	paths() []string
}

// memory keeps the files in memory. Every write moves the modification time on by a second.
type memory struct {
	files        map[string][]byte
	fingerprints map[string]domain.Fingerprint
	clock        time.Time
}

func newMemory() *memory {
	return &memory{files: map[string][]byte{}, fingerprints: map[string]domain.Fingerprint{}, clock: time.Unix(1_700_000_000, 0)}
}

func (m *memory) stat(filePath string) (domain.Fingerprint, error) {
	fp, there := m.fingerprints[filePath]
	if !there {
		return domain.Fingerprint{}, fs.ErrNotExist
	}
	return fp, nil
}

func (m *memory) read(filePath string) ([]byte, error) {
	content, there := m.files[filePath]
	if !there {
		return nil, fs.ErrNotExist
	}
	return append([]byte(nil), content...), nil
}

func (m *memory) write(filePath string, content []byte) (domain.Fingerprint, error) {
	m.clock = m.clock.Add(time.Second)
	m.files[filePath] = append([]byte(nil), content...)
	m.fingerprints[filePath] = domain.Fingerprint{Path: filePath, Kind: kindOf(filePath), Size: int64(len(content)), ModTime: m.clock}
	return m.fingerprints[filePath], nil
}

func (m *memory) create(filePath string, content []byte) (domain.Fingerprint, error) {
	if _, there := m.files[filePath]; there {
		return domain.Fingerprint{}, host.ErrOccupied
	}
	return m.write(filePath, content)
}

func (m *memory) remove(filePath string) error {
	if _, there := m.files[filePath]; !there {
		return fs.ErrNotExist
	}
	delete(m.files, filePath)
	delete(m.fingerprints, filePath)
	return nil
}

func (m *memory) paths() []string {
	out := make([]string, 0, len(m.files))
	for p := range m.files {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// disk keeps the files in a folder, so that a test can change them and read them as another program
// would, and make them refuse the plugin with the permissions of the folder.
type disk struct{ root string }

func (d disk) at(filePath string) string { return filepath.Join(d.root, filepath.FromSlash(filePath)) }

func (d disk) stat(filePath string) (domain.Fingerprint, error) {
	info, err := os.Stat(d.at(filePath))
	if err != nil {
		return domain.Fingerprint{}, err
	}
	return domain.Fingerprint{Path: filePath, Kind: kindOf(filePath), Size: info.Size(), ModTime: info.ModTime()}, nil
}

func (d disk) read(filePath string) ([]byte, error) { return os.ReadFile(d.at(filePath)) }

func (d disk) write(filePath string, content []byte) (domain.Fingerprint, error) {
	target := d.at(filePath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return domain.Fingerprint{}, err
	}
	// A save lands by renaming over the name, so what refuses it is the folder the name is in.
	temp, err := os.CreateTemp(filepath.Dir(target), ".write-*")
	if err != nil {
		return domain.Fingerprint{}, err
	}
	_, werr := temp.Write(content)
	cerr := temp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(temp.Name())
		return domain.Fingerprint{}, err
	}
	if err := os.Chmod(temp.Name(), 0o644); err != nil {
		_ = os.Remove(temp.Name())
		return domain.Fingerprint{}, err
	}
	if err := os.Rename(temp.Name(), target); err != nil {
		_ = os.Remove(temp.Name())
		return domain.Fingerprint{}, err
	}
	return d.stat(filePath)
}

func (d disk) create(filePath string, content []byte) (domain.Fingerprint, error) {
	target := d.at(filePath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return domain.Fingerprint{}, err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return domain.Fingerprint{}, host.ErrOccupied
	}
	if err != nil {
		return domain.Fingerprint{}, err
	}
	_, werr := file.Write(content)
	if err := errors.Join(werr, file.Close()); err != nil {
		return domain.Fingerprint{}, err
	}
	return d.stat(filePath)
}

func (d disk) remove(filePath string) error { return os.Remove(d.at(filePath)) }

func (d disk) paths() []string {
	var out []string
	_ = filepath.WalkDir(d.root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
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
	return out
}
