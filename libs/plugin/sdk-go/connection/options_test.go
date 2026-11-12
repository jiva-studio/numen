package connection

import (
	"context"
	"errors"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func TestServeOptionsReadsTheOperationsOfTheSettingAndAnswersTheOptions(t *testing.T) {
	var operations []string
	var model string
	client := newProviderClient(t, Provider{"llm.reasoning": ServeOptions(func(_ context.Context, call Call, uses []string) ([]Option, error) {
		operations, model = uses, call.Options["model"]
		return []Option{{Value: "low", Title: "Low"}, {Value: "high", Title: "High", Group: "Slower"}}, nil
	})})
	request := newCallRequest("llm.reasoning")
	payload, err := proto.Marshal(&v1.OptionsRequest{Operations: []string{"llm.chat"}})
	if err != nil {
		t.Fatal(err)
	}
	request.Msg.Payload = payload

	got, err := readCallResponse(t, client, request)

	if err != nil || !got.isDone {
		t.Fatalf("Call = %+v, %v", got, err)
	}
	var listed v1.Options
	if err := proto.Unmarshal([]byte(got.done), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.GetOptions()) != 2 || listed.GetOptions()[1].GetValue() != "high" || listed.GetOptions()[1].GetGroup() != "Slower" {
		t.Errorf("options = %v", listed.GetOptions())
	}
	if !slices.Equal(operations, []string{"llm.chat"}) || model != "google/gemini-2.5-flash" {
		t.Errorf("operations = %v, model = %q", operations, model)
	}
}

func TestServeOptionsPassesTheFailureOfTheListOn(t *testing.T) {
	failure := NewTryNextError(ReasonRejectedKey, 0)
	operation := ServeOptions(func(context.Context, Call, []string) ([]Option, error) { return nil, failure })

	_, err := operation(t.Context(), Call{}, nil, nil)

	if !errors.Is(err, failure) {
		t.Errorf("err = %v, want the failure of the list", err)
	}
}

func TestServeOptionsRefusesAPayloadThatIsNoOptionsRequest(t *testing.T) {
	operation := ServeOptions(func(context.Context, Call, []string) ([]Option, error) { return nil, nil })

	_, err := operation(t.Context(), Call{}, []byte{0xff, 0xff}, nil)

	if reason, _ := getReason(err); !IsStop(err) || reason != ReasonInvalidRequest {
		t.Errorf("err = %v, want a stop of an invalid request", err)
	}
}
