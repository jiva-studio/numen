// Package mcptool translates tools and tool results between the host's wire form and MCP.
package mcptool

import (
	"encoding/json"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

// NewTool declares a host tool to MCP clients. The input schema is always an object schema; the
// output schema is declared only when the host lists an object schema.
func NewTool(wireTool *tools.ToolInfo) *sdk.Tool {
	hints := wireTool.GetHints()
	tool := &sdk.Tool{
		Meta:         nil,
		Name:         wireTool.GetName(),
		Title:        wireTool.GetTitle(),
		Description:  wireTool.GetDescription(),
		InputSchema:  readInputSchema(wireTool.GetInputSchema()),
		OutputSchema: nil,
		Icons:        nil,
		Annotations: &sdk.ToolAnnotations{
			DestructiveHint: &hints.Destructive,
			IdempotentHint:  hints.Idempotent,
			OpenWorldHint:   &hints.OpenWorld,
			ReadOnlyHint:    hints.ReadOnly,
			Title:           "",
		},
	}
	if schema, isObject := readObject(wireTool.GetOutputSchema()); isObject && hasObjectType(schema) {
		tool.OutputSchema = json.RawMessage(wireTool.GetOutputSchema())
	}
	return tool
}

// NewResult translates the host's response to a call of the named tool. A host error code becomes
// an error result.
func NewResult(name string, res *tools.CallResponse) *sdk.CallToolResult {
	if res.GetError() != tools.ErrorCodeUnspecified {
		return NewErrorResult(describeError(res.GetError(), name))
	}

	wireResult := res.GetResult()
	result := &sdk.CallToolResult{
		Meta:              nil,
		Content:           nil,
		StructuredContent: nil,
		IsError:           wireResult.GetIsError(),
		InputRequests:     nil,
		RequestState:      "",
	}
	if len(wireResult.GetStructured()) > 0 {
		result.StructuredContent = json.RawMessage(wireResult.GetStructured())
	}

	text := wireResult.GetText()
	if text == "" {
		text = string(wireResult.GetStructured())
	}
	if text != "" {
		result.Content = []sdk.Content{newTextContent(text)}
	}
	return result
}

// NewErrorResult returns a tool result that reports the message as an error.
func NewErrorResult(message string) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		Meta:              nil,
		Content:           []sdk.Content{newTextContent(message)},
		StructuredContent: nil,
		IsError:           true,
		InputRequests:     nil,
		RequestState:      "",
	}
}

func describeError(code tools.ErrorCode, name string) string {
	switch code {
	case tools.ErrorCodeUnknownTool:
		return "unknown tool: " + name
	case tools.ErrorCodeBadArguments:
		return "invalid tool arguments"
	case tools.ErrorCodeUnspecified:
		return fmt.Sprintf("tool error: %v", code)
	default:
		return fmt.Sprintf("tool error: %v", code)
	}
}

func newTextContent(text string) *sdk.TextContent {
	return &sdk.TextContent{Text: text, Meta: nil, Annotations: nil}
}

func readInputSchema(raw []byte) json.RawMessage {
	schema, isObject := readObject(raw)
	if !isObject {
		return json.RawMessage(`{"type":"object"}`)
	}
	if hasObjectType(schema) {
		return json.RawMessage(raw)
	}
	schema["type"] = json.RawMessage(`"object"`)
	encoded, err := json.Marshal(schema)
	if err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	return encoded
}

func readObject(raw []byte) (map[string]json.RawMessage, bool) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, false
	}
	return object, true
}

func hasObjectType(schema map[string]json.RawMessage) bool {
	var schemaType string
	return json.Unmarshal(schema["type"], &schemaType) == nil && schemaType == "object"
}
