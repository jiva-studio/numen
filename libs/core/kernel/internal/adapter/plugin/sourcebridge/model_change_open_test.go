package sourcebridge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const embeddingPluginID = "md.numen.embeddings"

// failingActivator is a plugin host that cannot start any plugin.
type failingActivator struct{}

func (failingActivator) EnsureActive(context.Context, string) (transport.Endpoint, error) {
	return transport.Endpoint{}, errors.New("the plugin did not start")
}

func (failingActivator) Release(string) {}

func (a failingActivator) EnsureActiveForQuestion(ctx context.Context, pluginID string) (transport.Endpoint, error) {
	return a.EnsureActive(ctx, pluginID)
}

func (failingActivator) ReleaseQuestion(string) {}

func (failingActivator) ResolveCapability(string) (manifest.Discovered, error) {
	return manifest.Discovered{Manifest: manifest.Manifest{ID: embeddingPluginID}}, nil
}

func (failingActivator) WatchPluginConfig(string, func()) func() { return func() {} }

// A plugin that cannot be opened after a setting is reported with its id, and no listener runs.
func TestEmbedderReportsFailedOpenAfterSettingChange(t *testing.T) {
	b := &Source{
		service: bridge.Service[numenv1connect.SearchSourceServiceClient]{
			Host: failingActivator{}, PluginID: embeddingPluginID, NewClient: numenv1connect.NewSearchSourceServiceClient,
		},
	}
	var reports []error
	b.OnDescriptionReadError(func(err error) { reports = append(reports, err) })
	ran := 0
	b.OnDescriptionChange(func() { ran++ })

	b.announceDescriptionChange()

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
