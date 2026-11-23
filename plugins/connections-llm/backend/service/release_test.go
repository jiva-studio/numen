package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process/processtest"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestAListOfModelsReleasesTheProcess(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey, IsLabeled: true, Models: []fakeopenai.Model{{ID: "openai/gpt-4o-mini"}}})
	host := processtest.NewHost(t)
	provider := newTestService(t).built.provider.roles()
	caller := llmtest.NewReleasingCaller(provider, newCall("openrouter", address, testKey, ""), host.NewClient(t))

	frames, err := caller.Call(t.Context(), llm.OperationModels, nil, connection.PriorityInteractive)
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

func TestAChatKeepsTheProcess(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey, IsLabeled: true, Models: []fakeopenai.Model{{ID: "openai/gpt-4o-mini"}}})
	host := processtest.NewHost(t)
	provider := newTestService(t).built.provider.roles()
	caller := llmtest.NewReleasingCaller(provider, newCall("openrouter", address, testKey, "openai/gpt-4o-mini"), host.NewClient(t))

	if _, err := llm.Complete(t.Context(), caller, &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: "name a river"}}}, connection.PriorityInteractive); err != nil {
		t.Fatal(err)
	}

	if releases := host.CountReleases(); releases != 0 {
		t.Errorf("releases = %d, want none", releases)
	}
}
