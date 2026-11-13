package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/usecase/indexstats"
)

const recipe = "local|bge-m3|1024|512|mean|int8@0.4"

func getStart() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC) }

func newIndex() *hosttest.Search {
	search := hosttest.NewSearch("physics")
	search.SetModel(hosttest.SearchModel{Name: "bge-m3", Recipe: recipe, FloorHint: 0})
	search.SetProgress(13100, 12480)
	search.SetIndexStats(host.IndexStats{
		Recipe: "", Model: "", Dimensions: 1024, ChunkCount: 0, VectorCount: 0,
		SourceCount: 84,
		Extensions: []host.ExtensionStats{
			{Extension: "md", SourceCount: 70, ChunkCount: 4100},
			{Extension: "pdf", SourceCount: 9, ChunkCount: 7000},
			{Extension: "epub", SourceCount: 5, ChunkCount: 2000},
		},
		DatabaseBytes: 73400320, VectorBytes: 52428800,
	})
	return search
}

func TestAHostWithoutAModelOrAVaultIsAFailedPrecondition(t *testing.T) {
	noModel := hosttest.NewSearch("physics")
	noVault := newIndex()
	noVault.HideVault()
	for name, index := range map[string]indexstats.Index{"no model": noModel, "no vault": noVault} {
		_, err := getStats(indexstats.NewStats(index, newFakeClock(getStart()), 8))(t.Context(), api.GetStatsRequest{})

		var failure *server.APIError
		if !errors.As(err, &failure) || failure.Code != server.CodeFailedPrecondition {
			t.Errorf("%s: error = %v, want failed_precondition", name, err)
		}
	}
}

func TestAPluginWithoutThePermissionIsDeniedAndAnyOtherFailureIsInternal(t *testing.T) {
	denied := newIndex()
	denied.Deny(host.PermissionIndexRead)
	_, err := getStats(indexstats.NewStats(denied, newFakeClock(getStart()), 8))(t.Context(), api.GetStatsRequest{})
	var failure *server.APIError
	if !errors.As(err, &failure) || failure.Code != server.CodePermissionDenied {
		t.Errorf("error = %v, want permission_denied", err)
	}

	_, err = getStats(indexstats.NewStats(failingIndex{}, newFakeClock(getStart()), 8))(t.Context(), api.GetStatsRequest{})
	if !errors.Is(err, errHostUnreachable) || errors.As(err, &failure) {
		t.Errorf("error = %v, want wrapped errHostUnreachable", err)
	}
}

var errHostUnreachable = errors.New("host unreachable")

type failingIndex struct{}

func (failingIndex) GetIndexStats(context.Context) (host.IndexStats, error) {
	return host.IndexStats{}, errHostUnreachable
}

func post(t *testing.T, web *httptest.Server, path, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, web.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer plugin-token")
	res, err := web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, data
}

func TestTheHostConfiguresThePluginThatThenAnswersTheAPI(t *testing.T) {
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "plugin-token"}, func(s *server.Server) error {
		return mount(s, indexstats.NewStats(newIndex(), newFakeClock(getStart()), 16))
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	configured := `{"values":{"host.performanceProfile":"eco"}}`
	if status, body := post(t, web, "/numen.v1.PluginLifecycleService/Configure", configured); status != http.StatusOK {
		t.Fatalf("Configure = %d %s", status, body)
	}
	status, body := post(t, web, "/"+api.Service+"/"+api.MethodGetStats, `{}`)

	var got api.Stats
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || got.Vectors != 12480 || got.Profile != "eco" || got.Threads != 1 {
		t.Errorf("GetStats = %d %s", status, body)
	}
}

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}
