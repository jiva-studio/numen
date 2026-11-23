package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process/processtest"
)

func TestADetectionAnswerReleasesTheProcess(t *testing.T) {
	withoutClaudeAnywhere(t)
	host := processtest.NewHost(t)
	caller := llmtest.NewReleasingCaller(newTestAgent(t, nil).Provider(), connection.Call{Type: "claude-code"}, host.NewClient(t))

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
