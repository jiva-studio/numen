package embedderbridge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

const embeddingPluginID = "md.numen.embeddings"

// failingActivator is a plugin host that cannot activate any plugin.
type failingActivator struct{}

func (failingActivator) EnsureActive(context.Context, string) (transport.Endpoint, error) {
	return transport.Endpoint{}, errors.New("the plugin did not start")
}

func (failingActivator) Release(string) {}

// A plugin that cannot be opened after a setting is reported with its id, and no listener runs.
func TestEmbedderReportsFailedOpenAfterSettingChange(t *testing.T) {
	b := &Embedder{
		pluginID: embeddingPluginID,
		service: bridge.Service[numenv1connect.EmbeddingServiceClient]{
			Host: failingActivator{}, PluginID: embeddingPluginID, NewClient: numenv1connect.NewEmbeddingServiceClient,
		},
	}
	var reports []error
	b.OnModelReadError(func(err error) { reports = append(reports, err) })
	ran := 0
	b.OnModelChange(func() { ran++ })

	b.announceModelChange()

	if len(reports) != 1 {
		t.Fatalf("%d reports for one failed open, want 1", len(reports))
	}
	if !strings.Contains(reports[0].Error(), embeddingPluginID) {
		t.Errorf("the error %q does not name the plugin %s", reports[0], embeddingPluginID)
	}
	if ran != 0 {
		t.Errorf("a listener ran %d times for a plugin that was not opened", ran)
	}
}
