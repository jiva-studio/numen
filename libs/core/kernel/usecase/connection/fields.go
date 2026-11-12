package connection

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func getAddress(declared manifest.ConnectionType, fields map[string]any) string {
	for _, field := range declared.Fields {
		if !field.IsAddress {
			continue
		}
		if field.FixedValue != nil {
			return *field.FixedValue
		}
		address, isText := fields[field.Key].(string)
		if !isText {
			return ""
		}
		return address
	}
	return ""
}

func getSecretKeys(declared manifest.ConnectionType) []string {
	var keys []string
	for _, field := range declared.Fields {
		if field.Type == secretFieldType {
			keys = append(keys, field.Key)
		}
	}
	return keys
}

func normalizeFields(declared manifest.ConnectionType, requested map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for key, value := range requested {
		field, isDeclared := declared.GetField(key)
		switch {
		case !isDeclared:
			return nil, newInvalidError("the type has no field %q", key)
		case field.Type == secretFieldType:
			return nil, newInvalidError("%q is a secret and is sent as a secret", key)
		case field.FixedValue != nil:
			return nil, newInvalidError("%q has a fixed value", key)
		}
		cleaned, isSet, err := normalizeValue(field, value)
		if err != nil {
			return nil, err
		}
		if isSet {
			out[key] = cleaned
		}
	}
	return out, nil
}

func normalizeValue(field manifest.ConnectionField, value any) (cleaned any, isSet bool, err error) {
	if value == nil {
		return nil, false, nil
	}
	switch field.Type {
	case "string", "file":
		text, isText := value.(string)
		if !isText {
			return nil, false, newInvalidError("%q is a text", field.Key)
		}
		return text, text != "", nil
	case "boolean":
		flag, isBool := value.(bool)
		if !isBool {
			return nil, false, newInvalidError("%q is true or false", field.Key)
		}
		return flag, true, nil
	case "enum":
		text, isText := value.(string)
		if !isText {
			return nil, false, newInvalidError("%q is one of its options", field.Key)
		}
		if text == "" {
			return nil, false, nil
		}
		if _, found := field.Options.Find(text); !found {
			return nil, false, newInvalidError("%q is not an option of %q", text, field.Key)
		}
		return text, true, nil
	case "number":
		number, err := normalizeNumber(field, value)
		return number, err == nil, err
	}
	return nil, false, newInvalidError("%q has the type %q, which a connection does not take", field.Key, field.Type)
}

func toFloat(value any) (float64, bool) {
	switch item := value.(type) {
	case float64:
		return item, true
	case float32:
		return float64(item), true
	case int:
		return float64(item), true
	case int32:
		return float64(item), true
	case int64:
		return float64(item), true
	}
	return 0, false
}

func normalizeNumber(field manifest.ConnectionField, value any) (float64, error) {
	number, isNumber := toFloat(value)
	if !isNumber {
		return 0, newInvalidError("%q is a number", field.Key)
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, newInvalidError("%q is a finite number", field.Key)
	}
	if field.Min != nil && number < *field.Min {
		return 0, newInvalidError("%q is below %v", field.Key, *field.Min)
	}
	if field.Max != nil && number > *field.Max {
		return 0, newInvalidError("%q is above %v", field.Key, *field.Max)
	}
	if field.Step != nil && *field.Step > 0 {
		base := 0.0
		if field.Min != nil {
			base = *field.Min
		}
		steps := (number - base) / *field.Step
		if math.Abs(steps-math.Round(steps)) > 1e-9 {
			return 0, newInvalidError("%q is not a multiple of %v", field.Key, *field.Step)
		}
	}
	return number, nil
}

func checkSecretKeys(declared manifest.ConnectionType, requested map[string]string) error {
	keys := getSecretKeys(declared)
	for key := range requested {
		if !slices.Contains(keys, key) {
			return newInvalidError("the type has no secret field %q", key)
		}
	}
	return nil
}

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name != "" && !connection.IsValidName(name) {
		return "", newInvalidError("the name is over %d characters", connection.MaxNameLength)
	}
	return name, nil
}

func checkRequired(declared manifest.ConnectionType, fields map[string]any, secrets map[string]string) error {
	for _, field := range declared.Fields {
		if !field.IsRequired || field.FixedValue != nil {
			continue
		}
		if field.Type == secretFieldType {
			if secrets[field.Key] == "" {
				return newInvalidError("%s is required", describeField(field))
			}
			continue
		}
		if _, isSet := fields[field.Key]; !isSet {
			return newInvalidError("%s is required", describeField(field))
		}
	}
	return nil
}

func describeField(field manifest.ConnectionField) string {
	if field.Title != "" {
		return fmt.Sprintf("%q", field.Title)
	}
	return fmt.Sprintf("%q", field.Key)
}

func checkAddress(declared manifest.ConnectionType, fields map[string]any, hasKey, isPrivateAllowed bool) error {
	for _, field := range declared.Fields {
		if !field.IsAddress || field.FixedValue != nil {
			continue
		}
		address, isText := fields[field.Key].(string)
		if !isText || address == "" {
			return nil
		}
		return CheckAddress(address, hasKey, isPrivateAllowed)
	}
	return nil
}
