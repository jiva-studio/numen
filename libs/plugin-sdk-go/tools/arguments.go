package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

// quotedPattern matches a Go-quoted string, as jsonschema-go writes property names in its messages.
var quotedPattern = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)

// keywordPattern matches the keyword jsonschema-go writes before the reason of a failure, such as
// "type: " or "dependentRequired[\"a\"]: ".
var keywordPattern = regexp.MustCompile(`^[A-Za-z]+(\[[^\]]*\])?: `)

// readArguments validates the JSON arguments of a call against schema and decodes them into In. No
// arguments and JSON null are validated as the empty object and leave In zero.
func readArguments[In any](schema *jsonschema.Resolved, arguments []byte) (In, error) {
	var in In
	if len(arguments) == 0 {
		return in, validateArguments(schema, map[string]any{})
	}
	var value any
	if err := json.Unmarshal(arguments, &value); err != nil {
		return in, fmt.Errorf("the arguments are not JSON: %w", err)
	}
	if value == nil {
		return in, validateArguments(schema, map[string]any{})
	}
	if err := validateArguments(schema, value); err != nil {
		return in, err
	}
	if err := json.Unmarshal(arguments, &in); err != nil {
		return in, fmt.Errorf("the arguments do not fit: %w", err)
	}
	return in, nil
}

// validateArguments validates the decoded arguments of a call against schema.
func validateArguments(schema *jsonschema.Resolved, value any) error {
	if err := schema.Validate(value); err != nil {
		return errors.New(describeValidationError(err))
	}
	return nil
}

// describeValidationError writes a jsonschema-go validation failure as a sentence that names the
// argument. jsonschema-go wraps the failure once per schema on the way to it, each layer starting
// with "validating " and the JSON pointer of that schema.
func describeValidationError(err error) string {
	var path []string
	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			break
		}
		layer := strings.TrimSuffix(err.Error(), ": "+inner.Error())
		if pointer, isSchema := strings.CutPrefix(layer, "validating "); isSchema {
			path = readPropertyPath(pointer)
		}
		err = inner
	}
	reason := err.Error()
	if missing, isMissing := strings.CutPrefix(reason, "required: missing properties: "); isMissing {
		return describeArguments(path, readQuoted(missing), "is missing", "are missing")
	}
	if unknown, isUnknown := strings.CutPrefix(reason, "unexpected additional properties "); isUnknown {
		return describeArguments(path, readQuoted(unknown), "is not known", "are not known")
	}
	reason = keywordPattern.ReplaceAllString(reason, "")
	if len(path) == 0 {
		return "the arguments are invalid: " + reason
	}
	return fmt.Sprintf("the argument %s is invalid: %s", strings.Join(path, "."), reason)
}

// describeArguments writes a sentence about the named properties of the object at path, with
// single or plural as the predicate.
func describeArguments(path, names []string, single, plural string) string {
	qualified := make([]string, 0, len(names))
	for _, name := range names {
		qualified = append(qualified, strings.Join(append(append([]string(nil), path...), name), "."))
	}
	if len(qualified) == 1 {
		return fmt.Sprintf("the argument %s %s", qualified[0], single)
	}
	return fmt.Sprintf("the arguments %s %s", strings.Join(qualified, ", "), plural)
}

// readPropertyPath returns the property names along a JSON pointer into a schema, such as
// ["location", "title"] for /properties/location/properties/title.
func readPropertyPath(pointer string) []string {
	var names []string
	segments := strings.Split(pointer, "/")
	for i := 0; i+1 < len(segments); i++ {
		if segments[i] == "properties" {
			names = append(names, strings.NewReplacer("~1", "/", "~0", "~").Replace(segments[i+1]))
			i++
		}
	}
	return names
}

// readQuoted returns the Go-quoted strings in text, unquoted.
func readQuoted(text string) []string {
	var values []string
	for _, quoted := range quotedPattern.FindAllString(text, -1) {
		if value, err := strconv.Unquote(quoted); err == nil {
			values = append(values, value)
		}
	}
	return values
}
