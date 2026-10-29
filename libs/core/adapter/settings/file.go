package settings

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// checkObject rejects bytes that are not a single JSON object, since the
// settings are the fields of one object, and accepts them otherwise.
func checkObject(raw []byte) error {
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(raw, &whole); err != nil {
		return fmt.Errorf("%w: %s", settingsport.ErrNotASetting, describeErrorLocation(err))
	}
	if whole == nil {
		return fmt.Errorf("%w: the settings are the fields of one object", settingsport.ErrNotASetting)
	}
	return nil
}

// describeErrorLocation describes a settings file error by its position and by
// the field it occurs in.
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
