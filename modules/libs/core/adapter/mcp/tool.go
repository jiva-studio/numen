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

	// Generate input schema if not explicitly provided.
	if tool.InputSchema == nil {
		if inType.Kind() == reflect.Struct {
			schema, err := jsonschema.ForType(inType, &jsonschema.ForOptions{})
			if err != nil {
				schema = &jsonschema.Schema{Type: "object"}
			}
			tool.InputSchema = schema
		} else {
			tool.InputSchema = &jsonschema.Schema{Type: "object"}
		}
	}

	// Generate output schema if Out is not any and not explicitly provided.
	outType := reflect.TypeFor[Out]()
	if tool.OutputSchema == nil && outType != reflect.TypeFor[any]() {
		if outType.Kind() == reflect.Struct || (outType.Kind() == reflect.Pointer && outType.Elem().Kind() == reflect.Struct) {
			targetOut := outType
			if targetOut.Kind() == reflect.Pointer {
				targetOut = targetOut.Elem()
			}
			schema, err := jsonschema.ForType(targetOut, &jsonschema.ForOptions{})
			if err == nil {
				tool.OutputSchema = schema
			}
		}
	}

	canonMap := buildCanonicalMap(inType)

	server.AddTool(tool, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		var in In

		if req.Params != nil && len(req.Params.Arguments) > 0 {
			normalizedJSON, err := normalizeArguments(req.Params.Arguments, canonMap)
			if err != nil {
				var errRes sdk.CallToolResult
				errRes.SetError(fmt.Errorf("normalizing arguments: %w", err))
				return &errRes, nil
			}

			if err := json.Unmarshal(normalizedJSON, &in); err != nil {
				var errRes sdk.CallToolResult
				errRes.SetError(fmt.Errorf("unmarshaling arguments: %w", err))
				return &errRes, nil
			}
		}

		res, out, err := handler(ctx, req, in)
		if err != nil {
			var errRes sdk.CallToolResult
			errRes.SetError(err)
			return &errRes, nil
		}

		if res == nil {
			res = &sdk.CallToolResult{}
		}

		if len(res.Content) == 0 && res.StructuredContent == nil {
			outJSON, err := json.Marshal(out)
			if err == nil {
				if isObjectJSON(outJSON) {
					res.StructuredContent = out
				} else {
					res.Content = append(res.Content, &sdk.TextContent{
						Text: string(outJSON),
					})
				}
			}
		}

		return res, nil
	})
}
