package providerbridge_test

import (
	"slices"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// runningHost hands out the endpoint of the activator only while isRunning is set.
type runningHost struct {
	*activator
	isRunning bool
	acquired  []string
}

func (h *runningHost) AcquireRunning(pluginID string) (transport.Endpoint, bool) {
	if !h.isRunning {
		return transport.Endpoint{}, false
	}
	h.acquired = append(h.acquired, pluginID)
	return h.endpoint, true
}

func TestRunningSourceOpensTheProviderOfARunningPluginAndReleasesIt(t *testing.T) {
	service := &scriptedService{frames: []*v1.ProviderServiceCallResponse{done([]byte("finished"))}}
	host := &runningHost{activator: newActivator(t, service), isRunning: true}

	provider, release, isRunning := providerbridge.NewRunningSource(host, host).OpenRunningProvider(t.Context(), pluginID)
	if !isRunning {
		t.Fatal("the running plugin was not opened")
	}
	answer, err := provider.Call(t.Context(), "llm.chat", modelcall.ProviderCall{ConnectionID: "c0000wren"}, nil, func([]byte) error { return nil })
	release()
	release()

	if err != nil || string(answer) != "finished" {
		t.Errorf("answer = %q, %v", answer, err)
	}
	if !slices.Equal(host.acquired, []string{pluginID}) || !slices.Equal(host.released, []string{pluginID}) {
		t.Errorf("acquired = %v, released = %v, want one of each", host.acquired, host.released)
	}
}

func TestRunningSourceDoesNotOpenAPluginThatDoesNotRun(t *testing.T) {
	host := &runningHost{activator: newActivator(t, &scriptedService{})}

	_, _, isRunning := providerbridge.NewRunningSource(host, host).OpenRunningProvider(t.Context(), pluginID)

	if isRunning || len(host.released) != 0 {
		t.Errorf("running = %v, released = %v, want neither", isRunning, host.released)
	}
}
