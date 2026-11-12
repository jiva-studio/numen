package service

import (
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// The handler of a host that links the plugin in serves the local model it is configured for.
func TestTheLinkedHandlerServesTheModelItsConfigureNames(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	token := hostcall.WithBearer("linked-token")

	if _, err := server.NewLifecycleClient(web.Client(), web.URL, token).Configure(t.Context(),
		map[string]string{"source": "local", "model": "Xenova/bge-m3"}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	info, err := embedding.NewClient(web.Client(), web.URL, token).GetModelInfo(t.Context())
	if err != nil {
		t.Fatalf("GetModelInfo: %v", err)
	}

	if info.Name != "Xenova/bge-m3" || info.Dimensions != 1024 {
		t.Errorf("model = %s at %d, want Xenova/bge-m3 at 1024", info.Name, info.Dimensions)
	}
}

func TestMountRegistersCloseCallback(t *testing.T) {
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	s := server.New(server.WithToken("token"))
	if err := Mount(s); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("s.Close: %v", err)
	}
}
