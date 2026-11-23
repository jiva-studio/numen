package service

import (
	"context"
	"encoding/json"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/hostmodel"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health/healthtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
)

func TestTheMountedPluginServesTheModelItsConfigureNames(t *testing.T) {
	setEmptyModelDir(t)
	handler := newMountedHandler(t, map[string]string{hostcall.PluginTokenEnv: linkedToken})
	web := httptest.NewServer(handler)
	defer web.Close()

	if err := servertest.NewLifecycleClient(web.Client(), web.URL, linkedToken).Configure(t.Context(),
		map[string]string{"source": "local", "model": "Xenova/bge-m3"}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	info := getModel(t, web)

	if info.Name != "Xenova/bge-m3" || info.Width != 1024 {
		t.Errorf("model = %s at %d, want Xenova/bge-m3 at 1024", info.Name, info.Width)
	}
}

func mountMockPlugin(t *testing.T) (http.Handler, *server.Server) {
	t.Helper()
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	var mounted *server.Server
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": linkedToken}, func(s *server.Server) error {
		mounted = s
		return mount(s, hosttest.NewQueue().EnqueueWork, hostmodel.OpenFromEnvironment, getPlaces(host.NewSearchClientFromEnv), health.NewClientFromEnv)
	})
	if err != nil {
		t.Fatalf("NewLinkedHandler: %v", err)
	}
	return handler, mounted
}

func configureLocal(t *testing.T, handler http.Handler) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		"http://plugin/numen.v1.PluginLifecycleService/Configure", strings.NewReader(`{"config":{"source":"local"}}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+linkedToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Configure status = %d, want 200", rec.Code)
	}
}

func embedQuery(t *testing.T, handler http.Handler, within time.Duration) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), within)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://plugin/numen.v1.SearchSourceService/EmbedQueries", strings.NewReader(`{"queries":["boat"]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+linkedToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code
}

func TestConfiguringThePluginDoesNotLoadTheModel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		handler, mounted := mountMockPlugin(t)
		configureLocal(t, handler)
		synctest.Wait()

		t.Setenv("NUMEN_TEST_MOCK_OPEN_HOLD_MS", "60000")
		if code := embedQuery(t, handler, 10*time.Millisecond); code == http.StatusOK {
			t.Error("the first embed after Configure found the model already loaded")
		}

		if err := mounted.Close(); err != nil {
			t.Errorf("mounted.Close: %v", err)
		}
	})
}

func TestTheModelLoadsOnTheFirstEmbed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		handler, mounted := mountMockPlugin(t)
		t.Cleanup(func() {
			if err := mounted.Close(); err != nil {
				t.Errorf("mounted.Close: %v", err)
			}
		})
		configureLocal(t, handler)
		synctest.Wait()

		const loadTime = 200 * time.Millisecond
		t.Setenv("NUMEN_TEST_MOCK_OPEN_HOLD_MS", "200")
		codes := make(chan int, 1)
		go func() { codes <- embedQuery(t, handler, time.Minute) }()

		time.Sleep(loadTime - time.Millisecond)
		synctest.Wait()
		select {
		case code := <-codes:
			t.Fatalf("the first embed returned %d before the %v the model takes to load", code, loadTime)
		default:
		}

		time.Sleep(time.Millisecond)
		synctest.Wait()
		select {
		case code := <-codes:
			if code != http.StatusOK {
				t.Fatalf("EmbedQueries status = %d, want 200", code)
			}
		default:
			t.Fatal("the first embed did not return after the model loaded")
		}
	})
}

func newMountedHandler(t *testing.T, vars map[string]string) http.Handler {
	t.Helper()
	for name, value := range vars {
		t.Setenv(name, value)
	}
	handler, err := servertest.NewHandler(vars, Mount)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return handler
}

func setEmptyModelDir(t *testing.T) {
	t.Helper()
	// An empty folder makes the load Configure starts fail at once instead of downloading.
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
	handler := newMountedHandler(t, hostHealth.BuildVars(linkedToken))
	web := httptest.NewServer(handler)
	defer web.Close()
	lifecycle := servertest.NewLifecycleClient(web.Client(), web.URL, linkedToken)

	if err := lifecycle.Configure(t.Context(), map[string]string{"source": "local", "model": "Example/no-such-model"}); err != nil {
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

	if err := lifecycle.Configure(t.Context(), map[string]string{"source": "local", "model": "Xenova/bge-m3"}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if status := describeStatus(t, web); status != http.StatusOK {
		t.Fatalf("DescribeSearchSource answered %d for a known model", status)
	}
	if items := hostHealth.List(); len(items) != 0 {
		t.Errorf("items = %+v after a known model answered, want none", items)
	}
}
