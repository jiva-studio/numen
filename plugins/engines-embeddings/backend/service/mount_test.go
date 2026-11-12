package service

import (
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchsource"
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
	info, err := searchsource.NewClient(web.Client(), web.URL, token).Describe(t.Context())
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}

	if info.Name != "Xenova/bge-m3" || info.Width != 1024 {
		t.Errorf("model = %s at %d, want Xenova/bge-m3 at 1024", info.Name, info.Width)
	}
}

func TestMountRegistersCloseCallback(t *testing.T) {
	t.Setenv("NUMEN_MOCK_ML", "true")
	s := server.New(server.WithToken("token"))
	if err := Mount(s); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("s.Close: %v", err)
	}
}
