package pluginport

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// PluginSettingOption is one value an enum setting can hold.
type PluginSettingOption struct {
	Value string `json:"value"`
	// Title is what a person reads for the value. Empty shows the value itself.
	Title string `json:"title,omitempty"`
	// Detail is what a person reads under the setting while the value is in force.
	Detail string `json:"detail,omitempty"`
	// Dependencies are the ids of the manifest's dependencies the host provisions only while the
	// option is in force.
	Dependencies []string `json:"dependencies,omitempty"`
}

// GetTitle returns the title of the option, or its value where it has none.
func (o PluginSettingOption) GetTitle() string {
	if o.Title != "" {
		return o.Title
	}
	return o.Value
}

// PluginSettingOptions are the values an enum setting can hold, in the order they are offered.
type PluginSettingOptions []PluginSettingOption

// UnmarshalJSON reads a list of values or a list of options.
func (o *PluginSettingOptions) UnmarshalJSON(data []byte) error {
	var values []string
	if err := json.Unmarshal(data, &values); err == nil {
		options := make(PluginSettingOptions, 0, len(values))
		for _, value := range values {
			options = append(options, PluginSettingOption{Value: value})
		}
		*o = options
		return nil
	}
	var options []PluginSettingOption
	if err := json.Unmarshal(data, &options); err != nil {
		return errors.New("options must be a list of values or of objects with a value")
	}
	*o = options
	return nil
}

// Find returns the option holding the value.
func (o PluginSettingOptions) Find(value string) (PluginSettingOption, bool) {
	for _, one := range o {
		if one.Value == value {
			return one, true
		}
	}
	return PluginSettingOption{}, false
}

// PluginSettingOptionsBy gives a setting its options from the value of another setting.
type PluginSettingOptionsBy struct {
	// Setting is the key of the enum setting the options follow.
	Setting string `json:"setting"`
	// Options are the options under each value of that setting.
	Options map[string]PluginSettingOptions `json:"options"`
}

// PluginSettingCondition holds while a setting holds one of the values listed.
type PluginSettingCondition struct {
	Setting string   `json:"setting"`
	Is      []string `json:"is"`
}

// PluginSecret is one secret, such as an API key, a plugin asks the host to keep.
type PluginSecret struct {
	// Name is what the plugin reads the key under.
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	// When is the condition under which the secret is shown and handed to the plugin.
	When *PluginSettingCondition `json:"when,omitempty"`
}

// GetSecretNames returns the names of the secrets the plugin declares.
func (m PluginManifest) GetSecretNames() []string {
	names := make([]string, 0, len(m.Secrets))
	for _, secret := range m.Secrets {
		names = append(names, secret.Name)
	}
	return names
}

// GetSetting returns the declaration of the setting with the key.
func (m PluginManifest) GetSetting(key string) (PluginManifestSetting, bool) {
	for _, s := range m.Settings {
		if s.Key == key {
			return s, true
		}
	}
	return PluginManifestSetting{}, false
}

// FormatSettingValue is the value as a condition and a list of options compare it.
func FormatSettingValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
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
func (m PluginManifest) GetSettingOptions(s PluginManifestSetting, values map[string]any) PluginSettingOptions {
	if s.OptionsBy == nil {
		return s.Options
	}
	return s.OptionsBy.Options[FormatSettingValue(values[s.OptionsBy.Setting])]
}

// ResolveSettings returns the values in force from the values stored: each setting holds its stored
// value, else its default; a setting whose options follow another holds the first of its options
// where its value is not among them; a setting whose condition fails is left out. Values of keys the
// manifest does not declare are kept.
func (m PluginManifest) ResolveSettings(stored map[string]any) map[string]any {
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
		if !m.isActive(s.When, values, 0) {
			delete(values, s.Key)
		}
	}
	return values
}

// resolveValue returns the value in force of one setting under the values resolved before it, and
// false where it holds none.
func (m PluginManifest) resolveValue(s PluginManifestSetting, values map[string]any) (any, bool) {
	value, ok := values[s.Key]
	if s.OptionsBy == nil {
		if !ok && s.Default != nil {
			return s.Default, true
		}
		return value, ok
	}
	options := m.GetSettingOptions(s, values)
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

func (s PluginManifestSetting) isAcceptedOptionValue(options PluginSettingOptions, value any) bool {
	if _, isOption := options.Find(FormatSettingValue(value)); isOption {
		return true
	}
	if s.Type != SettingTypeString {
		return false
	}
	str, isStr := value.(string)
	return isStr && str != "" && !s.hasOptionInAnyGroup(str)
}

func (s PluginManifestSetting) hasOptionInAnyGroup(value string) bool {
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

// IsSecretActive reports whether the secret is handed to the plugin under the values in force.
func (m PluginManifest) IsSecretActive(secret PluginSecret, values map[string]any) bool {
	return m.isActive(secret.When, values, 0)
}

// isActive reports whether the condition holds, and the conditions of the settings it reads.
func (m PluginManifest) isActive(when *PluginSettingCondition, values map[string]any, depth int) bool {
	if when == nil {
		return true
	}
	if depth > len(m.Settings) || !slices.Contains(when.Is, FormatSettingValue(values[when.Setting])) {
		return false
	}
	parent, ok := m.GetSetting(when.Setting)
	return ok && m.isActive(parent.When, values, depth+1)
}

// listSettingLinkIssues checks what the settings say of one another and of the dependencies: a
// setting whose options follow another names an enum setting declared before it and lists options
// under each of its values; a condition names a setting declared before it; an option provisions
// dependencies the manifest declares.
func (m PluginManifest) listSettingLinkIssues() []error {
	var issues []error
	declared := map[string]PluginManifestSetting{}
	for _, s := range m.Settings {
		issues = append(issues, m.listOptionsByIssues(s, declared)...)
		issues = append(issues, listConditionIssues("setting "+s.Key, s.When, declared)...)
		for _, options := range m.getAllOptions(s) {
			for _, option := range options {
				for _, id := range option.Dependencies {
					if !slices.ContainsFunc(m.Dependencies, func(req DependencyRequirement) bool { return req.ID == id }) {
						issues = append(issues, fmt.Errorf("%w: %s option %q provisions %q, which the manifest does not declare", ErrInvalidSetting, s.Key, option.Value, id))
					}
				}
			}
		}
		declared[s.Key] = s
	}
	return issues
}

func (m PluginManifest) listOptionsByIssues(s PluginManifestSetting, declared map[string]PluginManifestSetting) []error {
	if s.OptionsBy == nil {
		return nil
	}
	if (s.Type != SettingTypeEnum && s.Type != "string") || len(s.Options) > 0 {
		return []error{fmt.Errorf("%w: %s takes its options from %s and is no enum or string without options of its own", ErrInvalidSetting, s.Key, s.OptionsBy.Setting)}
	}
	parent, ok := declared[s.OptionsBy.Setting]
	if !ok || parent.Type != SettingTypeEnum || parent.OptionsBy != nil || len(parent.Options) == 0 {
		return []error{fmt.Errorf("%w: %s takes its options from %q, which is no enum with options declared before it", ErrInvalidSetting, s.Key, s.OptionsBy.Setting)}
	}
	var issues []error
	for _, option := range parent.Options {
		if s.Type == SettingTypeEnum && len(s.OptionsBy.Options[option.Value]) == 0 {
			issues = append(issues, fmt.Errorf("%w: %s lists no options under %s %q", ErrInvalidSetting, s.Key, parent.Key, option.Value))
		}
	}
	return issues
}

// listConditionIssues checks a condition against every setting the manifest declares.
func (m PluginManifest) listConditionIssues(subject string, when *PluginSettingCondition) []error {
	declared := make(map[string]PluginManifestSetting, len(m.Settings))
	for _, s := range m.Settings {
		declared[s.Key] = s
	}
	return listConditionIssues(subject, when, declared)
}

func listConditionIssues(subject string, when *PluginSettingCondition, declared map[string]PluginManifestSetting) []error {
	if when == nil {
		return nil
	}
	parent, ok := declared[when.Setting]
	if !ok {
		return []error{fmt.Errorf("%w: %s is shown by %q, which is no setting declared before it", ErrInvalidSetting, subject, when.Setting)}
	}
	if len(when.Is) == 0 {
		return []error{fmt.Errorf("%w: %s is shown by %s and lists no values", ErrInvalidSetting, subject, when.Setting)}
	}
	var issues []error
	for _, value := range when.Is {
		switch {
		case parent.Type == "boolean" && value != "true" && value != "false":
			issues = append(issues, fmt.Errorf("%w: %s is shown by %s holding %q, and it holds true or false", ErrInvalidSetting, subject, when.Setting, value))
		case len(parent.Options) > 0:
			if _, isOption := parent.Options.Find(value); !isOption {
				issues = append(issues, fmt.Errorf("%w: %s is shown by %s holding %q, which is none of its options", ErrInvalidSetting, subject, when.Setting, value))
			}
		}
	}
	return issues
}

// getAllOptions returns every list of options the setting declares.
func (m PluginManifest) getAllOptions(s PluginManifestSetting) []PluginSettingOptions {
	lists := []PluginSettingOptions{s.Options}
	if s.OptionsBy != nil {
		for _, key := range slices.Sorted(maps.Keys(s.OptionsBy.Options)) {
			lists = append(lists, s.OptionsBy.Options[key])
		}
	}
	return lists
}

// getOptionalDependencies returns the ids of the dependencies an option provisions, and those of
// the options in force under the values given.
func (m PluginManifest) getOptionalDependencies(values map[string]any) (optional, active map[string]bool) {
	optional, active = map[string]bool{}, map[string]bool{}
	for _, s := range m.Settings {
		for _, options := range m.getAllOptions(s) {
			for _, option := range options {
				addAll(optional, option.Dependencies)
			}
		}
		value, ok := values[s.Key]
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
