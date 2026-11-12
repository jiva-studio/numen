package manifest

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

func listManifestSettingIssues(settings []Setting) []error {
	var issues []error
	seenKeys := make(map[string]bool, len(settings))
	for _, s := range settings {
		key := strings.TrimSpace(s.Key)
		if key == "" {
			issues = append(issues, ErrEmptySettingKey)
			continue
		}
		if slices.Contains(listReservedSettingKeys(), key) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrReservedSettingKey, key))
			continue
		}
		if seenKeys[key] {
			issues = append(issues, fmt.Errorf("%w: %s", ErrDuplicateSettingKey, key))
			continue
		}
		seenKeys[key] = true
		if strings.Contains(key, ".") {
			issues = append(issues, fmt.Errorf("%w: %s holds a dot, and a key is one segment of a path", ErrInvalidSettingKey, key))
		}
		issues = append(issues, listManifestSettingIssue(s, key)...)
	}
	return append(issues, listDuplicateOperationIssues(settings)...)
}

func listManifestSettingIssue(s Setting, key string) []error {
	var issues []error
	if strings.TrimSpace(s.Title) == "" {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEmptySettingTitle, key))
	}
	if !slices.Contains(listSettingTypes(), s.Type) {
		issues = append(issues, fmt.Errorf("%w: %q for %s", ErrInvalidSettingType, s.Type, key))
	}
	if s.Type == SettingTypeEnum && len(s.Options) == 0 && s.OptionsBy == nil && s.OptionsFrom == "" {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEnumMissingOptions, key))
	}
	if s.Type == "range" && s.Min != nil && s.Max != nil && *s.Min > *s.Max {
		issues = append(issues, fmt.Errorf("%w: %s", ErrRangeInvalidBounds, key))
	}
	issues = append(issues, listGenerateSettingIssues(s, key)...)
	issues = append(issues, listFileSettingIssues(s, key)...)
	return append(issues, listConnectionsSettingIssues(s, key)...)
}

func listGenerateSettingIssues(s Setting, key string) []error {
	if s.Generate == "" {
		return nil
	}
	var issues []error
	if s.Generate != SettingGenerateToken {
		issues = append(issues, fmt.Errorf("%w: %s generates %q, and the host generates only %q", ErrInvalidSetting, key, s.Generate, SettingGenerateToken))
	}
	if s.Type != SettingTypeString {
		issues = append(issues, fmt.Errorf("%w: %s generates a value and is no string setting", ErrInvalidSetting, key))
	}
	if s.Default != nil {
		issues = append(issues, fmt.Errorf("%w: %s generates a value and declares a default", ErrInvalidSetting, key))
	}
	return issues
}

func listFileSettingIssues(s Setting, key string) []error {
	var issues []error
	if len(s.Accept) > 0 && s.Type != SettingTypeFile {
		issues = append(issues, fmt.Errorf("%w: %s accepts extensions and holds no file", ErrInvalidSetting, key))
	}
	if s.Multiple && s.Type != SettingTypeFile {
		issues = append(issues, fmt.Errorf("%w: %s holds several paths and is no file setting", ErrInvalidSetting, key))
	}
	for _, extension := range s.Accept {
		if len(extension) < 2 || !strings.HasPrefix(extension, ".") {
			issues = append(issues, fmt.Errorf("%w: %s accepts %q, which is no extension", ErrInvalidSetting, key, extension))
		}
	}
	return issues
}

func (m Manifest) listSettingLinkIssues() []error {
	var issues []error
	declared := map[string]Setting{}
	for _, s := range m.Settings {
		issues = append(issues, m.listOptionsByIssues(s, declared)...)
		issues = append(issues, listConditionIssues("setting "+s.Key, s.When, declared)...)
		for _, options := range m.getAllOptions(s) {
			for _, option := range options {
				issues = append(issues, m.listOptionIssues(s, option)...)
			}
		}
		declared[s.Key] = s
	}
	return issues
}

func (m Manifest) listOptionIssues(s Setting, option SettingOption) []error {
	var issues []error
	if _, err := when.Parse(option.When); err != nil {
		issues = append(issues, fmt.Errorf("%w: %s option %q: when %q: %w", ErrInvalidSetting, s.Key, option.Value, option.When, err))
	}
	for _, id := range option.Dependencies {
		if !slices.ContainsFunc(m.Dependencies, func(req DependencyRequirement) bool { return req.ID == id }) {
			issues = append(issues, fmt.Errorf("%w: %s option %q provisions %q, which the manifest does not declare", ErrInvalidSetting, s.Key, option.Value, id))
		}
	}
	return issues
}

func (m Manifest) listOptionsByIssues(s Setting, declared map[string]Setting) []error {
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

func listConditionIssues(subject string, when *Condition, declared map[string]Setting) []error {
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
		case parent.Type == settingTypeBoolean && value != "true" && value != "false":
			issues = append(issues, fmt.Errorf("%w: %s is shown by %s holding %q, and it holds true or false", ErrInvalidSetting, subject, when.Setting, value))
		case len(parent.Options) > 0:
			if _, isOption := parent.Options.Find(value); !isOption {
				issues = append(issues, fmt.Errorf("%w: %s is shown by %s holding %q, which is none of its options", ErrInvalidSetting, subject, when.Setting, value))
			}
		}
	}
	return issues
}

func listConnectionsSettingIssues(s Setting, key string) []error {
	if s.Type != SettingTypeConnections {
		if s.Operation == "" {
			return nil
		}
		return []error{fmt.Errorf("%w: %s names an operation and holds no rows of connections", ErrInvalidSetting, key)}
	}
	var issues []error
	if !connectionOperationPattern.MatchString(s.Operation) {
		issues = append(issues, fmt.Errorf("%w: %s needs an operation of lowercase letters and digits, got %q", ErrInvalidSetting, key, s.Operation))
	}
	if s.Max == nil || *s.Max < 1 || *s.Max != math.Trunc(*s.Max) {
		issues = append(issues, fmt.Errorf("%w: %s needs max, the number of rows, as a whole number of at least 1", ErrInvalidSetting, key))
	}
	return append(issues, listConnectionsSettingExtraIssues(s, key)...)
}

func listConnectionsSettingExtraIssues(s Setting, key string) []error {
	var issues []error
	declared := []struct {
		name  string
		isSet bool
	}{
		{"default", s.Default != nil}, {"options", len(s.Options) > 0}, {"optionsBy", s.OptionsBy != nil},
		{"optionsFrom", s.OptionsFrom != ""}, {"min", s.Min != nil}, {"step", s.Step != nil}, {"preview", s.Preview},
	}
	for _, one := range declared {
		if one.isSet {
			issues = append(issues, fmt.Errorf("%w: %s holds rows of connections and declares %s", ErrInvalidSetting, key, one.name))
		}
	}
	return issues
}

func listDuplicateOperationIssues(settings []Setting) []error {
	var issues []error
	keyOf := map[string]string{}
	for _, s := range settings {
		if s.Type != SettingTypeConnections {
			continue
		}
		if first, isTaken := keyOf[s.Operation]; isTaken {
			issues = append(issues, fmt.Errorf("%w: %s and %s both hold rows for the operation %q", ErrInvalidSetting, first, s.Key, s.Operation))
			continue
		}
		keyOf[s.Operation] = s.Key
	}
	return issues
}
