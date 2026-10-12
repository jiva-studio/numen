package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// canonicalKey converts an identifier into lowercase alphanumeric characters,
// removing all underscores, dashes, spaces, and punctuation.
// This allows matching "SearchPath", "search_path", "search-path", "search path", and "SEARCHPATH"
// to the same canonical key "searchpath".
func canonicalKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		}
	}
	return b.String()
}

// buildCanonicalMap inspects the struct type T and builds a lookup table
// mapping canonical field names and json tags to the exact struct JSON key.
func buildCanonicalMap(t reflect.Type) map[string]string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	m := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		m[canonicalKey(jsonName)] = jsonName
		m[canonicalKey(field.Name)] = jsonName
	}
	return m
}

// normalizeArguments re-keys raw JSON arguments using the canonical map so that
// any casing, underscores, or dashes in caller arguments correctly bind to the struct's JSON keys.
func normalizeArguments(raw json.RawMessage, canonMap map[string]string) (json.RawMessage, error) {
	if len(raw) == 0 || !isObjectJSON(raw) || len(canonMap) == 0 {
		return raw, nil
	}

	var rawMap map[string]any
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return nil, err
	}

	normalized := make(map[string]any, len(rawMap))
	for k, v := range rawMap {
		targetKey, ok := canonMap[canonicalKey(k)]
		if ok {
			normalized[targetKey] = v
		} else {
			normalized[k] = v
		}
	}

	return json.Marshal(normalized)
}

func isObjectJSON(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

// addTool registers a tool with robust parameter normalization, case-insensitivity,
// and permissive argument binding.
func addTool[In, Out any](
	server *sdk.Server,
	tool *sdk.Tool,
	handler func(ctx context.Context, req *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error),
) {
	inType := reflect.TypeFor[In]()
	if tool.InputSchema == nil {
		tool.InputSchema = buildInputSchema(inType)
	}
	if tool.OutputSchema == nil {
		tool.OutputSchema = buildOutputSchema(reflect.TypeFor[Out]())
	}

	canonMap := buildCanonicalMap(inType)

	server.AddTool(tool, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		in, err := parseArguments[In](req, canonMap)
		if err != nil {
			return newErrorResult(err), nil
		}

		res, out, err := handler(ctx, req, in)
		if err != nil {
			return newErrorResult(err), nil
		}
		return fillResult(res, out), nil
	})
}

// newHandler returns the tool handler that calls run with the parsed input and
// builds the result from run's output.
func newHandler[In, Out any](
	run func(ctx context.Context, in In) (Out, error),
) func(context.Context, *sdk.CallToolRequest, In) (*sdk.CallToolResult, Out, error) {
	return func(ctx context.Context, _ *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error) {
		out, err := run(ctx, in)
		return nil, out, err
	}
}

// buildInputSchema returns the schema of a struct input type, and an empty
// object schema for any other type or when no schema can be derived.
func buildInputSchema(inType reflect.Type) *jsonschema.Schema {
	if inType.Kind() != reflect.Struct {
		return &jsonschema.Schema{Type: "object"}
	}
	schema, err := jsonschema.ForType(inType, &jsonschema.ForOptions{})
	if err != nil {
		return &jsonschema.Schema{Type: "object"}
	}
	return schema
}

// buildOutputSchema returns the schema of a struct output type or a pointer to
// one, and nil for any other type or when no schema can be derived.
func buildOutputSchema(outType reflect.Type) *jsonschema.Schema {
	if outType.Kind() == reflect.Pointer {
		outType = outType.Elem()
	}
	if outType.Kind() != reflect.Struct {
		return nil
	}
	schema, err := jsonschema.ForType(outType, &jsonschema.ForOptions{})
	if err != nil {
		return nil
	}
	return schema
}

// parseArguments decodes the call's arguments into In after re-keying them
// with canonMap. A call without arguments yields the zero In.
func parseArguments[In any](req *sdk.CallToolRequest, canonMap map[string]string) (In, error) {
	var in In
	if req.Params == nil || len(req.Params.Arguments) == 0 {
		return in, nil
	}
	normalizedJSON, err := normalizeArguments(req.Params.Arguments, canonMap)
	if err != nil {
		return in, fmt.Errorf("normalizing arguments: %w", err)
	}
	if err := json.Unmarshal(normalizedJSON, &in); err != nil {
		return in, fmt.Errorf("unmarshaling arguments: %w", err)
	}
	return in, nil
}

func newErrorResult(err error) *sdk.CallToolResult {
	var res sdk.CallToolResult
	res.SetError(err)
	return &res
}

// fillResult returns res, or an empty result when res is nil. A result with no
// content is filled with structured content when the output encodes to an object and
// with text otherwise.
func fillResult(res *sdk.CallToolResult, out any) *sdk.CallToolResult {
	if res == nil {
		res = &sdk.CallToolResult{}
	}
	if len(res.Content) != 0 || res.StructuredContent != nil {
		return res
	}
	outJSON, err := json.Marshal(out)
	if err != nil {
		return res
	}
	if isObjectJSON(outJSON) {
		res.StructuredContent = out
		return res
	}
	res.Content = append(res.Content, &sdk.TextContent{Text: string(outJSON)})
	return res
}
