package container

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func TestWindowPreparesModelFilesThroughThePluginHostRegistry(t *testing.T) {
	t.Setenv(provision.EnvMockML, "true")
	t.Setenv("NUMEN_MODELS_DIR", t.TempDir())
	host := newTestPluginRoles(testsupport.GetMachine(), "desktop.editor")
	host.Supervision.Dependencies.Replace(newDependencyRegistry(host.Supervision.Clock, ""))
	dir := t.TempDir()
	cfg := NewTestInstallation(Config{Paths: Paths{
		IndexPath: filepath.Join(dir, "index.db"), RegistryPath: filepath.Join(dir, "vaults.json"),
	}}, AdaptTestPlugins(host.Management, host.Supervision))
	window, err := cfg.OpenWindow(cfg.Execution, cfg.Preferences, cfg.Endpoints, cfg.Resources, t.Context(), WindowOpening{})
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, window)
	server := httptest.NewServer(window.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostDependencyServiceClient(server.Client(), server.URL)
	answer, err := client.FetchModelFile(t.Context(), connect.NewRequest(&v1.FetchModelFileRequest{
		Url: "https://models.invalid/weights", Sha256: "a431985659dc921974177a95adcfbb90fd9e51989a5e04d70d0b75f597b6e61d",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetPath() == "" || answer.Msg.GetError() != "" {
		t.Fatalf("prepare model: %v", answer.Msg)
	}
}
