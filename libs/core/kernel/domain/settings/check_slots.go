package settings

import "fmt"

// checkSlotLayout returns why a value is not a slot layout: an object by slot name, each a list of
// objects that hold the id of an entry and, optionally, whether it is hidden.
func checkSlotLayout(value any) error {
	layout, isObject := value.(map[string]any)
	if !isObject {
		return fmt.Errorf("%w: expected an object by slot name", ErrWrongType)
	}
	for slot, list := range layout {
		entries, isList := list.([]any)
		if !isList {
			return fmt.Errorf("%w: slot %q expects a list of entries", ErrWrongType, slot)
		}
		for _, one := range entries {
			if err := checkSlotLayoutEntry(slot, one); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkSlotLayoutEntry(slot string, value any) error {
	entry, isObject := value.(map[string]any)
	if !isObject {
		return fmt.Errorf("%w: an entry of slot %q expects an object", ErrWrongType, slot)
	}
	if id, isText := entry["id"].(string); !isText || id == "" {
		return fmt.Errorf("%w: an entry of slot %q expects an id", ErrWrongType, slot)
	}
	if hidden, has := entry["hidden"]; has {
		if _, isBool := hidden.(bool); !isBool {
			return fmt.Errorf("%w: hidden of an entry of slot %q expects true or false", ErrWrongType, slot)
		}
	}
	return nil
}
