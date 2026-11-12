package agentrun

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
)

func getOmittedFields() map[protoreflect.FullName]bool {
	return map[protoreflect.FullName]bool{
		"numen.v1.AskAgentRequest.vault_id": true,
	}
}

func collectFields(msg protoreflect.Message, covered map[protoreflect.FullName]bool) {
	msg.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		covered[field.FullName()] = true
		switch {
		case field.IsList() && field.Message() != nil:
			for i := range value.List().Len() {
				collectFields(value.List().Get(i).Message(), covered)
			}
		case field.IsMap() && field.MapValue().Message() != nil:
			value.Map().Range(func(_ protoreflect.MapKey, item protoreflect.Value) bool {
				collectFields(item.Message(), covered)
				return true
			})
		case field.Message() != nil && !field.IsList() && !field.IsMap():
			collectFields(value.Message(), covered)
		}
		return true
	})
}

func requireAllFields(t *testing.T, descriptor protoreflect.MessageDescriptor, covered map[protoreflect.FullName]bool) {
	t.Helper()
	visited := map[protoreflect.FullName]bool{}
	var walk func(protoreflect.MessageDescriptor)
	walk = func(message protoreflect.MessageDescriptor) {
		if visited[message.FullName()] {
			return
		}
		visited[message.FullName()] = true
		for i := range message.Fields().Len() {
			field := message.Fields().Get(i)
			if !covered[field.FullName()] && !getOmittedFields()[field.FullName()] {
				t.Errorf("the conversion does not carry the protocol field %s", field.FullName())
			}
			switch {
			case field.IsMap() && field.MapValue().Message() != nil:
				walk(field.MapValue().Message())
			case field.Message() != nil && !field.IsMap():
				walk(field.Message())
			}
		}
	}
	walk(descriptor)
}

func assertSame[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func getFullRequest() *Request {
	return &Request{
		Prompt:       "prompt",
		Focus:        "notes/focus.md",
		Conversation: "c-1",
		Context:      map[string]string{"org.example/key": "value"},
		Agent:        "agent",
		Model:        "model",
		Instructions: "instructions",
		Tools:        []string{"note_read", "note_edit"},
		MaxSteps:     7,
		ToolDeclarations: []toolcall.ToolDeclaration{{
			Name: "note_edit", Title: "Edit", Kind: "edit", About: "path", Element: "element", Match: "match", Text: "text",
		}},
		Root:       "/vault",
		Command:    []string{"program", "--flag"},
		ToolsURL:   "http://127.0.0.1:1/mcp",
		ToolsToken: "token",
		Reasoning:  "high",
	}
}

func TestRequestRoundTripsThroughTheProtocol(t *testing.T) {
	want := getFullRequest()
	msg := requestToProto(want)
	assertSame(t, requestFromProto(msg), want)

	covered := map[protoreflect.FullName]bool{}
	collectFields(msg.ProtoReflect(), covered)
	requireAllFields(t, msg.ProtoReflect().Descriptor(), covered)
}

func TestResponseRoundTripsThroughTheProtocol(t *testing.T) {
	call := &toolcall.ToolCall{
		Tool: "tool", About: "about", Length: 12, Path: "notes/a.md", Range: &toolcall.Range{Start: 1, End: 5}, Call: "c", Kind: "edit",
	}
	responses := []*Response{
		{Step: &TextStep{Text: "text"}},
		{Step: &ToolCallStep{ToolCall: call}},
		{Step: &StoppedStep{Stopped: "done"}},
		{Step: &ToolFinishedStep{}},
		{Step: &ModelStartedStep{}},
		{Step: &EditProgressStep{EditProgress: &EditProgress{Call: "c", Path: "notes/a.md", Match: "old", Text: "new"}}},
	}
	covered := map[protoreflect.FullName]bool{}
	for _, want := range responses {
		msg := responseToProto(want)
		assertSame(t, responseFromProto(msg), want)
		collectFields(msg.ProtoReflect(), covered)
	}
	requireAllFields(t, (&v1.AskAgentResponse{}).ProtoReflect().Descriptor(), covered)
}

func TestStateRoundTripsThroughTheProtocol(t *testing.T) {
	want := &State{Unreachable: "not found"}
	msg := stateToProto(want)
	assertSame(t, stateFromProto(msg), want)

	covered := map[protoreflect.FullName]bool{}
	collectFields(msg.ProtoReflect(), covered)
	requireAllFields(t, msg.ProtoReflect().Descriptor(), covered)
}

func TestProgramCommandsRoundTripThroughTheProtocol(t *testing.T) {
	want := []ProgramCommand{{Program: "codex", Command: []string{"codex", "--flag"}}}
	msg := programCommandsToProto(want)
	assertSame(t, programCommandsFromProto(msg), want)

	covered := map[protoreflect.FullName]bool{}
	collectFields(msg.ProtoReflect(), covered)
	requireAllFields(t, msg.ProtoReflect().Descriptor(), covered)
}

func TestProgramsRoundTripThroughTheProtocol(t *testing.T) {
	want := []*Program{{
		Name:        "codex",
		Title:       "Codex",
		IsInstalled: true,
		IsDefault:   true,
		LimitsTools: true,
		Models: []*reasoning.Model{{
			Name:             "model",
			Title:            "Model",
			IsDefault:        true,
			ReasoningLevels:  []reasoning.Level{{Name: "low", Title: "Low"}, {Name: "high", Title: "High"}},
			DefaultReasoning: "low",
		}},
	}}
	msg := programsToProto(want)
	assertSame(t, programsFromProto(msg), want)

	covered := map[protoreflect.FullName]bool{}
	collectFields(msg.ProtoReflect(), covered)
	requireAllFields(t, msg.ProtoReflect().Descriptor(), covered)
}

func TestToolSetRoundTripsThroughTheProtocol(t *testing.T) {
	want := &ToolSet{
		Tools:                 []*ToolDefinition{{Name: "tool", Description: "description", ParametersJSONSchema: `{"type":"object"}`}},
		SystemPromptExtension: "extension",
	}
	msg := toolSetToProto(want)
	assertSame(t, toolSetFromProto(msg), want)

	covered := map[protoreflect.FullName]bool{}
	collectFields(msg.ProtoReflect(), covered)
	requireAllFields(t, msg.ProtoReflect().Descriptor(), covered)

	registration := &v1.RegisterToolRequest{Tool: toolDefinitionToProto(want.Tools[0])}
	assertSame(t, toolDefinitionFromProto(registration.GetTool()), want.Tools[0])
}

func TestNilValuesStayNil(t *testing.T) {
	if requestToProto(nil) != nil || requestFromProto(nil) != nil || responseToProto(nil) != nil ||
		responseFromProto(nil) != nil || toolCallToProto(nil) != nil || toolCallFromProto(nil) != nil ||
		editProgressToProto(nil) != nil || editProgressFromProto(nil) != nil ||
		toolDefinitionToProto(nil) != nil || toolDefinitionFromProto(nil) != nil {
		t.Error("a nil value converts to a value")
	}
}
