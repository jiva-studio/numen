package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Read hands back the settings file as its person wrote it: every byte of it,
// in the order they arranged it. A file that is not there reads as an empty
// object, which is what an installation nobody has configured runs on.
func Read(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return []byte("{}\n"), nil
	}
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// Write replaces the settings file whole, with the bytes as they were typed.
//
// Bytes the settings cannot be read out of are refused and the file is left as
// it was. What is said names where in the file the trouble is; the file holds a
// person's keys, and nothing standing in it is repeated back.
//
// Seen is the file as the caller last read it. A file standing at anything else
// is left alone and port.ErrChanged comes back: someone turning a setting in
// their own window outranks a caller that read the file, thought about it, and
// arrived late. Nil is a caller that compares nothing, and its bytes land.
func Write(path string, raw []byte, seen *string) error {
	if err := object(raw); err != nil {
		return err
	}
	if err := distinct(raw); err != nil {
		return fmt.Errorf("%w: %s", port.ErrNotASetting, where(err))
	}
	if err := holds(raw); err != nil {
		return fmt.Errorf("%w: %s", port.ErrNotASetting, where(err))
	}

	return reaching(path, func(path string) error {
		if err := stands(path, seen); err != nil {
			return err
		}
		return replace(path, raw)
	})
}

// stands says whether the file holds what its caller last read, and nothing
// where the caller presents nothing. It is read the way Read reads it, so a
// file that is not there stands at the empty object a caller was given.
func stands(path string, seen *string) error {
	if seen == nil {
		return nil
	}
	held, err := Read(path)
	if err != nil {
		return err
	}
	if string(held) != *seen {
		return port.ErrChanged
	}
	return nil
}

// object says what is wrong with bytes that are not the one object the
// settings are the fields of, and nothing where they are it.
//
// A bare `null` unmarshals into anything and leaves it alone, so bytes that
// came back nought are refused by name rather than read as an installation
// nobody has configured.
func object(raw []byte) error {
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(raw, &whole); err != nil {
		return fmt.Errorf("%w: %s", port.ErrNotASetting, where(err))
	}
	if whole == nil {
		return fmt.Errorf("%w: the settings are the fields of one object", port.ErrNotASetting)
	}
	return nil
}

// where says what is wrong with a settings file by the place it goes wrong at,
// and by the field it goes wrong in.
func where(err error) string {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return fmt.Sprintf("it does not read as JSON, at byte %d", syntax.Offset)
	}

	var kind *json.UnmarshalTypeError
	if errors.As(err, &kind) {
		if kind.Field != "" {
			return fmt.Sprintf("%s is of the wrong kind, at byte %d", kind.Field, kind.Offset)
		}
		return fmt.Sprintf("a value is of the wrong kind, at byte %d", kind.Offset)
	}

	return err.Error()
}
