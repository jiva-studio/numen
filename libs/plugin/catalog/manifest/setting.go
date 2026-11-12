package manifest

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// SettingOption is one value an enum setting can hold. Its When is a condition over the variables of
// when.json that the window evaluates; empty offers the option always.
type SettingOption struct {
	Value        string
	Title        string
	Detail       string
	Dependencies []string
	When         string
}

// GetTitle returns the title of the option, or its value where it has none.
func (o SettingOption) GetTitle() string {
	if o.Title != "" {
		return o.Title
	}
	return o.Value
}

// SettingOptions are the values an enum setting can hold, in the order they are offered.
type SettingOptions []SettingOption

func (o SettingOptions) Find(value string) (SettingOption, bool) {
	for _, one := range o {
		if one.Value == value {
			return one, true
		}
	}
	return SettingOption{}, false
}

// OptionsBy gives a setting its options from the value of another setting.
type OptionsBy struct {
	Setting string
	Options map[string]SettingOptions
}

// Condition holds while a setting holds one of the values listed.
type Condition struct {
	Setting string
	Is      []string
}

// SecretDeclaration is a secret declared by a plugin in its manifest.
type SecretDeclaration struct {
	Name        string
	Title       string
	Description string
	When        *Condition
	Group       string
}

// GetSecretNames returns the names of the secrets the plugin declares.
func (m Manifest) GetSecretNames() []string {
	names := make([]string, 0, len(m.Secrets))
	for _, secret := range m.Secrets {
		names = append(names, secret.Name)
	}
	return names
}

func (m Manifest) GetSetting(key string) (Setting, bool) {
	for _, s := range m.Settings {
		if s.Key == key {
			return s, true
		}
	}
	return Setting{}, false
}

// FormatSettingValue is the value as a condition and a list of options compare it.
func FormatSettingValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	// gomutants:disable-next-line BRANCH_CASE reason="string case avoids fmt.Sprint allocation"
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return fmt.Sprint(value)
}

// GetSettingOptions returns the options of the setting under the values given.
func (m Manifest) GetSettingOptions(s Setting, values map[string]any) SettingOptions {
	if s.OptionsBy == nil {
		return s.Options
	}
	return s.OptionsBy.Options[FormatSettingValue(values[s.OptionsBy.Setting])]
}

// ResolveSettings returns the values in force from the values stored: each setting holds its stored
// value, else its default; a setting whose options follow another holds the first of its options
// where its value is not among them; a setting whose condition fails is left out. Values of keys the
// manifest does not declare are kept.
func (m Manifest) ResolveSettings(stored map[string]any) map[string]any {
	values := maps.Clone(stored)
	if values == nil {
		values = map[string]any{}
	}
	for _, s := range m.Settings {
		if value, ok := m.resolveValue(s, values); ok {
			values[s.Key] = value
		} else {
			delete(values, s.Key)
		}
	}
	for _, s := range m.Settings {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="cycle detection bound exceeds any valid setting chain"
		if !m.isActive(s.When, values, 0) {
			delete(values, s.Key)
		}
	}
	return values
}

func (m Manifest) resolveValue(s Setting, values map[string]any) (any, bool) {
	value, ok := values[s.Key]
	if s.OptionsBy == nil {
		if !ok && s.Default != nil {
			return s.Default, true
		}
		return value, ok
	}
	options := m.GetSettingOptions(s, values)
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="nil value when not ok is rejected by isAcceptedOptionValue"
	if ok && s.isAcceptedOptionValue(options, value) {
		return value, true
	}
	if len(options) > 0 {
		return options[0].Value, true
	}
	if s.Default != nil {
		return s.Default, true
	}
	return nil, false
}

func (s Setting) isAcceptedOptionValue(options SettingOptions, value any) bool {
	if _, isOption := options.Find(FormatSettingValue(value)); isOption {
		return true
	}
	if s.Type != SettingTypeString {
		return false
	}
	str, isStr := value.(string)
	return isStr && str != "" && !s.hasOptionInAnyGroup(str)
}

func (s Setting) hasOptionInAnyGroup(value string) bool {
	// gomutants:disable-next-line BRANCH_IF reason="caller checks OptionsBy != nil"
	if s.OptionsBy == nil {
		return false
	}
	for _, list := range s.OptionsBy.Options {
		if _, ok := list.Find(value); ok {
			return true
		}
	}
	return false
}

// IsSettingShown reports whether the condition of the setting holds under the values in force that
// ResolveSettings returned, and the conditions of the settings it reads.
func (m Manifest) IsSettingShown(s Setting, resolved map[string]any) bool {
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="cycle detection bound exceeds any valid setting chain"
	return m.isActive(s.When, resolved, 0)
}

// IsSecretActive reports whether the secret is handed to the plugin under the values in force.
func (m Manifest) IsSecretActive(secret SecretDeclaration, values map[string]any) bool {
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="cycle detection bound exceeds any valid setting chain"
	return m.isActive(secret.When, values, 0)
}

func (m Manifest) isActive(when *Condition, values map[string]any, depth int) bool {
	if when == nil {
		return true
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="cycle detection bound exceeds any valid setting chain"
	if depth > len(m.Settings) || !slices.Contains(when.Is, FormatSettingValue(values[when.Setting])) {
		return false
	}
	parent, ok := m.GetSetting(when.Setting)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="cycle detection bound exceeds any valid setting chain"
	return ok && m.isActive(parent.When, values, depth+1)
}

func (m Manifest) getAllOptions(s Setting) []SettingOptions {
	lists := []SettingOptions{s.Options}
	if s.OptionsBy != nil {
		for _, key := range slices.Sorted(maps.Keys(s.OptionsBy.Options)) {
			lists = append(lists, s.OptionsBy.Options[key])
		}
	}
	return lists
}

func (m Manifest) getOptionalDependencies(values map[string]any) (optional, active map[string]bool) {
	optional, active = map[string]bool{}, map[string]bool{}
	for _, s := range m.Settings {
		for _, options := range m.getAllOptions(s) {
			for _, option := range options {
				addAll(optional, option.Dependencies)
			}
		}
		value, ok := values[s.Key]
		// gomutants:disable-next-line INVERT_LOGICAL reason="Find returns false for empty string when ok is false"
		if option, isOption := m.GetSettingOptions(s, values).Find(FormatSettingValue(value)); ok && isOption {
			addAll(active, option.Dependencies)
		}
	}
	return optional, active
}

func addAll(set map[string]bool, ids []string) {
	for _, id := range ids {
		set[id] = true
	}
}
