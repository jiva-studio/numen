package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health/healthtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
)

// The handler of a host that links the plugin in serves the local model it is configured for.
func TestTheLinkedHandlerServesTheModelItsConfigureNames(t *testing.T) {
	setEmptyModelDir(t)
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": linkedToken})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	if _, err := servertest.NewLifecycleClient(web.Client(), web.URL, linkedToken).Configure(t.Context(),
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
	s := server.New()
	if err := Mount(s); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("s.Close: %v", err)
	}
}

// setEmptyModelDir points the local model at an empty folder, so that the load Configure starts fails at
// once instead of downloading.
func setEmptyModelDir(t *testing.T) {
	t.Helper()
	t.Setenv("NUMEN_EMBEDDING_DIR", t.TempDir())
}

// linkedToken is the token the host gives the plugin it links in.
const linkedToken = "linked-token"

// servedModel is the part of the answer of DescribeSearchSource the tests read.
type servedModel struct {
	Name  string `json:"name"`
	Width int32  `json:"width"`
}

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
	t.Cleanup(func() {
		if err := res.Body.Close(); err != nil {
			t.Error(err)
		}
	})
	var info servedModel
	if res.StatusCode != http.StatusOK {
		t.Fatalf("DescribeSearchSource answered %s", res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		t.Fatalf("DescribeSearchSource: %v", err)
	}
	return info
}

func describeStatus(t *testing.T, web *httptest.Server) int {
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
	if err := res.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return res.StatusCode
}

// A local model the plugin has no profile for is reported to the host as the source item, with the
// settings of the source as its fix, and the item is cleared once a model it knows answers.
func TestAMissingLocalModelIsReportedAsTheSourceItemUntilAKnownModelAnswers(t *testing.T) {
	setEmptyModelDir(t)
	hostHealth := healthtest.NewHost(t)
	handler, err := NewHandler(hostHealth.Vars(linkedToken))
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	lifecycle := servertest.NewLifecycleClient(web.Client(), web.URL, linkedToken)

	if _, err := lifecycle.Configure(t.Context(), map[string]string{"source": "local", "model": "Example/no-such-model"}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if status := describeStatus(t, web); status == http.StatusOK {
		t.Fatal("a model without a profile was described")
	}
	items := hostHealth.List()
	if len(items) != 1 || items[0].Key != "source" || items[0].Severity != health.Error || items[0].Text == "" {
		t.Fatalf("items = %+v, want one error item under source", items)
	}
	if action := items[0].Action; action == nil || action.Command != "app:settings.open" || action.Values["setting"] != "source" {
		t.Fatalf("action = %+v, want the settings of the source", items[0].Action)
	}

	if _, err := lifecycle.Configure(t.Context(), map[string]string{"source": "local", "model": "Xenova/bge-m3"}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if status := describeStatus(t, web); status != http.StatusOK {
		t.Fatalf("DescribeSearchSource answered %d for a known model", status)
	}
	if items := hostHealth.List(); len(items) != 0 {
		t.Errorf("items = %+v after a known model answered, want none", items)
	}
}
