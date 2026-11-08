package manifest

import (
	"fmt"
	"math"
)

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
