package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/api"
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

func TestTheStatsCarryTheCountsTheModelAndTheProfile(t *testing.T) {
	view := newStats(newIndex(), clock.NewFake(getStart()), 16)
	view.configure(map[string]string{"host.performanceProfile": "maximum"})

	got, err := view.get(t.Context(), api.GetStatsRequest{})

	if err != nil {
		t.Fatal(err)
	}
	want := api.Stats{
		Vectors: 12480, Chunks: 13100, Waiting: 620, Sources: 84,
		Kinds: []api.KindCount{
			{Kind: api.KindNotes, Sources: 70, Chunks: 4100},
			{Kind: api.KindPDF, Sources: 9, Chunks: 7000},
			{Kind: api.KindEPUB, Sources: 5, Chunks: 2000},
		},
		Model: "bge-m3", Recipe: recipe, Dimensions: 1024,
		Profile: "maximum", Threads: 8, Speed: nil, ChangedAt: "",
		DatabaseBytes: 73400320, VectorBytes: 52428800,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestTheProfileIsBalancedUntilTheHostConfiguresThePlugin(t *testing.T) {
	view := newStats(newIndex(), clock.NewFake(getStart()), 16)

	got, err := view.get(t.Context(), api.GetStatsRequest{})

	if err != nil || got.Profile != "balanced" || got.Threads != 4 {
		t.Errorf("profile %q with %d threads (%v), want balanced with 4", got.Profile, got.Threads, err)
	}
}

func TestTheSpeedAndTheTimeOfTheLastChangeFollowTheCallsOfTheView(t *testing.T) {
	index := newIndex()
	now := clock.NewFake(getStart())
	view := newStats(index, now, 8)
	ctx := t.Context()

	first, err := view.get(ctx, api.GetStatsRequest{})
	if err != nil || first.Speed != nil || first.ChangedAt != "" {
		t.Fatalf("first call: speed %v, changed at %q (%v)", first.Speed, first.ChangedAt, err)
	}

	now.Advance(2 * time.Second)
	index.SetProgress(13100, 12520)
	second, err := view.get(ctx, api.GetStatsRequest{})
	if err != nil || second.Speed == nil || *second.Speed != 20 || second.ChangedAt != "2026-11-02T12:00:02Z" {
		t.Errorf("second call: speed %v, changed at %q (%v), want 20 and 12:00:02", second.Speed, second.ChangedAt, err)
	}

	now.Advance(time.Second)
	index.SetProgress(13100, 13100)
	done, err := view.get(ctx, api.GetStatsRequest{})
	if err != nil || done.Speed != nil || done.Waiting != 0 || done.ChangedAt != "2026-11-02T12:00:03Z" {
		t.Errorf("embedded: speed %v, waiting %d, changed at %q (%v)", done.Speed, done.Waiting, done.ChangedAt, err)
	}
}

func TestAnIndexWithMoreVectorsThanChunksWaitsForNothing(t *testing.T) {
	index := newIndex()
	index.SetProgress(10, 12)

	got, err := newStats(index, clock.NewFake(getStart()), 8).get(t.Context(), api.GetStatsRequest{})

	if err != nil || got.Waiting != 0 {
		t.Errorf("waiting %d (%v), want 0", got.Waiting, err)
	}
}

func TestAHostWithoutAModelOrAVaultIsAFailedPrecondition(t *testing.T) {
	noModel := hosttest.NewSearch("physics")
	noVault := newIndex()
	noVault.HideVault()
	for name, index := range map[string]Index{"no model": noModel, "no vault": noVault} {
		_, err := newStats(index, clock.NewFake(getStart()), 8).get(t.Context(), api.GetStatsRequest{})

		var failure *server.APIError
		if !errors.As(err, &failure) || failure.Code != server.CodeFailedPrecondition {
			t.Errorf("%s: error = %v, want failed_precondition", name, err)
		}
	}
}

func TestAPluginWithoutThePermissionIsDeniedAndAnyOtherFailureIsInternal(t *testing.T) {
	denied := newIndex()
	denied.Deny(host.PermissionIndexRead)
	_, err := newStats(denied, clock.NewFake(getStart()), 8).get(t.Context(), api.GetStatsRequest{})
	var failure *server.APIError
	if !errors.As(err, &failure) || failure.Code != server.CodePermissionDenied {
		t.Errorf("error = %v, want permission_denied", err)
	}

	_, err = newStats(failingIndex{}, clock.NewFake(getStart()), 8).get(t.Context(), api.GetStatsRequest{})
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
		return mount(s, newIndex(), clock.NewFake(getStart()), 16)
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
