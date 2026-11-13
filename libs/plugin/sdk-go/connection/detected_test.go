package connection

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func TestServeDetectedAnswersTheConnectionsFoundForTheType(t *testing.T) {
	var typeID string
	client := newProviderClient(t, Provider{OperationDetected: ServeDetected(func(_ context.Context, call Call) ([]DetectedConnection, error) {
		typeID = call.Type
		return []DetectedConnection{{Name: "Orchard agent", Fields: map[string]string{"program": "orchard"}}, {Name: "Meadow agent"}}, nil
	})})

	got, err := readCallResponse(t, client, newCallRequest(OperationDetected))

	if err != nil || !got.isDone {
		t.Fatalf("Call = %+v, %v", got, err)
	}
	var listed v1.DetectedConnections
	if err := proto.Unmarshal([]byte(got.done), &listed); err != nil {
		t.Fatal(err)
	}
	found := listed.GetConnections()
	if len(found) != 2 || found[0].GetName() != "Orchard agent" || found[0].GetFields()["program"] != "orchard" || found[1].GetName() != "Meadow agent" {
		t.Errorf("connections = %v", found)
	}
	if typeID != "openrouter" {
		t.Errorf("type = %q, want the type of the call", typeID)
	}
}

func TestServeDetectedPassesTheFailureOfTheDetectionOn(t *testing.T) {
	failure := NewTryNextError(ReasonUnavailable, 0)
	operation := ServeDetected(func(context.Context, Call) ([]DetectedConnection, error) { return nil, failure })

	_, err := operation(t.Context(), Call{}, nil, nil)

	if !errors.Is(err, failure) {
		t.Errorf("err = %v, want the failure of the detection", err)
	}
}
