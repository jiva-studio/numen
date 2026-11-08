package settings

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

func checkObject(raw []byte) error {
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(raw, &whole); err != nil {
		return fmt.Errorf("%w: numen.json: %s", settingsport.ErrNotASetting, describeErrorLocation(err))
	}
	if whole == nil {
		return fmt.Errorf("%w: numen.json: the file is not a JSON object", settingsport.ErrNotASetting)
	}
	return nil
}

func describeErrorLocation(err error) string {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return fmt.Sprintf("invalid JSON at byte %d", syntax.Offset)
	}

	var kind *json.UnmarshalTypeError
	if errors.As(err, &kind) {
		if kind.Field != "" {
			return fmt.Sprintf("%s has the wrong type at byte %d", kind.Field, kind.Offset)
		}
		return fmt.Sprintf("a value has the wrong type at byte %d", kind.Offset)
	}

	return err.Error()
}
