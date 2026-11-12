package hosttest

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/atomicfile"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// vaultFiles is where the files of a vault are kept. The root is the folder of a vault on disk, and
// empty for a vault in memory.
type vaultFiles struct {
	fileReader
	fileWriter
	root string
}

func inMemory() vaultFiles {
	m := newMemory()
	return vaultFiles{fileReader: m, fileWriter: m}
}

func onDisk(root string) vaultFiles {
	d := disk{root: root}
	return vaultFiles{fileReader: d, fileWriter: d, root: root}
}

// fileReader reads the files of a vault.
type fileReader interface {
	// stat returns what the file is, or fs.ErrNotExist.
	stat(filePath string) (domain.FileStat, error)
	read(filePath string) ([]byte, error)
	// paths lists every file, with forward slashes.
	paths() ([]string, error)
}

// fileWriter changes the files of a vault.
type fileWriter interface {
	// write makes the file or replaces it, and returns what it is now.
	write(filePath string, content []byte) (domain.FileStat, error)
	// create makes a file that is not there, else it returns host.ErrAlreadyExists.
	create(filePath string, content []byte) (domain.FileStat, error)
	remove(filePath string) error
}

// memory keeps the files in memory. Every write moves the modification time on by a second.
type memory struct {
	files map[string][]byte
	stats map[string]domain.FileStat
	clock time.Time
}

func newMemory() *memory {
	return &memory{files: map[string][]byte{}, stats: map[string]domain.FileStat{}, clock: time.Unix(1_700_000_000, 0)}
}

func (m *memory) stat(filePath string) (domain.FileStat, error) {
	fp, there := m.stats[filePath]
	if !there {
		return domain.FileStat{}, fs.ErrNotExist
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

func (m *memory) write(filePath string, content []byte) (domain.FileStat, error) {
	m.clock = m.clock.Add(time.Second)
	m.files[filePath] = append([]byte(nil), content...)
	m.stats[filePath] = domain.FileStat{Path: filePath, Kind: getKind(filePath), Size: int64(len(content)), ModTime: m.clock}
	return m.stats[filePath], nil
}

func (m *memory) create(filePath string, content []byte) (domain.FileStat, error) {
	if _, there := m.files[filePath]; there {
		return domain.FileStat{}, host.ErrAlreadyExists
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

func (d disk) at(filePath string) (string, error) {
	rel := filepath.FromSlash(filePath)
	if !filepath.IsLocal(rel) {
		return "", fs.ErrInvalid
	}
	root := filepath.Clean(d.root)
	target := filepath.Join(root, rel)
	if !strings.HasPrefix(target, root+string(filepath.Separator)) {
		return "", fs.ErrInvalid
	}
	return target, nil
}

func (d disk) stat(filePath string) (domain.FileStat, error) {
	target, err := d.at(filePath)
	if err != nil {
		return domain.FileStat{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return domain.FileStat{}, err
	}
	if !info.Mode().IsRegular() {
		return domain.FileStat{}, host.ErrNotANote
	}
	return domain.FileStat{Path: filePath, Kind: getKind(filePath), Size: info.Size(), ModTime: info.ModTime()}, nil
}

func (d disk) read(filePath string) ([]byte, error) {
	target, err := d.at(filePath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(target)
}

func (d disk) write(filePath string, content []byte) (domain.FileStat, error) {
	target, err := d.at(filePath)
	if err != nil {
		return domain.FileStat{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return domain.FileStat{}, err
	}
	// A save is a rename over the target name, so permission to write is decided by the folder containing the name.
	if err := atomicfile.WriteFile(target, bytes.NewReader(content), 0o600); err != nil {
		return domain.FileStat{}, err
	}
	return d.stat(filePath)
}

func (d disk) create(filePath string, content []byte) (domain.FileStat, error) {
	target, err := d.at(filePath)
	if err != nil {
		return domain.FileStat{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return domain.FileStat{}, err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return domain.FileStat{}, host.ErrAlreadyExists
	}
	if err != nil {
		return domain.FileStat{}, err
	}
	_, werr := file.Write(content)
	if err := errors.Join(werr, file.Close()); err != nil {
		return domain.FileStat{}, err
	}
	return d.stat(filePath)
}

func (d disk) remove(filePath string) error {
	target, err := d.at(filePath)
	if err != nil {
		return err
	}
	return os.Remove(target)
}

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
