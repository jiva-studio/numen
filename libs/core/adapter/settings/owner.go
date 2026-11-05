package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// OwnerApp is the owner of the settings of the shell. Every other first-level key of the file
// other than the version is the id of a plugin.
const OwnerApp = "app"

// The fields of the object of a plugin that the host keeps. They are no settings of the manifest:
// enabled holds a boolean, and storedSecrets the names of the secrets in the keychain.
const (
	fieldEnabled       = "enabled"
	fieldStoredSecrets = "storedSecrets"
)

// The reasons CheckWrite rejects a write.
var (
	// ErrUndeclared is a key that its owner does not declare, or an owner that has no declaration.
	ErrUndeclared = errors.New("the setting is not declared")
	// ErrWrongType is a value of another type than the declaration states.
	ErrWrongType = errors.New("the value is of the wrong type")
	// ErrNotAnOption is a value that is none of the options of an enum setting.
	ErrNotAnOption = errors.New("the value is none of the options")
	// ErrOutOfRange is a number outside the bounds of a range or a number setting.
	ErrOutOfRange = errors.New("the number is outside the bounds")
)

// Declarations returns the settings an owner declares, and false for an owner that has no declaration.
type Declarations func(owner string) ([]pluginport.PluginManifestSetting, bool)

// Registries returns the values a registry lists, such as the installed themes, and false for a
// registry that has no provider.
type Registries func(name string) ([]string, bool)

// CheckWrite takes the declarations of the owners, the registries, the path of the setting and the
// value. It returns why the value is not one the setting at the path takes. The first element of
// the path is the owner and the rest is the key path under it, as the file lays them out. One
// function checks a write for every owner: the declared key, the type of the value, the
// membership of an enum and the bounds of a range. The keys `agent` and `ai` of the owner app are
// outside the check.
func CheckWrite(declared Declarations, registries Registries, path []string, value any) error {
	label := strings.Join(path, ".")
	if len(path) < 2 {
		return fmt.Errorf("%w: %q names no key of an owner", ErrUndeclared, label)
	}
	owner, key := path[0], path[1:]
	settings, found := declared(owner)
	if !found {
		return fmt.Errorf("%w: %q has no owner that declares settings", ErrUndeclared, label)
	}
	name, err := getSettingName(owner, key, value)
	if err != nil || name == "" {
		return wrapAt(label, err)
	}
	index := slices.IndexFunc(settings, func(one pluginport.PluginManifestSetting) bool { return one.Key == name })
	if index < 0 {
		return fmt.Errorf("%w: %q", ErrUndeclared, label)
	}
	return wrapAt(label, checkValue(settings[index], registries, value))
}

// getSettingName returns the key a declaration lists the setting under: the dotted path for the owner
// app, and the one segment for a plugin. It returns no name, and no error, for a path that is outside
// the check or that leads to a field the host keeps.
func getSettingName(owner string, key []string, value any) (string, error) {
	if owner == OwnerApp {
		if isOutsideDeclaration(key[0]) {
			return "", nil
		}
		return strings.Join(key, "."), nil
	}
	if len(key) != 1 {
		return "", fmt.Errorf("%w: a key of a plugin is one segment", ErrUndeclared)
	}
	if reserved, err := checkHostField(key[0], value); reserved {
		return "", err
	}
	return key[0], nil
}

// wrapAt names the setting an error is about.
func wrapAt(label string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", label, err)
}

// checkHostField checks a field the host keeps in the object of a plugin, and reports whether the
// name is one.
func checkHostField(name string, value any) (reserved bool, err error) {
	switch name {
	case fieldEnabled:
		if _, isBool := value.(bool); !isBool && value != nil {
			return true, fmt.Errorf("%w: enabled holds true or false", ErrWrongType)
		}
		return true, nil
	case fieldStoredSecrets:
		return true, nil
	}
	return false, nil
}

// checkValue returns why a declared setting does not accept the value. A null value resets the
// setting and is always taken.
func checkValue(declared pluginport.PluginManifestSetting, registries Registries, value any) error {
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
			return fmt.Errorf("%w: expected text", ErrWrongType)
		}
	case "number", "range":
		return checkNumber(declared, value)
	case "enum":
		return checkOption(declared, registries, value)
	}
	return nil
}

// checkNumber checks a number against the bounds the declaration gives.
func checkNumber(declared pluginport.PluginManifestSetting, value any) error {
	number, isNumber := toNumber(value)
	if !isNumber || math.IsNaN(number) || math.IsInf(number, 0) {
		return fmt.Errorf("%w: expected a number", ErrWrongType)
	}
	if (declared.Min != nil && number < *declared.Min) || (declared.Max != nil && number > *declared.Max) {
		return fmt.Errorf("%w: %v goes from %s to %s", ErrOutOfRange, number, formatLimit(declared.Min), formatLimit(declared.Max))
	}
	return nil
}

func formatLimit(limit *float64) string {
	if limit == nil {
		return "no bound"
	}
	return fmt.Sprint(*limit)
}

// toNumber returns the value as a float where it is a number of any Go type.
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

// checkOption checks that the value is one of the options of an enum: its own list, or the values
// of the registry it lists them from. Options that follow another setting take any text here.
func checkOption(declared pluginport.PluginManifestSetting, registries Registries, value any) error {
	text, isText := value.(string)
	if !isText {
		return fmt.Errorf("%w: expected one of the options", ErrWrongType)
	}
	switch {
	case declared.OptionsFrom != "":
		if registries == nil {
			return nil
		}
		names, found := registries(declared.OptionsFrom)
		if found && !slices.Contains(names, text) {
			return fmt.Errorf("%w: %q is not in %s", ErrNotAnOption, text, declared.OptionsFrom)
		}
	case declared.OptionsBy != nil:
	default:
		if _, isOption := declared.Options.Find(text); !isOption {
			return fmt.Errorf("%w: %q", ErrNotAnOption, text)
		}
	}
	return nil
}

// GetOwner returns the values stored under an owner, the plugin id or OwnerApp.
func (f File) GetOwner(owner string) map[string]any {
	if owner != OwnerApp {
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
	if app, isObject := root[OwnerApp].(map[string]any); isObject {
		return app
	}
	return nil
}

// GetValue returns the value in force at a path: the stored value, else the default its owner
// declares. The first element of the path is the owner.
func (f File) GetValue(path ...string) (any, bool) {
	if len(path) < 2 {
		return nil, false
	}
	if value, found := lookupPath(f.GetOwner(path[0]), path[1:]); found && value != nil {
		return value, true
	}
	if path[0] == OwnerApp {
		if value := getAppDefault(strings.Join(path[1:], ".")); value != nil {
			return value, true
		}
	}
	return nil, false
}

// DropInvalid leaves out of force every stored value that CheckWrite refuses for the declarations
// given: an undeclared key, a value of the wrong type or outside its bounds, a value that is no
// option, and an object of an owner that has no declaration. Each one adds a notice to the file
// and is applied nowhere. The bytes of the file on disk are not touched.
func (f *File) DropInvalid(declared Declarations, registries Registries) {
	f.dropInvalidApp(declared, registries)
	for _, id := range slices.Sorted(maps.Keys(f.Plugins)) {
		if _, found := declared(id); !found {
			delete(f.Plugins, id)
			f.AddNotice(fmt.Sprintf("%s is the object of no plugin that is installed, and is ignored", id))
			continue
		}
		for _, key := range slices.Sorted(maps.Keys(f.Plugins[id])) {
			if key == fieldStoredSecrets && !isListOfTexts(f.Plugins[id][key]) {
				delete(f.Plugins[id], key)
				f.AddNotice(fmt.Sprintf("%s.%s: wrong type, expected a list of the names of the secrets, and is ignored", id, key))
				continue
			}
			if err := CheckWrite(declared, registries, []string{id, key}, f.Plugins[id][key]); err != nil {
				delete(f.Plugins[id], key)
				f.AddNotice(fmt.Sprintf("%v, and is ignored", err))
			}
		}
	}
}

// dropInvalidApp puts the default in force of every setting of the shell whose value the check
// with the registries refuses.
func (f *File) dropInvalidApp(declared Declarations, registries Registries) {
	for _, one := range readAppDeclaration() {
		path := append([]string{OwnerApp}, strings.Split(one.Key, ".")...)
		value, found := f.GetValue(path...)
		if !found || value == one.Default {
			continue
		}
		if err := CheckWrite(declared, registries, path, value); err != nil {
			f.resetApp(one)
			f.AddNotice(fmt.Sprintf("%v, and the default %v stands", err, one.Default))
		}
	}
}

// resetApp puts the default of a setting of the shell in force.
func (f *File) resetApp(declared pluginport.PluginManifestSetting) {
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
	case keyWindowButtons:
		f.Window = DefaultWindow()
	case keyProfile:
		f.Performance = DefaultPerformance()
	}
}

// isListOfTexts reports whether a decoded value is a list whose items are all text.
func isListOfTexts(value any) bool {
	items, isList := value.([]any)
	if !isList {
		return false
	}
	for _, item := range items {
		if _, isText := item.(string); !isText {
			return false
		}
	}
	return true
}
