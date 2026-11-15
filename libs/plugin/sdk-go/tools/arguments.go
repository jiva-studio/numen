package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

// The JSON types of a schema.
const (
	typeNull    = "null"
	typeBoolean = "boolean"
	typeString  = "string"
	typeInteger = "integer"
	typeNumber  = "number"
	typeObject  = "object"
	typeArray   = "array"
)

// maxIntegerDigits is the longest decimal integer a whole number of an integer argument is
// rewritten to. A longer one is read as a number.
const maxIntegerDigits = 1000

func readArguments[In any](schema *jsonschema.Schema, arguments []byte) (In, error) {
	var in In
	value, err := decodeArguments(arguments)
	if err != nil {
		return in, err
	}
	if value == nil {
		_, err := normalizeValue(schema, map[string]any{}, "")
		return in, err
	}
	normalized, err := normalizeValue(schema, value, "")
	if err != nil {
		return in, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return in, errors.New("the arguments are invalid")
	}
	if err := json.Unmarshal(encoded, &in); err != nil {
		return in, describeDecodeError(encoded, err)
	}
	return in, nil
}

func decodeArguments(arguments []byte) (any, error) {
	if len(arguments) == 0 {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, errors.New("the arguments are not JSON")
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("the arguments are not JSON")
	}
	return value, nil
}

func normalizeValue(schema *jsonschema.Schema, value any, path string) (any, error) {
	if schema == nil {
		return value, nil
	}
	if err := checkType(schema, value, path); err != nil {
		return nil, err
	}
	switch v := value.(type) {
	case map[string]any:
		return normalizeObject(schema, v, path)
	case []any:
		return normalizeArray(schema, v, path)
	case json.Number:
		return normalizeNumber(schema, v, path)
	}
	return value, nil
}

func normalizeObject(schema *jsonschema.Schema, object map[string]any, path string) (any, error) {
	removeNullOptionals(schema, object)
	if err := checkProperties(schema, object, path); err != nil {
		return nil, err
	}
	names := slices.Sorted(maps.Keys(object))
	for _, name := range names {
		property, isProperty := schema.Properties[name]
		if !isProperty {
			property = schema.AdditionalProperties
		}
		normalized, err := normalizeValue(property, object[name], joinProperty(path, name))
		if err != nil {
			return nil, err
		}
		object[name] = normalized
	}
	return object, nil
}

func removeNullOptionals(schema *jsonschema.Schema, object map[string]any) {
	maps.DeleteFunc(object, func(name string, value any) bool {
		_, isProperty := schema.Properties[name]
		return isProperty && value == nil && !slices.Contains(schema.Required, name)
	})
}

func checkProperties(schema *jsonschema.Schema, object map[string]any, path string) error {
	missing := slices.DeleteFunc(slices.Clone(schema.Required), func(name string) bool {
		_, isPresent := object[name]
		return isPresent
	})
	if len(missing) > 0 {
		return describeArguments(path, missing, "is missing", "are missing")
	}
	if !isFalseSchema(schema.AdditionalProperties) {
		return nil
	}
	unknown := slices.DeleteFunc(slices.Sorted(maps.Keys(object)), func(name string) bool {
		_, isProperty := schema.Properties[name]
		return isProperty
	})
	if len(unknown) > 0 {
		return describeArguments(path, unknown, "is not known", "are not known")
	}
	return nil
}

func normalizeArray(schema *jsonschema.Schema, items []any, path string) (any, error) {
	if reason := checkLength(schema, len(items)); reason != "" {
		return nil, describeInvalid(path, reason)
	}
	for i, item := range items {
		normalized, err := normalizeValue(schema.Items, item, path+"["+strconv.Itoa(i)+"]")
		if err != nil {
			return nil, err
		}
		items[i] = normalized
	}
	return items, nil
}

func normalizeNumber(schema *jsonschema.Schema, number json.Number, path string) (any, error) {
	integer, isInteger := readInteger(number)
	if isInteger && !slices.Contains(readTypes(schema), typeNumber) {
		number = json.Number(integer.String())
	}
	value := new(big.Float)
	if isInteger {
		value.SetInt(integer)
	} else if parsed, err := strconv.ParseFloat(string(number), 64); err == nil || errors.Is(err, strconv.ErrRange) {
		value.SetFloat64(parsed)
	}
	if schema.Minimum != nil && value.Cmp(big.NewFloat(*schema.Minimum)) < 0 {
		return nil, describeInvalid(path, "it must be at least "+formatLimit(*schema.Minimum))
	}
	if schema.Maximum != nil && value.Cmp(big.NewFloat(*schema.Maximum)) > 0 {
		return nil, describeInvalid(path, "it must be at most "+formatLimit(*schema.Maximum))
	}
	return number, nil
}

func checkType(schema *jsonschema.Schema, value any, path string) error {
	allowed := readTypes(schema)
	actual := readJSONType(value)
	if len(allowed) == 0 || slices.Contains(allowed, actual) || (actual == typeInteger && slices.Contains(allowed, typeNumber)) {
		return nil
	}
	expected := slices.DeleteFunc(slices.Clone(allowed), func(name string) bool { return name == typeNull })
	if len(expected) == 0 {
		expected = allowed
	}
	words := make([]string, len(expected))
	for i, name := range expected {
		words[i] = writeTypeWord(name)
	}
	return fmt.Errorf("the argument %s must be %s, not %s", path, strings.Join(words, " or "), writeTypeWord(actual))
}

func checkLength(schema *jsonschema.Schema, length int) string {
	switch {
	case schema.MinItems != nil && schema.MaxItems != nil && *schema.MinItems == *schema.MaxItems && length != *schema.MinItems:
		return fmt.Sprintf("it must have exactly %d items", *schema.MinItems)
	case schema.MinItems != nil && length < *schema.MinItems:
		return fmt.Sprintf("it must have at least %d items", *schema.MinItems)
	case schema.MaxItems != nil && length > *schema.MaxItems:
		return fmt.Sprintf("it must have at most %d items", *schema.MaxItems)
	}
	return ""
}

func readTypes(schema *jsonschema.Schema) []string {
	if schema.Type != "" {
		return []string{schema.Type}
	}
	return schema.Types
}

func readJSONType(value any) string {
	switch v := value.(type) {
	case nil:
		return typeNull
	case bool:
		return typeBoolean
	case string:
		return typeString
	case map[string]any:
		return typeObject
	case []any:
		return typeArray
	case json.Number:
		if _, isInteger := readInteger(v); isInteger {
			return typeInteger
		}
	}
	return typeNumber
}

func writeTypeWord(name string) string {
	switch name {
	case typeNull:
		return typeNull
	case typeInteger, typeObject, typeArray:
		return "an " + name
	}
	return "a " + name
}

func readInteger(number json.Number) (*big.Int, bool) {
	text := string(number)
	mantissa, exponentText, hasExponent := strings.Cut(strings.ToLower(text), "e")
	exponent := 0
	if hasExponent {
		parsed, err := strconv.Atoi(exponentText)
		if err != nil || parsed > maxIntegerDigits || parsed < -maxIntegerDigits {
			return nil, false
		}
		exponent = parsed
	}
	whole, fraction, _ := strings.Cut(mantissa, ".")
	digits := strings.TrimRight(whole+fraction, "0")
	exponent += len(whole+fraction) - len(digits) - len(fraction)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return new(big.Int), true
	}
	if exponent < 0 || len(digits)+exponent > maxIntegerDigits {
		return nil, false
	}
	integer, isValid := new(big.Int).SetString(sign+digits+strings.Repeat("0", exponent), 10)
	return integer, isValid
}

func formatLimit(limit float64) string {
	return strconv.FormatFloat(limit, 'f', -1, 64)
}

func isFalseSchema(schema *jsonschema.Schema) bool {
	return schema != nil && schema.Not != nil && reflect.ValueOf(*schema.Not).IsZero()
}

func joinProperty(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func describeArguments(path string, names []string, single, plural string) error {
	qualified := make([]string, len(names))
	for i, name := range names {
		qualified[i] = joinProperty(path, name)
	}
	if len(qualified) == 1 {
		return fmt.Errorf("the argument %s %s", qualified[0], single)
	}
	return fmt.Errorf("the arguments %s %s", strings.Join(qualified, ", "), plural)
}

func describeInvalid(path, reason string) error {
	if path == "" {
		return fmt.Errorf("the arguments are invalid: %s", reason)
	}
	return fmt.Errorf("the argument %s is invalid: %s", path, reason)
}

func describeDecodeError(encoded []byte, err error) error {
	if typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		if path, isFound := findPathAt(encoded, typeErr.Offset); isFound && path != "" {
			return fmt.Errorf("the argument %s is invalid", path)
		}
	}
	return errors.New("the arguments are invalid")
}

func findPathAt(encoded []byte, offset int64) (string, bool) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	stack := []*pathLevel{{isRoot: true}}
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", false
		}
		delim, isDelim := token.(json.Delim)
		if isDelim && (delim == '}' || delim == ']') {
			stack = stack[:len(stack)-1]
			continue
		}
		path, isValue := stack[len(stack)-1].readValuePath(token)
		if !isValue {
			continue
		}
		if decoder.InputOffset() == offset {
			return path, true
		}
		if isDelim {
			stack = append(stack, &pathLevel{path: path, isArray: delim == '['})
		}
	}
}

// pathLevel is the top level of a JSON token stream, or an object or array open in it.
type pathLevel struct {
	path      string
	isRoot    bool
	isArray   bool
	index     int
	key       string
	isKeyRead bool
}

func (l *pathLevel) readValuePath(token json.Token) (string, bool) {
	switch {
	case l.isRoot:
		return "", true
	case l.isArray:
		path := l.path + "[" + strconv.Itoa(l.index) + "]"
		l.index++
		return path, true
	case !l.isKeyRead:
		key, isKey := token.(string)
		l.key, l.isKeyRead = key, isKey
		return "", false
	}
	l.isKeyRead = false
	return joinProperty(l.path, l.key), true
}
