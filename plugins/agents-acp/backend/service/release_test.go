package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process/processtest"
)

func TestADetectionAnswerReleasesTheProcess(t *testing.T) {
	host := processtest.NewHost(t)
	caller := llmtest.NewReleasingCaller(newTestServer(t).Provider(), connection.Call{Type: "acp"}, host.NewClient(t))

	frames, err := caller.Call(t.Context(), connection.OperationDetected, nil, connection.PriorityInteractive)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := connection.Collect(frames); err != nil {
		t.Fatal(err)
	}

	if releases := host.CountReleases(); releases != 1 {
		t.Errorf("releases = %d, want 1", releases)
	}
}
