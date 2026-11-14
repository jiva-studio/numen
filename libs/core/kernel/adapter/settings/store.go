package settings

import (
	"encoding/json"
	"errors"
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
	declared   Declarations
	registries Registries
}

// NewStore makes a store over the files. declared is nil where no plugin host runs: every plugin value
// then stands.
func NewStore(
	files *Files, getPath func() (string, error), declared Declarations, registries Registries,
) *Store {
	return &Store{files: files, getPath: getPath, declared: declared, registries: registries}
}

// ReadChecked returns the settings in force, or the defaults where the file is not there.
func (s *Store) ReadChecked() (settingsdomain.Stored, error) {
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
		return DefaultFile(), nil
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
func (s *Store) ReadUnchecked() (settingsdomain.Stored, error) {
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
	return target, nil
}

// ReadJSON returns the checked settings as JSON and the path of the file.
func (s *Store) ReadJSON() (content, path string, err error) {
	path, err = s.getPath()
	if err != nil {
		return "", "", err
	}
	target, err := s.ReadCheckedAt(path)
	if err != nil {
		return "", path, err
	}
	content, err = target.WriteJSON()
	return content, path, err
}

// Write puts each setting at its path in the file. The value of a setting is JSON.
func (s *Store) Write(entries []settingsdomain.Setting) error {
	writes := make([]Setting, 0, len(entries))
	for _, one := range entries {
		writes = append(writes, Setting{At: one.Path, Written: json.RawMessage(one.JSON)})
	}
	path, err := s.getPath()
	if err != nil {
		return err
	}
	into := DefaultFile()
	return s.files.Save(path, &into, writes...)
}

// Defaults are the settings of an unconfigured installation.
type Defaults struct{}

// GetDefaults returns the settings of an unconfigured installation.
func (Defaults) GetDefaults() settingsdomain.Stored {
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

// Validator checks a value against the declaration of the setting it is written to.
type Validator struct {
	declared   Declarations
	registries Registries
}

func NewValidator(declared Declarations, registries Registries) Validator {
	return Validator{declared: declared, registries: registries}
}

// CheckWrite returns the reason the value is refused, and nil where it is accepted.
func (v Validator) CheckWrite(path []string, value any) error {
	return CheckWrite(v.declared, v.registries, path, value)
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
