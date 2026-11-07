package editor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// getFailure is the failure the plugin page shows for the plugin, and nil while there is none.
func (w *window) getFailure(pluginID string) *pluginport.PluginFailure {
	for _, info := range w.host.GetPluginInfos() {
		if info.ID == pluginID {
			return info.Failure
		}
	}
	return nil
}

func TestAPluginWhoseConfigureFailedShowsTheFailureAndReceivesTheListOnceItRecovers(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	alpha := w.probes[alphaID]
	alpha.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))

	one := w.create(t, "openrouter", "Work", "value-of-the-work-key")
	w.allow(t, one, alphaID, "apiKey", "openrouter")

	plugintest.WaitUntil(t, "the failure of the plugin is shown", func() bool { return w.getFailure(alphaID) != nil })

	alpha.setFailure(nil)
	for range 4 {
		if err := w.host.RefreshSecrets(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	w.awaitIDs(t, alphaID, "apiKey", one)
}

func TestAHungConfigureDoesNotDelayTheOperation(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	alpha := w.probes[alphaID]
	hold := make(chan struct{})
	entered := make(chan struct{}, 1)
	alpha.mu.Lock()
	alpha.hold, alpha.entered = hold, entered
	alpha.mu.Unlock()
	t.Cleanup(func() { close(hold) })

	one := w.create(t, "openrouter", "Work", "value-of-the-work-key")
	result := make(chan error, 1)
	go func() {
		_, err := w.client.AllowPluginSecret(context.WithoutCancel(t.Context()), connect.NewRequest(&v1.AllowPluginSecretRequest{
			SecretId: one, PluginId: alphaID, Name: "apiKey", Group: "openrouter",
		}))
		result <- err
	}()

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the plugin never received the Configure call")
	}
	// The host gives a Configure call 3s; an operation that returns well before has not waited for it.
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("the operation failed: %v", err)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("the operation still waits for the hung Configure call")
	}
}

func TestAFailingPluginDoesNotFailAnOperationThatDoesNotConcernIt(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	w.probes[alphaID].setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))

	one := w.create(t, "openai", "Work", "value-of-the-work-key")
	w.allow(t, one, alphaID, "apiKey", "openrouter")
	// The operation below changes the list of beta only; w.allow fails the test when it returns an error.
	w.allow(t, one, betaID, "apiKey", "openai")

	w.awaitIDs(t, betaID, "apiKey", one)
}
