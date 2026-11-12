package tools

import (
	"reflect"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func TestToolInfo_RoundTripsThroughTheWire(t *testing.T) {
	want := &ToolInfo{
		Name: "greet", Title: "Greet", Description: "Greets.",
		InputSchema: []byte(`{"type":"object"}`), OutputSchema: []byte(`{"type":"object"}`),
		Effect:       ReadWrite,
		Instructions: []string{"Greet each guest by name with greet.", "Call greet once per guest."},
		Requires:     []string{"vault:manage", "index:read"},
	}

	got := readToolInfo(writeToolInfo(want))

	if !reflect.DeepEqual(got, want) {
		t.Errorf("tool info %+v, want %+v", got, want)
	}
}

func TestCallResponse_RoundTripsThroughTheWire(t *testing.T) {
	cases := []*CallResponse{
		{Result: &CallResult{Text: "failed", IsError: true}},
		{Result: &CallResult{Structured: []byte(`{"a":1}`)}},
		{Error: ErrorCodeUnknownTool},
		{Error: ErrorCodeBadArguments},
	}
	for _, want := range cases {
		got := readCallResponse(writeCallResponse(want))

		if !reflect.DeepEqual(got, want) {
			t.Errorf("call response %+v, want %+v", got, want)
		}
	}
}

// Every value the protocol defines for an enum maps to an SDK value that maps back to it, so a value
// added to the protocol without a mapping fails here.
func TestEnums_EveryProtocolValueRoundTrips(t *testing.T) {
	for value := range v1.ToolEffect_name {
		effect := v1.ToolEffect(value)
		if got := writeEffect(readEffect(effect)); got != effect {
			t.Errorf("effect %v came back as %v", effect, got)
		}
	}
	for value := range v1.ToolErrorCode_name {
		code := v1.ToolErrorCode(value)
		if got := writeErrorCode(readErrorCode(code)); got != code {
			t.Errorf("error code %v came back as %v", code, got)
		}
	}
}
