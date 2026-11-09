package settings

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// keyLayout is the section of the owner app that holds the user's arrangement of the bars.
const keyLayout = "layout"

// keyLayoutSlots is the member of the layout that maps a bar slot to its entries.
const keyLayoutSlots = "slots"

// Layout is the user's arrangement of the bar slots. The window writes it when the user moves an
// entry of the title bar or the status bar by Cmd/Ctrl+drag.
//
//numen:settings
type Layout struct {
	// Slots maps a bar slot to the keys of its entries in drawing order; a key is
	// <plugin id>/<view or command id>. An entry no slot lists is drawn where its plugin places it.
	Slots map[string][]string `json:"slots,omitempty"`
}

var errNotAKeyList = errors.New("expected a list of entry keys")

// readLayout returns the part of a layout in the file that is in force. A slot that is not a bar slot,
// a value that is not a list of strings, and a key that names no owner are left out, each with a
// notice.
func (f *File) readLayout(value any) Layout {
	label := OwnerApp + "." + keyLayout
	object, isObject := value.(map[string]any)
	if !isObject {
		f.AddNotice(fmt.Sprintf("numen.json: %q: %v; the value is ignored", label, fmt.Errorf("%w: expected an object", ErrWrongType)))
		return Layout{}
	}
	for _, member := range slices.Sorted(maps.Keys(object)) {
		if member != keyLayoutSlots {
			f.AddNotice(fmt.Sprintf("numen.json: %q: no such setting; the value is ignored", label+"."+member))
		}
	}
	stored, found := object[keyLayoutSlots]
	if !found || stored == nil {
		return Layout{}
	}
	return Layout{Slots: f.readLayoutSlots(label+"."+keyLayoutSlots, stored)}
}

func (f *File) readLayoutSlots(label string, value any) map[string][]string {
	slots, isObject := value.(map[string]any)
	if !isObject {
		f.AddNotice(fmt.Sprintf("numen.json: %q: %v; the value is ignored", label, fmt.Errorf("%w: expected an object", ErrWrongType)))
		return nil
	}
	inForce := map[string][]string{}
	for _, slot := range slices.Sorted(maps.Keys(slots)) {
		if keys, isInForce := f.readLayoutList(label+"."+slot, slot, slots[slot]); isInForce {
			inForce[slot] = keys
		}
	}
	return inForce
}

func (f *File) readLayoutList(label, slot string, value any) ([]string, bool) {
	if !manifest.IsBarSlot(slot) {
		f.AddNotice(fmt.Sprintf("numen.json: %q: %v; the value is ignored", label, fmt.Errorf("%w: not a bar slot", ErrUndeclared)))
		return nil, false
	}
	list, err := readKeyList(value)
	if err != nil {
		f.AddNotice(fmt.Sprintf("numen.json: %q: %v; the value is ignored", label, err))
		return nil, false
	}
	keys := make([]string, 0, len(list))
	for _, key := range list {
		if err := checkLayoutKey(key); err != nil {
			f.AddNotice(fmt.Sprintf("numen.json: %q: %v; the value is ignored", label, err))
			continue
		}
		keys = append(keys, key)
	}
	return keys, true
}

// checkLayoutWrite returns why a value is not one the path under app.layout takes: the layout
// object with only slots, the map of bar slots, or the list of one bar slot. A list holds valid keys,
// none twice.
func checkLayoutWrite(path []string, value any) error {
	switch {
	case len(path) == 0:
		return checkLayoutObject(value)
	case path[0] != keyLayoutSlots:
		return ErrUndeclared
	case len(path) == 1:
		return checkLayoutSlots(value)
	case len(path) == 2 && manifest.IsBarSlot(path[1]):
		return checkLayoutList(value)
	}
	return ErrUndeclared
}

func checkLayoutObject(value any) error {
	if value == nil {
		return nil
	}
	object, isObject := value.(map[string]any)
	if !isObject {
		return fmt.Errorf("%w: expected an object", ErrWrongType)
	}
	for member, nested := range object {
		if member != keyLayoutSlots {
			return fmt.Errorf("%w: %s", ErrUndeclared, member)
		}
		if err := checkLayoutSlots(nested); err != nil {
			return err
		}
	}
	return nil
}

func checkLayoutSlots(value any) error {
	if value == nil {
		return nil
	}
	slots, isObject := value.(map[string]any)
	if !isObject {
		return fmt.Errorf("%w: expected an object", ErrWrongType)
	}
	for slot, list := range slots {
		if !manifest.IsBarSlot(slot) {
			return fmt.Errorf("%w: %s is not a bar slot", ErrUndeclared, slot)
		}
		if err := checkLayoutList(list); err != nil {
			return err
		}
	}
	return nil
}

func checkLayoutList(value any) error {
	if value == nil {
		return nil
	}
	keys, err := readKeyList(value)
	if err != nil {
		return err
	}
	for index, key := range keys {
		if err := checkLayoutKey(key); err != nil {
			return err
		}
		if slices.Contains(keys[:index], key) {
			return fmt.Errorf("%w: %q is listed twice", ErrWrongType, key)
		}
	}
	return nil
}

func readKeyList(value any) ([]string, error) {
	list, isList := value.([]any)
	if !isList {
		return nil, fmt.Errorf("%w: %w", ErrWrongType, errNotAKeyList)
	}
	keys := make([]string, 0, len(list))
	for _, one := range list {
		key, isText := one.(string)
		if !isText {
			return nil, fmt.Errorf("%w: %w", ErrWrongType, errNotAKeyList)
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// checkLayoutKey returns why a key names no entry: a key is <owner>/<id>, the owner being app or a
// plugin id. A plugin that is not installed is a valid owner.
func checkLayoutKey(key string) error {
	owner, id, found := strings.Cut(key, "/")
	if !found || id == "" || (owner != OwnerApp && !manifest.IsValidID(owner)) {
		return fmt.Errorf("%w: %q is not <plugin id>/<id>", ErrWrongType, key)
	}
	return nil
}
