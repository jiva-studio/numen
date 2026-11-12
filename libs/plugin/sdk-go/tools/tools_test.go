package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

type greetIn struct {
	Name string `json:"name"`
}

type greetOut struct {
	Greeting string `json:"greeting"`
}

func newRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	err := Add(registry, Spec{
		Name: "greet", Title: "Greet", Description: "Greets.", Effect: ReadOnly,
		Servers: []Server{Full, Reading}, Step: Search,
	}, func(_ context.Context, in greetIn) (greetOut, error) {
		if in.Name == "" {
			return greetOut{}, errors.New("recipient required")
		}
		return greetOut{Greeting: "hello " + in.Name}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func call(t *testing.T, registry *Registry, name, arguments string) *v1.CallToolResponse {
	t.Helper()
	res, err := registryHandler{registry: registry}.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{Name: name, Arguments: []byte(arguments)}))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func TestListTools_ReturnsSchemasAndServers(t *testing.T) {
	res, err := registryHandler{registry: newRegistry(t)}.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Msg.GetTools()) != 1 {
		t.Fatalf("%d tools listed", len(res.Msg.GetTools()))
	}
	tool := res.Msg.GetTools()[0]
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(tool.GetInputSchema(), &input); err != nil || input.Properties["name"] == nil {
		t.Errorf("the input schema is %s (%v)", tool.GetInputSchema(), err)
	}
	if len(tool.GetOutputSchema()) == 0 {
		t.Error("the result is a struct and lists no schema")
	}
	if tool.GetEffect() != v1.ToolEffect_TOOL_EFFECT_READ_ONLY || tool.GetStep() != v1.ToolStep_TOOL_STEP_SEARCH ||
		len(tool.GetServers()) != 2 {
		t.Errorf("the tool came out as %v", tool)
	}
}

func TestACallRunsTheToolOnItsArgumentsAndAnswersStructured(t *testing.T) {
	answer := call(t, newRegistry(t), "greet", `{"name":"Sita"}`)

	if answer.GetError() != v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED || answer.GetResult().GetIsError() {
		t.Fatalf("the call was answered %v", answer)
	}
	if string(answer.GetResult().GetStructured()) != `{"greeting":"hello Sita"}` {
		t.Errorf("the result was %s", answer.GetResult().GetStructured())
	}
}

func TestAToolThatReturnsAnErrorRanAndRefused(t *testing.T) {
	answer := call(t, newRegistry(t), "greet", `{"name":""}`)

	if !answer.GetResult().GetIsError() || answer.GetResult().GetText() != "greet: recipient required" {
		t.Errorf("the call was answered %v", answer)
	}
}

func TestCall_UnknownNameAndNonObjectArgsReturnErrorCode(t *testing.T) {
	registry := newRegistry(t)

	if got := call(t, registry, "other", `{}`).GetError(); got != v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL {
		t.Errorf("an unknown tool was answered %v", got)
	}
	if got := call(t, registry, "greet", `[1]`).GetError(); got != v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS {
		t.Errorf("arguments that are no object were answered %v", got)
	}
	if got := call(t, registry, "greet", `1`).GetError(); got != v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS {
		t.Errorf("single-byte non-object arguments were answered %v", got)
	}
}

func TestCall_TextResultReturnsText(t *testing.T) {
	registry := New()
	if err := Add(registry, Spec{Name: "count", Effect: ReadOnly}, func(context.Context, int) (int, error) {
		return 3, nil
	}); err != nil {
		t.Fatal(err)
	}

	listing, err := registry.ListTools(t.Context())
	infos := listing.Tools
	if err != nil || len(infos) != 1 {
		t.Fatalf("ListTools = %v, %v", infos, err)
	}
	if string(infos[0].GetInputSchema()) != `{"type":"object"}` || len(infos[0].GetOutputSchema()) != 0 {
		t.Errorf("non-struct schemas: in=%s out=%s", infos[0].GetInputSchema(), infos[0].GetOutputSchema())
	}

	answer := call(t, registry, "count", "")

	if answer.GetResult().GetText() != "3" || len(answer.GetResult().GetStructured()) != 0 {
		t.Errorf("the call was answered %v", answer)
	}
}

func TestTwoToolsOfOneNameAreRefused(t *testing.T) {
	registry := newRegistry(t)
	if err := Add(registry, Spec{Name: "farewell", Effect: ReadWrite}, func(context.Context, greetIn) (*greetOut, error) {
		return &greetOut{Greeting: "bye"}, nil
	}); err != nil {
		t.Fatalf("second distinct tool rejected: %v", err)
	}

	err := Add(registry, Spec{Name: "greet", Effect: ReadOnly}, func(context.Context, greetIn) (greetOut, error) {
		return greetOut{}, nil
	})

	if !errors.Is(err, ErrDuplicate) {
		t.Errorf("the second tool was added with %v", err)
	}
}

func TestCall_ContextCarriesTokenMark(t *testing.T) {
	registry := New()
	type in struct{}
	type out struct {
		Mark string `json:"mark"`
	}
	if err := Add(registry, Spec{Name: "current", Effect: ReadOnly}, func(ctx context.Context, _ in) (out, error) {
		return out{Mark: GetCallContext(ctx)["org.example.sample/item"]}, nil
	}); err != nil {
		t.Fatal(err)
	}

	res, err := registryHandler{registry: registry}.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: "current", Arguments: []byte(`{}`), Context: map[string]string{"org.example.sample/item": "k7"},
	}))

	if err != nil || !strings.Contains(string(res.Msg.GetResult().GetStructured()), "k7") {
		t.Errorf("got %v (%v)", res.Msg.GetResult(), err)
	}
}

func TestCall_ContextCarriesVault(t *testing.T) {
	registry := New()
	type in struct{}
	type out struct {
		Vault string `json:"vault"`
	}
	if err := Add(registry, Spec{Name: "current", Effect: ReadOnly}, func(ctx context.Context, _ in) (out, error) {
		return out{Vault: GetVault(ctx)}, nil
	}); err != nil {
		t.Fatal(err)
	}

	res, err := registryHandler{registry: registry}.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: "current", Arguments: []byte(`{}`), VaultId: "v-garden",
	}))

	if err != nil || string(res.Msg.GetResult().GetStructured()) != `{"vault":"v-garden"}` {
		t.Errorf("got %v (%v)", res.Msg.GetResult(), err)
	}
}

func TestConversionsConstantsAndSchemaErrors(t *testing.T) {
	if ReadOnly != 1 || ReadWrite != 2 {
		t.Fatalf("effects = %d, %d", ReadOnly, ReadWrite)
	}
	if Full != 1 || Reading != 2 {
		t.Fatalf("servers = %d, %d", Full, Reading)
	}
	if ErrorCodeUnspecified != 0 || ErrorCodeUnknownTool != 1 || ErrorCodeBadArguments != 2 {
		t.Fatalf("error codes = %d, %d, %d", ErrorCodeUnspecified, ErrorCodeUnknownTool, ErrorCodeBadArguments)
	}
	if readEffect(v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED) != 0 {
		t.Fatal("expected 0 for unspecified effect")
	}
	if readToolInfo(nil) != nil || writeToolInfo(nil) != nil || readCallResponse(nil) != nil {
		t.Fatal("expected nil for nil conversions")
	}
	if got := writeCallResponse(nil); got == nil || got.GetError() != v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED || got.GetResult() != nil {
		t.Fatalf("writeCallResponse(nil) = %v", got)
	}
	if res := writeResult(make(chan int)); !res.GetIsError() || !strings.HasPrefix(res.GetText(), "the result cannot be encoded:") {
		t.Fatalf("writeResult(chan) = %v", res)
	}

	type unencodable struct {
		Ch chan int `json:"ch"`
	}
	registry := newRegistry(t)
	_ = call(t, registry, "greet", `{"name":"Ann"}`)
	if err := Add(registry, Spec{Name: "after-call"}, func(context.Context, greetIn) (greetOut, error) {
		return greetOut{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := Add(registry, Spec{Name: "bad-in"}, func(context.Context, unencodable) (greetOut, error) {
		return greetOut{}, nil
	}); err == nil {
		t.Fatal("expected error for unencodable input schema")
	}
	if err := Add(registry, Spec{Name: "bad-out"}, func(context.Context, greetIn) (unencodable, error) {
		return unencodable{}, nil
	}); err == nil {
		t.Fatal("expected error for unencodable output schema")
	}

	if err := Add(registry, Spec{Name: "ptr-struct"}, func(context.Context, *greetIn) (*greetOut, error) {
		return &greetOut{Greeting: "hi"}, nil
	}); err != nil {
		t.Fatalf("Add pointer struct failed: %v", err)
	}
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var ptrInfo *ToolInfo
	for _, info := range listing.Tools {
		if info.GetName() == "ptr-struct" {
			ptrInfo = info
		}
	}
	if ptrInfo == nil || !strings.Contains(string(ptrInfo.GetInputSchema()), `"name"`) || !strings.Contains(string(ptrInfo.GetOutputSchema()), `"greeting"`) {
		t.Fatalf("expected pointer struct schemas to contain fields, got in=%s out=%s", ptrInfo.GetInputSchema(), ptrInfo.GetOutputSchema())
	}
	var nilInfo *ToolInfo
	if nilInfo.GetName() != "" || nilInfo.GetInputSchema() != nil || nilInfo.GetOutputSchema() != nil {
		t.Fatal("expected empty name and nil schemas for nil ToolInfo")
	}
}
