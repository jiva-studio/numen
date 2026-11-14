package settings

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// The reasons CheckValue refuses a value.
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

// RegistryThemes names the registry that lists the installed themes.
const RegistryThemes = "themes"

// RegistryConnections names the registry that lists the ids of the stored connections. A setting of
// rows takes a list of at most max objects, each with a connection from this registry, from
// RegistryDetectedConnections or detected for a type RegistryPendingDetections lists, and the row options of
// its type as text, and no connection with the same options twice. A registry that lists no connections leaves
// the connection unchecked.
const RegistryConnections = "connections"

// RegistryDetectedConnections names the registry that lists the ids of the connections the providers detected.
const RegistryDetectedConnections = "detectedConnections"

// RegistryPendingDetections names the registry that lists the ids of the types whose providers have not reported
// what they detect yet. A row that names a connection detected for such a type is kept until they do.
const RegistryPendingDetections = "pendingDetections"

// CheckValue returns why a value is not one the setting at the path takes. The first path element
// is the owner and the rest is the key path under it. It checks the declared key, the type, enum
// membership and range bounds, and for rows of connections the rows (see RegistryConnections).
func CheckValue(declared Declarations, registries Registries, path []string, value any) error {
	label := strings.Join(path, ".")
	if len(path) < 2 {
		return fmt.Errorf("%q: %w", label, ErrUndeclared)
	}
	owner, key := path[0], path[1:]
	settings, found := declared(owner)
	if !found {
		return noDeclaringPluginError{owner: owner}
	}
	if owner != AppOwner && len(key) == 1 && key[0] == HostFieldDiscover {
		return wrapAt(label, checkDiscover(settings, value))
	}
	name, err := getSettingName(owner, key, value)
	if err != nil || name == "" {
		return wrapAt(label, err)
	}
	index := slices.IndexFunc(settings, func(one manifest.Setting) bool { return one.Key == name })
	if index < 0 {
		return fmt.Errorf("%q: %w", label, ErrUndeclared)
	}
	return wrapAt(label, CheckDeclared(settings[index], registries, value))
}

func getSettingName(owner string, key []string, value any) (string, error) {
	if owner == AppOwner {
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
	if name != HostFieldEnabled {
		return false, nil
	}
	if _, isBool := value.(bool); !isBool && value != nil {
		return true, fmt.Errorf("%w: expected true or false", ErrWrongType)
	}
	return true, nil
}

// CheckDeclared returns why a value is not one the declared setting takes: the type, enum membership,
// range bounds and, for rows of connections, the rows. A nil value is taken.
func CheckDeclared(declared manifest.Setting, registries Registries, value any) error {
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
