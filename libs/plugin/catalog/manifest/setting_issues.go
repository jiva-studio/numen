package manifest

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

// maxSecretGroupLength is the longest group name of a secret declaration, in characters.
const maxSecretGroupLength = 64

func (m Manifest) listSecretIssues() []error {
	var issues []error
	for _, secret := range m.Secrets {
		name := secret.Name
		issues = append(issues, m.listConditionIssues("secret "+name, secret.When)...)
		issues = append(issues, listSecretGroupIssues(secret)...)
		switch {
		case !manifestSecretPattern.MatchString(name):
			issues = append(issues, fmt.Errorf("%w: %q is not letters, digits, _ or -", ErrInvalidSecret, name))
		case slices.Contains(listReservedSettingKeys(), name):
			issues = append(issues, fmt.Errorf("%w: secret %q", ErrReservedSettingKey, name))
		case slices.ContainsFunc(m.Settings, func(s Setting) bool { return s.Key == name }):
			issues = append(issues, fmt.Errorf("%w: %q is also a setting", ErrInvalidSecret, name))
		}
	}
	return append(issues, listRepeatedSecretIssues(m.Secrets)...)
}

func listSecretGroupIssues(secret SecretDeclaration) []error {
	group := secret.Group
	if group == "" {
		return nil
	}
	switch {
	case strings.TrimSpace(group) != group:
		return []error{fmt.Errorf("%w: group %q of %q has leading or trailing space", ErrInvalidSecret, group, secret.Name)}
	case utf8.RuneCountInString(group) > maxSecretGroupLength:
		return []error{fmt.Errorf("%w: group of %q is longer than %d characters", ErrInvalidSecret, secret.Name, maxSecretGroupLength)}
	}
	return nil
}

func listRepeatedSecretIssues(secrets []SecretDeclaration) []error {
	var issues []error
	byName := make(map[string][]SecretDeclaration, len(secrets))
	for _, secret := range secrets {
		byName[secret.Name] = append(byName[secret.Name], secret)
	}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		declared := byName[name]
		if len(declared) < 2 {
			continue
		}
		if !isDisjoint(declared) {
			issues = append(issues, fmt.Errorf("%w: %q is declared twice without conditions on one setting that exclude each other", ErrInvalidSecret, name))
			continue
		}
		groups := make(map[string]bool, len(declared))
		for _, secret := range declared {
			if groups[secret.Group] {
				issues = append(issues, fmt.Errorf("%w: %q is declared twice for the group %q", ErrInvalidSecret, name, secret.Group))
			}
			groups[secret.Group] = true
		}
	}
	return issues
}

func isDisjoint(declared []SecretDeclaration) bool {
	values := make(map[string]bool)
	for _, secret := range declared {
		if secret.When == nil || secret.When.Setting != declared[0].When.Setting {
			return false
		}
		for _, is := range secret.When.Is {
			if values[is] {
				return false
			}
		}
		for _, is := range secret.When.Is {
			values[is] = true
		}
	}
	return true
}

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
	return issues
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
	return append(issues, listFileSettingIssues(s, key)...)
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

func (m Manifest) listConditionIssues(subject string, when *Condition) []error {
	declared := make(map[string]Setting, len(m.Settings))
	for _, s := range m.Settings {
		declared[s.Key] = s
	}
	return listConditionIssues(subject, when, declared)
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
