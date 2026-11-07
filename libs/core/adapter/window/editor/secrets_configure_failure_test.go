package editor_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const (
	workValue = "value-of-the-work-key"
	// configureTimeout is how long the host waits for one Configure call.
	configureTimeout = 3 * time.Second
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

// tryAllow allows the secret to the declaration and returns the error of the call.
func (w *window) tryAllow(ctx context.Context, secretID, pluginID, name, group string) error {
	_, err := w.client.AllowPluginSecret(ctx, connect.NewRequest(&v1.AllowPluginSecretRequest{
		SecretId: secretID, PluginId: pluginID, Name: name, Group: group,
	}))
	return err
}

// A plugin that does not accept its list, and puts the value in the text of its error, fails the
// operation with an error that names the plugin and holds no value. The permission stays, and the
// plugin is not marked as failed.
func TestAPluginWhoseConfigureFailsKeepsThePermissionAndShowsNoFailureOrValue(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	w.probes[alphaID].setFailure(connect.NewError(connect.CodeInternal, errors.New("rejected the key "+workValue)))
	one := w.create(t, "openrouter", "Work", workValue)

	err := w.tryAllow(t.Context(), one, alphaID, "apiKey", "openrouter")

	if err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	if !strings.Contains(err.Error(), alphaID) {
		t.Errorf("the error %q does not name the plugin", err)
	}
	if strings.Contains(err.Error(), workValue) {
		t.Errorf("the error holds the value: %q", err)
	}
	if got := allowedOf(w.find(t, one)); len(got) != 1 {
		t.Errorf("allowed = %v, want the permission kept", got)
	}
	if w.keys.snapshot()["secret/"+one] != workValue {
		t.Errorf("the key store holds %v", w.keys.snapshot())
	}
	if failure := w.getFailure(alphaID); failure != nil {
		t.Errorf("the plugin page shows %+v", failure)
	}
}

// A plugin whose Configure failed receives the list from the next refresh, and from the next
// operation that changes its list.
func TestAPluginWhoseConfigureFailedReceivesTheListFromTheNextRefreshAndOperation(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	alpha := w.probes[alphaID]
	alpha.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	one := w.create(t, "openrouter", "Work", workValue)
	if err := w.tryAllow(t.Context(), one, alphaID, "apiKey", "openrouter"); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}

	alpha.setFailure(nil)
	if err := w.host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	w.assertIDs(t, alphaID, "apiKey", one)

	alpha.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	two := w.create(t, "openrouter", "Home", "value-of-the-home-key")
	if err := w.tryAllow(t.Context(), two, alphaID, "apiKey", "openrouter"); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	alpha.setFailure(nil)
	three := w.create(t, "openrouter", "Third", "value-of-the-third-key")
	w.allow(t, three, alphaID, "apiKey", "openrouter")
	w.assertIDs(t, alphaID, "apiKey", one, two, three)
}

// A plugin that never answers delays the operation by the Configure timeout, once, and the operation
// still commits and names the plugin.
func TestAHungConfigureDelaysTheOperationByTheTimeoutOnceAndCommits(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	alpha := w.probes[alphaID]
	hold := make(chan struct{})
	alpha.mu.Lock()
	alpha.hold = hold
	alpha.mu.Unlock()
	t.Cleanup(func() { close(hold) })
	one := w.create(t, "openrouter", "Work", workValue)

	start := time.Now()
	err := w.tryAllow(t.Context(), one, alphaID, "apiKey", "openrouter")
	elapsed := time.Since(start)

	if elapsed < configureTimeout-500*time.Millisecond || elapsed > configureTimeout+1500*time.Millisecond {
		t.Errorf("the operation took %v, want about the Configure timeout of %v", elapsed, configureTimeout)
	}
	if err == nil || !strings.Contains(err.Error(), alphaID) {
		t.Errorf("the error = %v, want one that names the plugin", err)
	}
	if got := allowedOf(w.find(t, one)); len(got) != 1 {
		t.Errorf("allowed = %v, want the permission kept", got)
	}
}

// A failing or hung plugin does not fail or delay an operation that leaves its list as it is, and
// the operation sends it nothing.
func TestAFailingPluginDoesNotFailOrDelayAnOperationThatDoesNotConcernIt(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	alpha := w.probes[alphaID]
	hold := make(chan struct{})
	alpha.mu.Lock()
	alpha.hold = hold
	alpha.mu.Unlock()
	t.Cleanup(func() { close(hold) })
	one := w.create(t, "openai", "Work", workValue)
	attempts := alpha.attemptCount()

	start := time.Now()
	w.allow(t, one, betaID, "apiKey", "openai")
	elapsed := time.Since(start)

	if elapsed > configureTimeout/2 {
		t.Errorf("the operation took %v, it waited for a plugin it does not concern", elapsed)
	}
	if alpha.attemptCount() != attempts {
		t.Errorf("the plugin received %d calls, want %d", alpha.attemptCount(), attempts)
	}
	w.assertIDs(t, betaID, "apiKey", one)
}
