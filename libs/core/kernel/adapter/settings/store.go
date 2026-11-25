package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Store reads and writes the settings file at the path GetPath gives. What it reads as checked
// leaves out the values that the declarations of the app and the plugins refuse.
type Store struct {
	files      *Files
	getPath    func() (string, error)
	declared   settingsdomain.Declarations
	registries settingsdomain.Registries
}

// NewStore makes a store over the files. declared is nil where no plugin host runs: every plugin value
// then stands.
func NewStore(
	files *Files, getPath func() (string, error), declared settingsdomain.Declarations, registries settingsdomain.Registries,
) *Store {
	return &Store{files: files, getPath: getPath, declared: declared, registries: registries}
}

// ReadChecked returns the settings in force, or the defaults where the file is not there.
func (s *Store) ReadChecked() (settingsdomain.StoredSettings, error) {
	file, err := s.ReadCheckedFile()
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (s *Store) ReadCheckedFile() (File, error) {
	path, err := s.getPath()
	if err != nil {
		return File{}, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		file := DefaultFile()
		s.files.setLogLevel(file.LogLevel())
		return file, nil
	}
	return s.ReadCheckedAt(path)
}

// ReadUncheckedFile returns the settings as the file holds them, or the defaults where the file is not
// there.
func (s *Store) ReadUncheckedFile() (File, error) {
	path, err := s.getPath()
	if err != nil {
		return File{}, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return DefaultFile(), nil
	}
	return s.ReadUncheckedAt(path)
}

// ReadUnchecked returns the settings as the file holds them.
func (s *Store) ReadUnchecked() (settingsdomain.StoredSettings, error) {
	path, err := s.getPath()
	if err != nil {
		return nil, err
	}
	file, err := s.ReadUncheckedAt(path)
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// ReadUncheckedAt reads the file at path without leaving out any value.
func (s *Store) ReadUncheckedAt(path string) (File, error) {
	target := DefaultFile()
	if err := s.files.OpenAt(path, &target); err != nil {
		return File{}, err
	}
	return target, nil
}

// ReadCheckedAt reads the file at path and leaves out the values that no declaration accepts.
func (s *Store) ReadCheckedAt(path string) (File, error) {
	target, err := s.ReadUncheckedAt(path)
	if err != nil {
		return File{}, err
	}
	target.DropInvalidConnections()
	if s.declared != nil {
		target.DropInvalid(s.declared, s.registries)
	}
	s.files.setLogLevel(target.LogLevel())
	return target, nil
}

// ReadJSON returns the checked settings as JSON, the path of the file and the version of the state read.
// The version is taken before the file is read, so it is never newer than the content.
func (s *Store) ReadJSON() (settingsdomain.Snapshot, error) {
	snapshot := settingsdomain.Snapshot{Version: s.files.GetVersion()}
	path, err := s.getPath()
	if err != nil {
		return snapshot, err
	}
	snapshot.Path = path
	target, err := s.ReadCheckedAt(path)
	if err != nil {
		return snapshot, err
	}
	snapshot.Content, err = target.WriteJSON()
	return snapshot, err
}

// WriteJSON puts each setting at its path in the file and returns the checked settings as JSON that
// this write left, with the version it left them at, both taken under the hold of the lock that saved.
func (s *Store) WriteJSON(entries []settingsdomain.Setting) (content string, version uint64, err error) {
	path, err := s.getPath()
	if err != nil {
		return "", 0, err
	}
	into := DefaultFile()
	version, err = s.files.SaveVersioned(path, &into, convertWrites(entries)...)
	if err != nil {
		return "", 0, err
	}
	raiseVersion(&into)
	into.DropInvalidConnections()
	if s.declared != nil {
		into.DropInvalid(s.declared, s.registries)
	}
	s.files.setLogLevel(into.LogLevel())
	content, err = into.WriteJSON()
	if err != nil {
		return "", 0, fmt.Errorf("%w: %w", settingsdomain.ErrWrittenNotAnswered, err)
	}
	return content, version, nil
}

func convertWrites(entries []settingsdomain.Setting) []Setting {
	writes := make([]Setting, 0, len(entries))
	for _, one := range entries {
		writes = append(writes, Setting{At: one.Path, Written: json.RawMessage(one.JSON)})
	}
	return writes
}

// Write puts each setting at its path in the file. The value of a setting is JSON.
func (s *Store) Write(entries []settingsdomain.Setting) error {
	path, err := s.getPath()
	if err != nil {
		return err
	}
	into := DefaultFile()
	err = s.files.Save(path, &into, convertWrites(entries)...)
	if err == nil {
		s.files.setLogLevel(into.LogLevel())
	}
	return err
}

// Defaults are the settings of an unconfigured installation.
type Defaults struct{}

// GetDefaults returns the settings of an unconfigured installation.
func (Defaults) GetDefaults() settingsdomain.StoredSettings {
	file := DefaultFile()
	return &file
}

// GetConnectionsJSON is the connections section as JSON, and empty where the file holds no connection.
func (f *File) GetConnectionsJSON() (string, error) {
	if len(f.Connections) == 0 {
		return "", nil
	}
	return f.WriteConnectionsJSON()
}

type AppDeclaration struct{}

// GetAppDeclaration returns the settings the shell declares.
func (AppDeclaration) GetAppDeclaration() []manifest.Setting { return GetAppDeclaration() }

// JSONCodec converts the values of settings to and from JSON.
type JSONCodec struct{}

// Decode returns the value the JSON stands for.
func (JSONCodec) Decode(text string) (any, error) {
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return nil, err
	}
	return value, nil
}

// Encode returns the JSON of the value.
func (JSONCodec) Encode(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// ListConnectionIDs returns the ids of the valid connections of the file, in list order.
func (s *Store) ListConnectionIDs() ([]string, error) {
	stored, err := s.ReadUncheckedFile()
	if err != nil {
		return nil, err
	}
	stored.DropInvalidConnections()
	ids := make([]string, 0, len(stored.Connections))
	for _, one := range stored.Connections {
		ids = append(ids, one.ID)
	}
	return ids, nil
}
