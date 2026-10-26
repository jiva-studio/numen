package container_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// The window asks for its agent at every call, so a conversation asked through one call ends on the
// plugin it was asked in when another call finishes it.
func TestTheEditorsAgentEndsAConversationAskedThroughAnEarlierCall(t *testing.T) {
	host, one, _ := twoProviders(t)
	assembly := container.NewEditorAssembly(container.Config{PluginHost: host}, nil, nil)

	runTask(t, assembly.GetAgent(), agentport.Task{Question: "q", Agent: "alpha", Conversation: "c-1"})
	if err := assembly.GetAgent().EndConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(one.getFinished(), []string{"c-1"}) {
		t.Errorf("the plugin finished %v", one.getFinished())
	}
}
