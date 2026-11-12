package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
)

// The handler of a host that links the plugin in serves the local model it is configured for.
func TestTheLinkedHandlerServesTheModelItsConfigureNames(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": linkedToken})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	token := hostcall.WithBearer(linkedToken)

	if _, err := servertest.NewLifecycleClient(web.Client(), web.URL, token).Configure(t.Context(),
		map[string]string{"source": "local", "model": "Xenova/bge-m3"}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	info := getModel(t, web)

	if info.Name != "Xenova/bge-m3" || info.Width != 1024 {
		t.Errorf("model = %s at %d, want Xenova/bge-m3 at 1024", info.Name, info.Width)
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

// linkedToken is the token the host gives the plugin it links in.
const linkedToken = "linked-token"

// servedModel is the part of the answer of DescribeSearchSource the tests read.
type servedModel struct {
	Name  string `json:"name"`
	Width int32  `json:"width"`
}

// getModel asks the search source service of the plugin for its description over Connect JSON, as the host does.
func getModel(t *testing.T, web *httptest.Server) servedModel {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, web.URL+"/numen.v1.SearchSourceService/DescribeSearchSource", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+linkedToken)
	res, err := web.Client().Do(req)
	if err != nil {
		t.Fatalf("DescribeSearchSource: %v", err)
	}
	defer res.Body.Close()
	var info servedModel
	if res.StatusCode != http.StatusOK {
		t.Fatalf("DescribeSearchSource answered %s", res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		t.Fatalf("DescribeSearchSource: %v", err)
	}
	return info
}
