package hostendpoint

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugindependency"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type declaredModels map[string]manifest.Manifest

func (d declaredModels) FindModelDependency(pluginID, id string) (manifest.DependencyRequirement, bool) {
	m, ok := d[pluginID]
	if !ok {
		return manifest.DependencyRequirement{}, false
	}
	return m.FindModelDependency(id)
}

func readSpeechManifest(t *testing.T) manifest.Manifest {
	t.Helper()
	m, err := manifestjson.Read(strings.NewReader(`{"id": "md.numen.a", "name": "Speech", "version": "1.0.0", "backend": {"entrypoint": "bin/speech"},
		"dependencies": [{"id": "speech-encoder", "kind": "model", "version": "r1", "sourceUrl": "https://models.invalid/encoder.onnx",
			"checksumSha256": "` + strings.Repeat("e", 64) + `", "sizeBytes": 4096}]}`))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTheHostFetchesOnlyAModelTheManifestOfTheCallingPluginDeclares(t *testing.T) {
	registry := provision.NewObservedRegistry(nil, provision.NewPlaceholderProvisioner(t.TempDir()))
	callers := newCallers().asCallers()
	callers.DeclaredDependencies = plugindependency.NewFetchDeclaredModel(
		declaredModels{"md.numen.a": readSpeechManifest(t)},
		func() pluginport.DependencyRegistry { return registry },
	)
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostDependencyServiceHandler(&dependencyEndpoint{endpoint: &Endpoint{callers: callers}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewHostDependencyServiceClient(server.Client(), server.URL)
	call := func(token, id string) (*connect.Response[v1.FetchDependencyResponse], error) {
		req := connect.NewRequest(&v1.FetchDependencyRequest{Id: id})
		req.Header().Set("Authorization", "Bearer "+token)
		return client.FetchDependency(t.Context(), req)
	}

	if _, err := call("unrecognized", "speech-encoder"); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("an unrecognized token = %v, want Unauthenticated", err)
	}
	for _, asked := range []struct{ token, id string }{{"token-b", "speech-encoder"}, {"token-a", "absent-model"}} {
		refused, err := call(asked.token, asked.id)
		if err != nil {
			t.Fatal(err)
		}
		if refused.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NOT_FOUND || refused.Msg.GetDetail() == "" || refused.Msg.GetPath() != "" {
			t.Errorf("%s asking for %q = %v, want not found with a detail", asked.token, asked.id, refused.Msg)
		}
	}
	answer, err := call("token-a", "speech-encoder")
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the declared model = %v", answer.Msg)
	}
	if _, err := os.Stat(answer.Msg.GetPath()); err != nil {
		t.Errorf("the path of the declared model: %v", err)
	}
}
