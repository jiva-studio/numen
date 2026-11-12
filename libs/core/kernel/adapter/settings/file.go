package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

func checkObject(raw []byte) error {
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(raw, &whole); err != nil {
		return fmt.Errorf("%w: numen.json: %s", settingsdomain.ErrNotASetting, describeErrorLocation(err))
	}
	if whole == nil {
		return fmt.Errorf("%w: numen.json: the file is not a JSON object", settingsdomain.ErrNotASetting)
	}
	return nil
}

func describeErrorLocation(err error) string {
	if syntax, ok := errors.AsType[*json.SyntaxError](err); ok {
		return fmt.Sprintf("invalid JSON at byte %d", syntax.Offset)
	}

	if kind, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		if kind.Field != "" {
			return fmt.Sprintf("%s has the wrong type at byte %d", kind.Field, kind.Offset)
		}
		return fmt.Sprintf("a value has the wrong type at byte %d", kind.Offset)
	}

	return err.Error()
}
