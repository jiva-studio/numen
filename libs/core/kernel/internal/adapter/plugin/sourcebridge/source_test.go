package sourcebridge_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// The bridge is the search source the core works through, and makes the vectors of queries for calibration.
var (
	_ searchsourceport.WatchedSource = (*sourcebridge.Source)(nil)
	_ searchsourceport.QueryEmbedder = (*sourcebridge.Source)(nil)
)

// noConnection is the sentence a plugin answers while its settings name no connection.
const noConnection = "choose a connection in the settings of Vector Embeddings"

type mockSourceServer struct {
	numenv1connect.UnimplementedSearchSourceServiceHandler

	describeCalls    atomic.Int32
	modelName        atomic.Value
	width            atomic.Int32
	floorHint        atomic.Value
	minText          atomic.Int32
	chunkTokens      atomic.Int32
	recipe           atomic.Value
	failRead         atomic.Bool
	failPrecondition atomic.Bool

	mu      sync.Mutex
	asked   []string
	places  []*v1.SearchPlace
	queries []*v1.SearchSourceServiceSearchPassagesRequest
}

func (s *mockSourceServer) DescribeSearchSource(
	_ context.Context,
	_ *connect.Request[v1.DescribeSearchSourceRequest],
) (*connect.Response[v1.DescribeSearchSourceResponse], error) {
	s.describeCalls.Add(1)
	if s.failPrecondition.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(noConnection))
	}
	if s.failRead.Load() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("the model is not available"))
	}
	name := "test-model"
	if v, ok := s.modelName.Load().(string); ok && v != "" {
		name = v
	}
	width := int32(384)
	if d := s.width.Load(); d > 0 {
		width = d
	}
	hint := 0.0
	if v, ok := s.floorHint.Load().(float64); ok {
		hint = v
	}
	recipe := "test-provider|" + name
	if v, ok := s.recipe.Load().(string); ok && v != "" {
		recipe = v
	}
	return connect.NewResponse(&v1.DescribeSearchSourceResponse{
		Name:          name,
		Recipe:        recipe,
		Width:         width,
		Concurrency:   4,
		FloorHint:     hint,
		MinTextLength: s.minText.Load(),
		ChunkTokens:   s.chunkTokens.Load(),
	}), nil
}

func (s *mockSourceServer) IndexTexts(
	_ context.Context,
	req *connect.Request[v1.IndexTextsRequest],
) (*connect.Response[v1.IndexTextsResponse], error) {
	texts := req.Msg.GetTexts()
	s.mu.Lock()
	s.asked = append(s.asked, "index")
	s.places = append(s.places, req.Msg.GetPlace())
	s.mu.Unlock()
	vectors := make([]*v1.FloatVector, len(texts))
	for i := range texts {
		vectors[i] = &v1.FloatVector{Values: []float32{float32(i) * 0.1, 0.5}}
	}
	return connect.NewResponse(&v1.IndexTextsResponse{Vectors: vectors}), nil
}

func (s *mockSourceServer) EmbedQueries(
	_ context.Context,
	req *connect.Request[v1.EmbedQueriesRequest],
) (*connect.Response[v1.EmbedQueriesResponse], error) {
	s.mu.Lock()
	s.asked = append(s.asked, "queries")
	s.mu.Unlock()
	vectors := make([]*v1.FloatVector, len(req.Msg.GetQueries()))
	for i := range vectors {
		vectors[i] = &v1.FloatVector{Values: []float32{1, 0}}
	}
	return connect.NewResponse(&v1.EmbedQueriesResponse{Vectors: vectors}), nil
}

func (s *mockSourceServer) SearchPassages(
	_ context.Context,
	req *connect.Request[v1.SearchSourceServiceSearchPassagesRequest],
) (*connect.Response[v1.SearchSourceServiceSearchPassagesResponse], error) {
	s.mu.Lock()
	s.queries = append(s.queries, req.Msg)
	s.mu.Unlock()
	return connect.NewResponse(&v1.SearchSourceServiceSearchPassagesResponse{Matches: []*v1.ChunkMatch{
		{ChunkId: "41", Score: 0.9}, {ChunkId: "7", Score: 0.6},
	}}), nil
}

func (s *mockSourceServer) DescribeCoverage(
	_ context.Context,
	req *connect.Request[v1.DescribeCoverageRequest],
) (*connect.Response[v1.DescribeCoverageResponse], error) {
	return connect.NewResponse(&v1.DescribeCoverageResponse{
		Text: fmt.Sprintf("%d of %d", req.Msg.GetIndexedCount(), req.Msg.GetChunkCount()),
	}), nil
}

func TestOpenSourceBridgeNoPlugin(t *testing.T) {
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", t.TempDir())
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	bridge, err := sourcebridge.Open(t.Context(), pluginhost.Join(host))
	if err == nil {
		t.Fatal("expected error when no search source plugin found, got nil")
	}
	if bridge != nil {
		t.Fatalf("expected nil bridge, got %v", bridge)
	}
}

func startSource(t *testing.T, mockServer *mockSourceServer) *container.PluginRoles {
	t.Helper()
	mux := http.NewServeMux()
	wiretest.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewSearchSourceServiceHandler(mockServer))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	serverPort := wiretest.GetPort(t, server)

	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	manifestText := `{
		"id": "md.numen.embeddings",
		"name": "Vector Embeddings",
		"version": "1.0.0",
		"capabilities": ["search.source"],
		"targets": ["desktop.editor"],
		"services": ["numen.v1.SearchSourceService"], "backend": {"entrypoint": "bin/embeddings"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifestText), 0o600); err != nil {
		t.Fatal(err)
	}
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	cmd := wiretest.StartSleepProcess(t)
	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})
	return host
}

func TestTheBridgeReadsTheDescriptionOfTheSource(t *testing.T) {
	mockServer := &mockSourceServer{}
	mockServer.floorHint.Store(0.35)
	mockServer.minText.Store(40)
	mockServer.recipe.Store("test-provider|test-model|384|1a2b3c4d")
	ctx := t.Context()
	bridge, err := sourcebridge.Open(ctx, pluginhost.Join(startSource(t, mockServer)))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer testsupport.Close(t, bridge)

	described := bridge.Describe()
	want := searchsource.Description{Name: "test-model", SourceRecipe: "test-provider|test-model|384|1a2b3c4d", Width: 384, MinTextLength: 40, Concurrency: 4}
	if described != want {
		t.Errorf("description %+v, want %+v", described, want)
	}
	if hint := bridge.GetFloorHint(ctx); hint != 0.35 {
		t.Errorf("floor hint = %v, want 0.35", hint)
	}
}

func TestADescriptionIsAnsweredWithoutAskingThePluginAgain(t *testing.T) {
	mockServer := &mockSourceServer{}
	mockServer.recipe.Store("test-provider|test-model|384|1a2b3c4d")
	bridge, err := sourcebridge.Open(t.Context(), pluginhost.Join(startSource(t, mockServer)))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer testsupport.Close(t, bridge)
	reads := mockServer.describeCalls.Load()

	mockServer.recipe.Store("test-provider|test-model|384|ffffffff")
	described := bridge.Describe()

	if got := mockServer.describeCalls.Load() - reads; got != 0 {
		t.Errorf("Describe asked the plugin %d times, want none", got)
	}
	if described.SourceRecipe != "test-provider|test-model|384|1a2b3c4d" {
		t.Errorf("recipe = %q, want the one read on open", described.SourceRecipe)
	}
}

func TestANegativeFloorHintIsReadAsNone(t *testing.T) {
	mockServer := &mockSourceServer{}
	mockServer.floorHint.Store(-1.0)
	bridge, err := sourcebridge.Open(t.Context(), pluginhost.Join(startSource(t, mockServer)))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer testsupport.Close(t, bridge)

	if hint := bridge.GetFloorHint(t.Context()); hint != 0 {
		t.Errorf("a negative floor hint reads as %v, want 0", hint)
	}
}

func TestTheBridgeSendsTheTextsAndTheQueriesToTheSource(t *testing.T) {
	mockServer := &mockSourceServer{}
	ctx := t.Context()
	bridge, err := sourcebridge.Open(ctx, pluginhost.Join(startSource(t, mockServer)))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer testsupport.Close(t, bridge)
	place := searchsource.Place{VaultID: "01PHYSICS", Recipe: bridge.Describe().Recipe()}

	vectors, err := bridge.IndexTexts(ctx, place, []searchsource.Text{{ChunkID: "1", Text: "first text"}, {ChunkID: "2", Path: "book.pdf", VersionID: "v1", Text: "second text"}})
	if err != nil {
		t.Fatalf("IndexTexts failed: %v", err)
	}
	if len(vectors) != 2 || vectors[0][0] != 0.0 || vectors[1][0] != 0.1 {
		t.Errorf("vectors %v", vectors)
	}
	if empty, err := bridge.IndexTexts(ctx, place, nil); err != nil || empty != nil {
		t.Errorf("expected nil vectors for empty input, got %v, err=%v", empty, err)
	}
	if _, err := bridge.EmbedQueries(ctx, []string{"a question"}); err != nil {
		t.Fatalf("EmbedQueries failed: %v", err)
	}
	matches, err := bridge.SearchPassages(ctx, place, searchsource.Query{Text: "entropy", Limit: 5, Floor: 0.4, PathPrefix: "books/"})
	if err != nil {
		t.Fatalf("SearchPassages failed: %v", err)
	}
	if want := []searchsource.Match{{ChunkID: "41", Score: 0.9}, {ChunkID: "7", Score: 0.6}}; !slices.Equal(matches, want) {
		t.Errorf("matches %v, want %v", matches, want)
	}
	coverage, err := bridge.DescribeCoverage(ctx, place, searchsource.Counts{Chunks: 12, Indexed: 3})
	if err != nil || coverage != "3 of 12" {
		t.Errorf("coverage %q, %v", coverage, err)
	}

	mockServer.mu.Lock()
	defer mockServer.mu.Unlock()
	if want := []string{"index", "queries"}; !slices.Equal(mockServer.asked, want) {
		t.Errorf("the plugin was asked %v, want %v", mockServer.asked, want)
	}
	if got := mockServer.places[0]; got.GetVaultId() != "01PHYSICS" || got.GetRecipe() != place.Recipe {
		t.Errorf("the texts were handed with the place %v", got)
	}
	if q := mockServer.queries[0]; q.GetQuery() != "entropy" || q.GetLimit() != 5 || q.GetFloor() != 0.4 || q.GetPathPrefix() != "books/" {
		t.Errorf("the plugin was asked %v", q)
	}
}

// A source whose settings name no connection fails to open with the sentence the plugin gives, not with a code.
func TestASourceTheSettingsDoNotAllowFailsToOpenWithThePlainSentence(t *testing.T) {
	mockServer := &mockSourceServer{}
	mockServer.failPrecondition.Store(true)

	_, err := sourcebridge.Open(t.Context(), pluginhost.Join(startSource(t, mockServer)))

	if err == nil || err.Error() != noConnection {
		t.Errorf("err = %v, want %q", err, noConnection)
	}
}
