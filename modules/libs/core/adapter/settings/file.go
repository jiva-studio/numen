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
// it was. What is said names where in the file the trouble is, and nothing
// standing in it is repeated back.
//
// Seen is the file as the caller last read it. A file standing at anything else
// is left alone and port.ErrStale comes back: someone turning a setting in
// their own window outranks a caller that read the file, thought about it, and
// arrived late. Nil is a caller that compares nothing, and its bytes land.
// into says what shape the whole file takes, so bytes the settings cannot be
// read out of are refused before any of them land.
func Write(path string, raw []byte, baseline *string, into Document) error {
	if err := checkObject(raw); err != nil {
		return err
	}
	if err := checkNoDuplicateNames(raw); err != nil {
		return fmt.Errorf("%w: %s", port.ErrNotASetting, describeErrorLocation(err))
	}
	if err := checkSettings(raw, into); err != nil {
		return fmt.Errorf("%w: %s", port.ErrNotASetting, describeErrorLocation(err))
	}

	return runOnFile(path, func(path string) error {
		if err := checkUnchanged(path, baseline); err != nil {
			return err
		}
		return writeFileAtomically(path, raw)
	})
}

// checkUnchanged says whether the file holds what its caller last read, and
// nothing where the caller presents nothing. It is read the way Read reads it,
// so a file that is not there checkUnchanged at the empty object a caller was
// given.
func checkUnchanged(path string, baseline *string) error {
	if baseline == nil {
		return nil
	}
	held, err := Read(path)
	if err != nil {
		return err
	}
	if string(held) != *baseline {
		return port.ErrStale
	}
	return nil
}

// checkObject says what is wrong with bytes that are not the one checkObject the
// settings are the fields of, and nothing where they are it.
//
// A bare `null` unmarshals into anything and leaves it alone, so it is refused
// by name.
func checkObject(raw []byte) error {
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(raw, &whole); err != nil {
		return fmt.Errorf("%w: %s", port.ErrNotASetting, describeErrorLocation(err))
	}
	if whole == nil {
		return fmt.Errorf("%w: the settings are the fields of one object", port.ErrNotASetting)
	}
	return nil
}

// describeErrorLocation says what is wrong with a settings file by the place it goes wrong at,
// and by the field it goes wrong in.
func describeErrorLocation(err error) string {
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
