package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// The fields of the object of a plugin that the host keeps. They are no settings of the manifest:
// enabled holds a boolean.
const fieldEnabled = settingsdomain.HostFieldEnabled

// The reasons CheckWrite rejects a write.
var (
	// ErrUndeclared is a key that its owner does not declare, or an owner that has no declaration.
	ErrUndeclared = errors.New("no such setting")
	// ErrWrongType is a value of another type than the declaration states.
	ErrWrongType = errors.New("wrong type")
	// ErrNotAnOption is a value that is none of the options of an enum setting.
	ErrNotAnOption = errors.New("not one of the options")
	// ErrOutOfRange is a number outside the bounds of a range or a number setting.
	ErrOutOfRange = errors.New("out of range")
)

// noDeclaringPluginError is returned for an owner that no plugin declares settings under. It
// wraps ErrUndeclared.
type noDeclaringPluginError struct{ owner string }

func (e noDeclaringPluginError) Error() string {
	return fmt.Sprintf("%q: no plugin declares settings under this name", e.owner)
}

func (e noDeclaringPluginError) Is(target error) bool { return target == ErrUndeclared }

// Declarations returns the settings an owner declares, and false for an owner that has no declaration.
type Declarations func(owner string) ([]manifest.Setting, bool)

// Registries returns the values a registry lists, such as the installed themes, and false for a
// registry that has no provider.
type Registries func(name string) ([]string, bool)

// CheckWrite returns why a value is not one the setting at the path takes. The first path element
// is the owner and the rest is the key path under it. It checks the declared key, the type, enum
// membership and range bounds, and for rows of connections the rows (see RegistryConnections).
func CheckWrite(declared Declarations, registries Registries, path []string, value any) error {
	label := strings.Join(path, ".")
	if len(path) < 2 {
		return fmt.Errorf("%q: %w", label, ErrUndeclared)
	}
	owner, key := path[0], path[1:]
	settings, found := declared(owner)
	if !found {
		return noDeclaringPluginError{owner: owner}
	}
	name, err := getSettingName(owner, key, value)
	if err != nil || name == "" {
		return wrapAt(label, err)
	}
	index := slices.IndexFunc(settings, func(one manifest.Setting) bool { return one.Key == name })
	if index < 0 {
		return fmt.Errorf("%q: %w", label, ErrUndeclared)
	}
	return wrapAt(label, checkValue(settings[index], registries, value))
}

func getSettingName(owner string, key []string, value any) (string, error) {
	if owner == settingsdomain.AppOwner {
		return strings.Join(key, "."), nil
	}
	if len(key) != 1 {
		return "", fmt.Errorf("%w: a plugin setting key is a single name", ErrUndeclared)
	}
	if reserved, err := checkHostField(key[0], value); reserved {
		return "", err
	}
	return key[0], nil
}

func wrapAt(label string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%q: %w", label, err)
}

func checkHostField(name string, value any) (reserved bool, err error) {
	if name != fieldEnabled {
		return false, nil
	}
	if _, isBool := value.(bool); !isBool && value != nil {
		return true, fmt.Errorf("%w: expected true or false", ErrWrongType)
	}
	return true, nil
}

func checkValue(declared manifest.Setting, registries Registries, value any) error {
	if value == nil {
		return nil
	}
	switch declared.Type {
	case "boolean":
		if _, isBool := value.(bool); !isBool {
			return fmt.Errorf("%w: expected true or false", ErrWrongType)
		}
	case "string", "file":
		if _, isText := value.(string); !isText {
			return fmt.Errorf("%w: expected a string", ErrWrongType)
		}
	case "number", "range":
		return checkNumber(declared, value)
	case "enum":
		return checkOption(declared, registries, value)
	case manifest.SettingTypeConnections:
		return checkRows(declared, registries, value)
	}
	return nil
}

func checkNumber(declared manifest.Setting, value any) error {
	number, isNumber := toNumber(value)
	if !isNumber || math.IsNaN(number) || math.IsInf(number, 0) {
		return fmt.Errorf("%w: expected a number", ErrWrongType)
	}
	if (declared.Min != nil && number < *declared.Min) || (declared.Max != nil && number > *declared.Max) {
		return fmt.Errorf("%v is %w %s..%s", number, ErrOutOfRange, formatLimit(declared.Min, "-inf"), formatLimit(declared.Max, "+inf"))
	}
	return nil
}

func formatLimit(limit *float64, unbounded string) string {
	if limit == nil {
		return unbounded
	}
	return fmt.Sprint(*limit)
}

func toNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	}
	return 0, false
}

func checkOption(declared manifest.Setting, registries Registries, value any) error {
	text, isText := value.(string)
	if !isText {
		return fmt.Errorf("%w: expected a string", ErrWrongType)
	}
	switch {
	case declared.OptionsFrom != "":
		if registries == nil {
			return nil
		}
		names, found := registries(declared.OptionsFrom)
		if found && !slices.Contains(names, text) {
			return fmt.Errorf("%q is %w listed in %s", text, ErrNotAnOption, declared.OptionsFrom)
		}
	case declared.OptionsBy != nil:
	default:
		if _, isOption := declared.Options.Find(text); !isOption {
			return fmt.Errorf("%q is %w", text, ErrNotAnOption)
		}
	}
	return nil
}

// GetOwner returns the values stored under an owner, the plugin id or OwnerApp.
func (f *File) GetOwner(owner string) map[string]any {
	if owner != settingsdomain.AppOwner {
		return maps.Clone(f.Plugins[owner])
	}
	content, err := f.MarshalJSON()
	if err != nil {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal(content, &root); err != nil {
		return nil
	}
	if app, isObject := root[settingsdomain.AppOwner].(map[string]any); isObject {
		return app
	}
	return nil
}

// GetValue returns the value in force at a path: the stored value, else the default its owner
// declares. The first element of the path is the owner.
func (f *File) GetValue(path ...string) (any, bool) {
	if len(path) < 2 {
		return nil, false
	}
	if value, found := lookupPath(f.GetOwner(path[0]), path[1:]); found && value != nil {
		return value, true
	}
	if path[0] == settingsdomain.AppOwner {
		if value := getAppDefault(strings.Join(path[1:], ".")); value != nil {
			return value, true
		}
	}
	return nil, false
}

// DropInvalid leaves out of force every stored value that CheckWrite refuses for the declarations
// given: an undeclared key, a value of the wrong type or outside its bounds, a value that is no
// option, and an object of an owner that has no declaration. In a setting of type connections only
// the rows that fail are left out, and the rows beyond max, each with a notice. Each one adds a notice to the file
// and is applied nowhere. The bytes of the file on disk are not touched.
func (f *File) DropInvalid(declared Declarations, registries Registries) {
	registries = withConnectionIDs(f.Connections, registries)
	f.dropInvalidApp(declared, registries)
	for _, id := range slices.Sorted(maps.Keys(f.Plugins)) {
		settings, found := declared(id)
		if !found {
			delete(f.Plugins, id)
			f.AddNotice(fmt.Sprintf("numen.json: no installed plugin %s; its settings are ignored", id))
			continue
		}
		dropInvalidRowsOf(f, id, settings)
		for _, key := range slices.Sorted(maps.Keys(f.Plugins[id])) {
			if err := CheckWrite(declared, registries, []string{id, key}, f.Plugins[id][key]); err != nil {
				delete(f.Plugins[id], key)
				f.AddNotice(fmt.Sprintf("numen.json: %v; the value is ignored", err))
			}
		}
	}
}

func (f *File) dropInvalidApp(declared Declarations, registries Registries) {
	for _, one := range readAppDeclaration() {
		path := append([]string{settingsdomain.AppOwner}, strings.Split(one.Key, ".")...)
		value, found := f.GetValue(path...)
		if !found || value == one.Default {
			continue
		}
		if err := CheckWrite(declared, registries, path, value); err != nil {
			f.resetApp(one)
			f.AddNotice(fmt.Sprintf("numen.json: %v; the default %v applies", err, one.Default))
		}
	}
}

func (f *File) resetApp(declared manifest.Setting) {
	switch declared.Key {
	case keyTheme:
		f.Appearance.Theme = getAppString(keyTheme)
	case keyMode:
		f.Appearance.Mode = getAppString(keyMode)
	case keyInterfaceScale:
		f.Appearance.InterfaceScale = getAppNumber(keyInterfaceScale)
	case keyTextScale:
		f.Appearance.TextScale = getAppNumber(keyTextScale)
	case keySyncNaming:
		f.Titles = DefaultTitles()
	case keyProfile:
		f.Performance = DefaultPerformance()
	}
}

// RegistryOptions lists the options a registry offers to the settings that draw on it, and false for
// a registry that is not known.
type RegistryOptions func(name string) (manifest.SettingOptions, bool)

func (r RegistryOptions) ListRegistryOptions(name string) (manifest.SettingOptions, bool) {
	return r(name)
}
