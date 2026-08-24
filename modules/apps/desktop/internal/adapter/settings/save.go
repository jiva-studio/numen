package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Setting is one field of the file and where in it that field sits:
// `appearance.theme` is `Setting{At: []string{"appearance", "theme"}}`.
type Setting struct {
	At    []string
	Value any
}

// Save writes settings into the file, leaving everything else in it where it
// was.
//
// The file is read as an object, the named fields are set in that object, and
// it is written back. A field this build knows nothing about — a key a person
// typed, a section a later build reads — comes through the write unread and
// unchanged.
//
// A file that does not parse is not written: what a person has in it is worth
// more than the setting being saved. A file that is not there is written
// holding the named fields alone, and every field a settings file leaves out
// keeps its default.
func Save(path string, settings ...Setting) error {
	file := map[string]any{}
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(raw, &file); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}

	for _, setting := range settings {
		if err := set(file, setting); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}

	body, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return replace(path, append(body, '\n'))
}

// set puts one value where its field sits, making the sections on the way to
// it. A name the file holds as something other than a section is left as it
// is, and the setting is refused.
func set(file map[string]any, setting Setting) error {
	if len(setting.At) == 0 {
		return errors.New("a setting with no name")
	}
	at := file
	for _, name := range setting.At[:len(setting.At)-1] {
		held, there := at[name]
		if !there || held == nil {
			made := map[string]any{}
			at[name] = made
			at = made
			continue
		}
		section, ok := held.(map[string]any)
		if !ok {
			return fmt.Errorf("%s holds %T, and a setting goes inside a section", name, held)
		}
		at = section
	}
	at[setting.At[len(setting.At)-1]] = setting.Value
	return nil
}

// replace writes the file beside itself and renames it over the top, so a
// machine that dies mid-write leaves the settings whole. The mode is the
// person's alone: they type their service keys into this file.
func replace(path string, content []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
