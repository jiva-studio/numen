package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process/processtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/internal/adapter/floorrecords"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/relevance"
)

const (
	maxNotes    = 100
	token       = "plugin-token"
	modelRecipe = "test|test-model|4|0|mean|int8@0.4"
)

func getMeasuredAt() time.Time {
	return time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
}

// fixture is a plugin over a host that shows an embedded vault of 25 notes.
type fixture struct {
	app       *service.App
	search    *hosttest.Search
	vault     *hosttest.Vault
	queue     *hosttest.Queue
	processes *processtest.Host
	web       *httptest.Server
}

func sampledChunks(n, sources int) []hosttest.SampledChunk {
	out := make([]hosttest.SampledChunk, n)
	for i := range out {
		out[i] = hosttest.SampledChunk{Path: fmt.Sprintf("source-%d.md", 100+i%sources), Vector: []float32{64, 64, 64, 64}}
	}
	return out
}

func setNotes(vault *hosttest.Vault, n int) {
	for i := 1; i <= maxNotes; i++ {
		if i > n {
			vault.Remove(fmt.Sprintf("note-%d.md", i))
			continue
		}
		vault.File(fmt.Sprintf("note-%d.md", i), "---\ntitle: title\n---\n")
	}
}

func newFixture(t *testing.T, hint float64) *fixture {
	t.Helper()
	search := hosttest.NewSearch("physics", "chemistry")
	search.SetModel(hosttest.SearchModel{Name: "test-model", Recipe: modelRecipe, FloorHint: hint})
	search.SetEmbedder(func(string) []float32 { return []float32{1, 0, 0, 0} })
	search.SetSampledChunks(modelRecipe, sampledChunks(60, 3))
	search.SetProgress(60, 60)

	vault := hosttest.New("physics")
	vault.Beside(hosttest.New("chemistry"))
	setNotes(vault, 25)
	queue := hosttest.NewQueue()
	processes := processtest.NewHost(t)
	app := service.NewAppOver(service.Hosts{
		Floors: relevance.Host{Search: searchOf(search), Vaults: func(id domain.VaultID) host.NoteLookup { return vault.With(id) }, Store: foldersOf(vault)},
		Queue:  queue,
	}, newFakeClock(getMeasuredAt()), processes.NewClient(t))

	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, app.Mount)
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return &fixture{app: app, search: search, vault: vault, queue: queue, processes: processes, web: web}
}

func (f *fixture) call(t *testing.T, method string, req, res any) (int, server.APIError) {
	t.Helper()
	return f.callWith(t, token, method, req, res)
}

func (f *fixture) callWith(t *testing.T, bearer, method string, req, res any) (int, server.APIError) {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.web.URL+"/"+api.Service+"/"+method, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	response, err := f.web.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	})
	var failure server.APIError
	if response.StatusCode != http.StatusOK {
		if err := json.NewDecoder(response.Body).Decode(&failure); err != nil && response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status %d: %v", response.StatusCode, err)
		}
		return response.StatusCode, failure
	}
	if err := json.NewDecoder(response.Body).Decode(res); err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, failure
}

func TestTheServiceCalibratesSetsAndReadsTheFloor(t *testing.T) {
	f := newFixture(t, 0.4)

	var got api.GetFloorResponse
	if status, failure := f.call(t, api.MethodGetFloor, api.GetFloorRequest{}, &got); status != http.StatusOK {
		t.Fatalf("get: %d %v", status, failure)
	}
	if got.Source != api.SourceModel || got.Floor != 0.4 || got.Model != "test-model" || got.Calibration != nil {
		t.Fatalf("before anything is kept: %+v", got)
	}

	var calibrated api.CalibrateFloorResponse
	if status, failure := f.call(t, api.MethodCalibrateFloor, api.CalibrateFloorRequest{}, &calibrated); status != http.StatusOK {
		t.Fatalf("calibrate: %d %v", status, failure)
	}
	state := calibrated.State
	if calibrated.Outcome != api.OutcomeCalibrated || state.Source != api.SourceCalibrated || state.Floor < 0.51 || state.Floor > 0.53 {
		t.Errorf("calibrated %+v", calibrated)
	}
	c := state.Calibration
	if c == nil || c.QueryCount != 25 || c.ChunkCount != 60 || len(c.Histogram) != 20 || c.Histogram[10] != 25*60 ||
		c.CalibratedAt != getMeasuredAt().Unix() {
		t.Errorf("calibration %+v", c)
	}

	var set api.SetFloorResponse
	if status, failure := f.call(t, api.MethodSetFloor, api.SetFloorRequest{IsManual: true, Floor: 0.7}, &set); status != http.StatusOK {
		t.Fatalf("set: %d %v", status, failure)
	}
	if set.State.Source != api.SourceManual || set.State.Floor != 0.7 || !set.State.IsManual || set.State.ManualFloor != 0.7 {
		t.Errorf("set %+v", set)
	}
	if stored, _ := f.search.StoredFloor(modelRecipe); stored != 0.7 {
		t.Errorf("the host applies %v, want 0.7", stored)
	}

	f.search.SetSampledChunks(modelRecipe, []hosttest.SampledChunk{{Path: "note-1.md", Vector: []float32{127, 0, 0, 0}}})
	var probe api.ProbeFloorResponse
	if status, failure := f.call(t, api.MethodProbeFloor, api.ProbeFloorRequest{Query: "entropy"}, &probe); status != http.StatusOK {
		t.Fatalf("probe: %d %v", status, failure)
	}
	if probe.BestSimilarity < 0.99 || probe.Floor != 0.7 || !probe.DoesPass {
		t.Errorf("probe %+v", probe)
	}
}

func TestTheServiceAnswersCodesForWhatItCannotServe(t *testing.T) {
	f := newFixture(t, 0.4)

	_, failure := f.call(t, api.MethodSetFloor, api.SetFloorRequest{IsManual: true, Floor: 2}, &api.SetFloorResponse{})
	if failure.Code != server.CodeInvalidArgument {
		t.Errorf("a floor above one gave %+v", failure)
	}
	f.search.FailEmbedding(errors.New("the plugin is not running"))
	_, failure = f.call(t, api.MethodCalibrateFloor, api.CalibrateFloorRequest{}, &api.CalibrateFloorResponse{})
	if failure.Code != server.CodeUnavailable {
		t.Errorf("a model that fails gave %+v", failure)
	}
	_, failure = f.call(t, api.MethodProbeFloor, api.ProbeFloorRequest{Query: "entropy"}, &api.ProbeFloorResponse{})
	if failure.Code != server.CodeUnavailable {
		t.Errorf("a probe on a model that fails gave %+v", failure)
	}
	f.search.Deny(host.PermissionIndexRead)
	_, failure = f.call(t, api.MethodGetFloor, api.GetFloorRequest{}, &api.GetFloorResponse{})
	if failure.Code != server.CodePermissionDenied {
		t.Errorf("a permission the manifest lacks gave %+v", failure)
	}
}

func TestACallWithoutTheTokenOfTheHostIsRefused(t *testing.T) {
	f := newFixture(t, 0.4)
	status, _ := f.callWith(t, "", api.MethodGetFloor, api.GetFloorRequest{}, &api.GetFloorResponse{})
	if status != http.StatusUnauthorized {
		t.Errorf("a call without the token gave status %d", status)
	}
	status, _ = f.callWith(t, "another-token", api.MethodGetFloor, api.GetFloorRequest{}, &api.GetFloorResponse{})
	if status != http.StatusUnauthorized {
		t.Errorf("a call with another token gave status %d", status)
	}
}

func TestASearchSourceReadyEventMeasuresTheVaultOnceHoweverOftenItRepeats(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t, 0.4)
	event := getReadyEvent("physics")

	for range 3 {
		if err := f.app.HandleEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	if batches := f.search.EmbeddedBatches(); len(batches) != 1 {
		t.Errorf("the vault was measured %d times, want once", len(batches))
	}
	if stored, found := f.search.StoredFloor(modelRecipe); !found || stored < 0.51 || stored > 0.53 {
		t.Errorf("the host applies %v (%v), want the measured floor", stored, found)
	}
	if f.search.FloorWrites() != 1 {
		t.Errorf("the floor was written %d times, want once", f.search.FloorWrites())
	}
}

func TestAnEventOtherThanSearchSourceReadyMeasuresNothing(t *testing.T) {
	f := newFixture(t, 0.4)

	if err := f.app.HandleEvent(t.Context(), "onVaultOpened:physics"); err != nil {
		t.Fatal(err)
	}

	if _, found := f.search.StoredFloor(modelRecipe); found || len(f.search.EmbeddedBatches()) != 0 {
		t.Error("an event other than search source ready measured the vault")
	}
}

func TestAnEventForAnotherVaultMeasuresThatVault(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t, 0.4)
	if err := f.app.HandleEvent(ctx, getReadyEvent("chemistry")); err != nil {
		t.Fatal(err)
	}
	if _, found := f.search.StoredFloor(modelRecipe); found {
		t.Error("the shown vault was given the floor measured on another")
	}
	if batches := f.search.EmbeddedBatches(); len(batches) != 0 {
		t.Errorf("an empty vault was measured: %d", len(batches))
	}
	err := f.app.HandleEvent(ctx, getReadyEvent("biology"))
	if !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("an unknown vault gave %v", err)
	}
}

func TestAFailedMeasurementIsListedAsATaskAndAFinishedOneRemovesIt(t *testing.T) {
	ctx := t.Context()
	f := newFixture(t, 0.4)
	event := getReadyEvent("physics")

	f.search.FailEmbedding(errors.New("the plugin is not running"))
	if err := f.app.HandleEvent(ctx, event); !errors.Is(err, host.ErrEmbeddingFailed) {
		t.Fatalf("got %v, want the failure of the model", err)
	}
	tasks := f.queue.ListTasks()
	if len(tasks) != 1 || tasks[0].FailureReason == "" || tasks[0].CompletedCount != 0 || tasks[0].TotalCount != 0 {
		t.Fatalf("tasks %+v, want the failure listed with zero counts", tasks)
	}

	f.search.FailEmbedding(nil)
	if err := f.app.HandleEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if tasks := f.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %v, want the failure taken out", tasks)
	}

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	if err := f.app.HandleEvent(canceledCtx, event); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled event gave %v, want context.Canceled", err)
	}
	if tasks := f.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("canceled event listed tasks %v, want none", tasks)
	}
}

type failingQueue struct {
	host.Queue
	reportErr error
	finishErr error
}

func (q failingQueue) ReportTask(ctx context.Context, task host.Task) error {
	if q.reportErr != nil {
		return q.reportErr
	}
	return q.Queue.ReportTask(ctx, task)
}

func (q failingQueue) FinishTask(ctx context.Context, id string) error {
	if q.finishErr != nil {
		return q.finishErr
	}
	return q.Queue.FinishTask(ctx, id)
}

func TestMountAndEventReportingValidateDependenciesAndQueueErrors(t *testing.T) {
	if _, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, service.NewApp().Mount); !errors.Is(err, hostcall.ErrNoHost) {
		t.Errorf("Mount without host env gave %v, want ErrNoHost", err)
	}

	search := hosttest.NewSearch("physics")
	vault := hosttest.New("physics")
	queue := hosttest.NewQueue()
	clk := newFakeClock(getMeasuredAt())

	for _, tc := range []struct {
		name  string
		hosts service.Hosts
	}{
		{"missing search", service.Hosts{Floors: relevance.Host{Vaults: func(id domain.VaultID) host.NoteLookup { return vault.With(id) }, Store: foldersOf(vault)}, Queue: queue}},
		{"missing vault", service.Hosts{Floors: relevance.Host{Search: searchOf(search), Store: foldersOf(vault)}, Queue: queue}},
		{"missing store", service.Hosts{Floors: relevance.Host{Search: searchOf(search), Vaults: func(id domain.VaultID) host.NoteLookup { return vault.With(id) }}, Queue: queue}},
		{"missing queue", service.Hosts{Floors: relevance.Host{Search: searchOf(search), Vaults: func(id domain.VaultID) host.NoteLookup { return vault.With(id) }, Store: foldersOf(vault)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := service.NewAppOver(tc.hosts, clk, nil)
			if _, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, app.Mount); err == nil {
				t.Error("Mount succeeded with incomplete host")
			}
		})
	}

	errReport := errors.New("report task failed")
	errFinish := errors.New("finish task failed")
	search.SetModel(hosttest.SearchModel{Name: "test-model", Recipe: modelRecipe, FloorHint: 0.4})
	search.SetEmbedder(func(string) []float32 { return []float32{1, 0, 0, 0} })
	search.SetSampledChunks(modelRecipe, sampledChunks(60, 3))
	search.SetProgress(60, 60)
	setNotes(vault, 25)

	appReportFail := service.NewAppOver(service.Hosts{
		Floors: relevance.Host{Search: searchOf(search), Vaults: func(id domain.VaultID) host.NoteLookup { return vault.With(id) }, Store: foldersOf(vault)},
		Queue:  failingQueue{Queue: queue, reportErr: errReport},
	}, clk, nil)
	if _, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, appReportFail.Mount); err != nil {
		t.Fatal(err)
	}
	search.FailEmbedding(errors.New("embed fail"))
	err := appReportFail.HandleEvent(t.Context(), getReadyEvent("physics"))
	if !errors.Is(err, host.ErrEmbeddingFailed) || !errors.Is(err, errReport) {
		t.Errorf("HandleEvent with failing ReportTask gave %v, want both errors", err)
	}

	search.FailEmbedding(nil)
	appFinishFail := service.NewAppOver(service.Hosts{
		Floors: relevance.Host{Search: searchOf(search), Vaults: func(id domain.VaultID) host.NoteLookup { return vault.With(id) }, Store: foldersOf(vault)},
		Queue:  failingQueue{Queue: queue, finishErr: errFinish},
	}, clk, nil)
	if _, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, appFinishFail.Mount); err != nil {
		t.Fatal(err)
	}
	if err := appFinishFail.HandleEvent(t.Context(), getReadyEvent("physics")); !errors.Is(err, errFinish) {
		t.Errorf("HandleEvent with failing FinishTask gave %v, want errFinish", err)
	}
}

func TestAnEventBeforeTheAppIsMountedIsIgnored(t *testing.T) {
	app := service.NewApp()
	if err := app.HandleEvent(t.Context(), getReadyEvent("physics")); err != nil {
		t.Errorf("got %v, want the event ignored", err)
	}
}

func searchOf(search *hosttest.Search) func(domain.VaultID) relevance.Search {
	return func(id domain.VaultID) relevance.Search {
		on := search.With(id)
		return relevance.Search{Index: on, Policy: on}
	}
}

func foldersOf(vault *hosttest.Vault) func(domain.VaultID) relevance.Records {
	return func(id domain.VaultID) relevance.Records { return floorrecords.New(vault.Store().With(id)) }
}

func getReadyEvent(vault string) string {
	return "onSearchSourceReady:" + vault
}

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}

func TestAnEventReleasesTheProcessOnceItIsHandled(t *testing.T) {
	f := newFixture(t, 0.4)

	if err := f.app.HandleEvent(t.Context(), getReadyEvent("physics")); err != nil {
		t.Fatal(err)
	}

	if releases := f.processes.CountReleases(); releases != 1 {
		t.Errorf("releases = %d, want 1", releases)
	}
}

func TestAnEventThatFailsReleasesTheProcessToo(t *testing.T) {
	f := newFixture(t, 0.4)
	f.search.FailEmbedding(errors.New("model stopped"))

	if err := f.app.HandleEvent(t.Context(), getReadyEvent("physics")); err == nil {
		t.Fatal("the measurement succeeded without embeddings")
	}

	if releases := f.processes.CountReleases(); releases != 1 {
		t.Errorf("releases = %d, want 1", releases)
	}
}

func TestACallOfTheSettingsViewKeepsTheProcess(t *testing.T) {
	f := newFixture(t, 0.4)
	var state api.GetFloorResponse

	if status, failure := f.call(t, api.MethodGetFloor, api.GetFloorRequest{}, &state); status != http.StatusOK {
		t.Fatalf("GetFloor = %d %+v", status, failure)
	}

	if releases := f.processes.CountReleases(); releases != 0 {
		t.Errorf("releases = %d, want none", releases)
	}
}
