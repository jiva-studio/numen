package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

type greetIn struct {
	Name string `json:"name"`
}

type greetOut struct {
	Greeting string `json:"greeting"`
}

func newRegistry(t *testing.T) *tools.Registry {
	t.Helper()
	registry := tools.New()
	err := tools.Add(registry, tools.Spec{
		Name: "greet", Title: "Greet", Description: "Greets.", Effect: tools.ReadOnly,
		Servers: []tools.Server{tools.Full, tools.Reading}, Step: tools.Search,
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

func call(t *testing.T, registry *tools.Registry, name, arguments string) *numenv1.CallToolResponse {
	t.Helper()
	res, err := registry.CallTool(t.Context(), connect.NewRequest(&numenv1.CallToolRequest{Name: name, Arguments: []byte(arguments)}))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func TestListTools_ReturnsSchemasAndServers(t *testing.T) {
	res, err := newRegistry(t).ListTools(t.Context(), connect.NewRequest(&numenv1.ListToolsRequest{}))
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
	if tool.GetEffect() != numenv1.ToolEffect_TOOL_EFFECT_READ_ONLY || tool.GetStep() != numenv1.ToolStep_TOOL_STEP_SEARCH ||
		len(tool.GetServers()) != 2 {
		t.Errorf("the tool came out as %v", tool)
	}
}

func TestACallRunsTheToolOnItsArgumentsAndAnswersStructured(t *testing.T) {
	answer := call(t, newRegistry(t), "greet", `{"name":"Sita"}`)

	if answer.GetError() != numenv1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED || answer.GetResult().GetIsRefusal() {
		t.Fatalf("the call was answered %v", answer)
	}
	if string(answer.GetResult().GetStructured()) != `{"greeting":"hello Sita"}` {
		t.Errorf("the result was %s", answer.GetResult().GetStructured())
	}
}

func TestAToolThatReturnsAnErrorRanAndRefused(t *testing.T) {
	answer := call(t, newRegistry(t), "greet", `{}`)

	if !answer.GetResult().GetIsRefusal() || answer.GetResult().GetText() != "recipient required" {
		t.Errorf("the call was answered %v", answer)
	}
}

func TestArgumentsThatDoNotFitTheStructAreAError(t *testing.T) {
	answer := call(t, newRegistry(t), "greet", `{"name":3}`)

	if !answer.GetResult().GetIsRefusal() {
		t.Errorf("the call was answered %v", answer)
	}
}

func TestCall_UnknownNameAndNonObjectArgsReturnErrorCode(t *testing.T) {
	registry := newRegistry(t)

	if got := call(t, registry, "other", `{}`).GetError(); got != numenv1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL {
		t.Errorf("an unknown tool was answered %v", got)
	}
	if got := call(t, registry, "greet", `[1]`).GetError(); got != numenv1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS {
		t.Errorf("arguments that are no object were answered %v", got)
	}
}

func TestCall_TextResultReturnsText(t *testing.T) {
	registry := tools.New()
	if err := tools.Add(registry, tools.Spec{Name: "count", Effect: tools.ReadOnly}, func(context.Context, struct{}) (int, error) {
		return 3, nil
	}); err != nil {
		t.Fatal(err)
	}

	answer := call(t, registry, "count", "")

	if answer.GetResult().GetText() != "3" || len(answer.GetResult().GetStructured()) != 0 {
		t.Errorf("the call was answered %v", answer)
	}
}

func TestTwoToolsOfOneNameAreRefused(t *testing.T) {
	registry := newRegistry(t)

	err := tools.Add(registry, tools.Spec{Name: "greet", Effect: tools.ReadOnly}, func(context.Context, greetIn) (greetOut, error) {
		return greetOut{}, nil
	})

	if !errors.Is(err, tools.ErrDuplicate) {
		t.Errorf("the second tool was added with %v", err)
	}
}

func TestCall_ContextCarriesTokenMark(t *testing.T) {
	registry := tools.New()
	type in struct{}
	type out struct {
		Mark string `json:"mark"`
	}
	if err := tools.Add(registry, tools.Spec{Name: "current", Effect: tools.ReadOnly}, func(ctx context.Context, _ in) (out, error) {
		return out{Mark: tools.GetCallContext(ctx)["org.example.sample/item"]}, nil
	}); err != nil {
		t.Fatal(err)
	}

	res, err := registry.CallTool(t.Context(), connect.NewRequest(&numenv1.CallToolRequest{
		Name: "current", Arguments: []byte(`{}`), Context: map[string]string{"org.example.sample/item": "k7"},
	}))

	if err != nil || !strings.Contains(string(res.Msg.GetResult().GetStructured()), "k7") {
		t.Errorf("got %v (%v)", res.Msg.GetResult(), err)
	}
}

func TestCall_ContextCarriesVault(t *testing.T) {
	registry := tools.New()
	type in struct{}
	type out struct {
		Vault string `json:"vault"`
	}
	if err := tools.Add(registry, tools.Spec{Name: "current", Effect: tools.ReadOnly}, func(ctx context.Context, _ in) (out, error) {
		return out{Vault: tools.GetVault(ctx)}, nil
	}); err != nil {
		t.Fatal(err)
	}

	res, err := registry.CallTool(t.Context(), connect.NewRequest(&numenv1.CallToolRequest{
		Name: "current", Arguments: []byte(`{}`), VaultId: "v-garden",
	}))

	if err != nil || string(res.Msg.GetResult().GetStructured()) != `{"vault":"v-garden"}` {
		t.Errorf("got %v (%v)", res.Msg.GetResult(), err)
	}
}
