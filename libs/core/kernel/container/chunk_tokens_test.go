package container_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const chunkEmbeddingPluginID = "md.numen.embeddings"

// chunkingPlugin is an embedding plugin that reports the chunk size in tokens it is given.
type chunkingPlugin struct {
	wiretest.PluginLifecycle
	tokens atomic.Int32
}

func (*chunkingPlugin) Configure(
	_ context.Context, _ *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

func (p *chunkingPlugin) GetEmbeddingModel(
	context.Context, *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{
		Name: "test-model", Dimensions: 8, MaxTokens: 512, Pooling: "mean", From: "test", Concurrency: 1,
		ChunkTokens: p.tokens.Load(),
	}), nil
}

func (*chunkingPlugin) EmbedBatch(
	_ context.Context, req *connect.Request[v1.EmbedBatchRequest],
) (*connect.Response[v1.EmbedBatchResponse], error) {
	vectors := make([]*v1.FloatVector, len(req.Msg.GetTexts()))
	for i := range vectors {
		vectors[i] = &v1.FloatVector{Values: []float32{1, 0, 0, 0, 0, 0, 0, 0}}
	}
	return connect.NewResponse(&v1.EmbedBatchResponse{Vectors: vectors}), nil
}

func openChunkingConfig(t *testing.T, tokens int32) (container.Config, *plugin.Host, *chunkingPlugin) {
	t.Helper()
	reporter := &chunkingPlugin{}
	reporter.tokens.Store(tokens)
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(reporter))
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(reporter))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	port := wiretest.GetPort(t, server)

	root := t.TempDir()
	dir := filepath.Join(root, "embeddings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{"id": "` + chunkEmbeddingPluginID + `", "name": "Embeddings", "version": "1.0.0", "capabilities": ["embedding"],
		"targets": ["desktop.editor"], "backend": {"entrypoint": "bin/run"}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	cmd := wiretest.StartSleepProcess(t)
	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, wiretest.GetReadyOutput(port), nil
	})
	return container.NewConfig(container.Config{
		PluginHost: container.AdaptPluginHost(host),
		Paths:      container.Paths{ServiceDir: ".numen"},
	}), host, reporter
}

func openEmbedder(t *testing.T, cfg container.Config) {
	t.Helper()
	embedder, closeEmbedder, err := cfg.Embedder(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if embedder == nil {
		t.Fatal("no embedder was opened")
	}
	t.Cleanup(func() { testsupport.Close(t, closerOf(closeEmbedder)) })
}

type closerOf func() error

func (c closerOf) Close() error { return c() }

func TestChunkLimitIsTheCharacterBoundOfTheChunkSizeInTheCell(t *testing.T) {
	cfg := container.NewConfig(container.Config{})

	cfg.Resources.Chunks.Set(512)
	if got := cfg.CutNotes().Sizes.Limit; got != chunking.GetCharacterBound(512) {
		t.Errorf("cut at %d characters with 512 tokens, want %d", got, chunking.GetCharacterBound(512))
	}
	cfg.Resources.Chunks.Set(0)
	if got := cfg.CutNotes().Sizes.Limit; got != chunking.DefaultLimit {
		t.Errorf("cut at %d characters with no size known, want the default %d", got, chunking.DefaultLimit)
	}
}

func TestEveryCopyOfAConfigChunksAtTheSizeSetThroughOneCopy(t *testing.T) {
	cfg := container.NewConfig(container.Config{})
	copied := cfg

	cfg.Resources.Chunks.Set(300)

	if got := copied.CutNotes().Sizes.Limit; got != 600 {
		t.Errorf("a copy of the configuration cuts at %d characters, want 600", got)
	}
}

func TestNotesAreChunkedAtTheSizeTheEmbeddingPluginReports(t *testing.T) {
	cfg, _, _ := openChunkingConfig(t, 300)

	openEmbedder(t, cfg)

	if got := cfg.Resources.Chunks.Get(); got != 300 {
		t.Errorf("the cell holds %d tokens, want the 300 the plugin reports", got)
	}
	if got := cfg.CutNotes().Sizes.Limit; got != 600 {
		t.Errorf("notes are cut at %d characters, want 600", got)
	}
}

func TestAChangeOfTheChunkSizeOfThePluginReachesTheCell(t *testing.T) {
	cfg, host, reporter := openChunkingConfig(t, 256)
	openEmbedder(t, cfg)

	reporter.tokens.Store(128)
	if err := updateSetting(t.Context(), host, chunkEmbeddingPluginID, "chunkTokens", 128); err != nil {
		t.Fatal(err)
	}

	wiretest.WaitUntil(t, "the cell to hold the new chunk size", func() bool { return cfg.Resources.Chunks.Get() == 128 })
}

func TestWithNoEmbeddingPluginNotesAreChunkedAtTheDefaultLimit(t *testing.T) {
	cfg := container.NewConfig(container.Config{})

	embedder, closeEmbedder, err := cfg.Embedder(t.Context())
	if err != nil || embedder != nil {
		t.Fatalf("Embedder = %v, %v; want none", embedder, err)
	}
	if err := closeEmbedder(); err != nil {
		t.Fatal(err)
	}
	if got := cfg.CutNotes().Sizes.Limit; got != chunking.DefaultLimit {
		t.Errorf("notes are cut at %d characters, want the default %d", got, chunking.DefaultLimit)
	}
	if cfg.Resources.Chunks.Get() != 0 {
		t.Errorf("the cell holds %d tokens with no plugin", cfg.Resources.Chunks.Get())
	}
}

func TestNoChunkSizeFromThePluginKeepsTheCellAtZero(t *testing.T) {
	cfg, _, _ := openChunkingConfig(t, 0)

	openEmbedder(t, cfg)

	if got := cfg.Resources.Chunks.Get(); got != 0 {
		t.Errorf("the cell holds %d tokens, want 0", got)
	}
	if got := cfg.CutNotes().Sizes.Limit; got != chunking.DefaultLimit {
		t.Errorf("notes are cut at %d characters, want the default %d", got, chunking.DefaultLimit)
	}
}

// The file texts follow the number through the recipe they keep, which names the sizes.
func TestFileTextsAreChunkedAtTheLimitOfTheCell(t *testing.T) {
	db := makeIndex(t)
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{ServiceDir: ".numen"},
	})
	v := domain.Vault{ID: "v", Path: t.TempDir()}

	cfg.Resources.Chunks.Set(256)
	small, err := container.IndexTexts(cfg, db, v)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Resources.Chunks.Set(512)
	large, err := container.IndexTexts(cfg, db, v)
	if err != nil {
		t.Fatal(err)
	}

	if small.Sizes.Limit != 512 || large.Sizes.Limit != 1024 {
		t.Errorf("file texts are cut at %d and %d characters, want 512 and 1024", small.Sizes.Limit, large.Sizes.Limit)
	}
}

func TestEverySiteThatChunksNotesUsesTheSizeThePluginReports(t *testing.T) {
	cfg, _, _ := openChunkingConfig(t, 300)
	openEmbedder(t, cfg)
	db := makeIndex(t)
	v := vaultfixture.NewVault(t, map[string]string{"Hedge.md": "# Hedge\n\nThe hedge along the lane is thick and tangled.\n"})
	const limit = 600

	if got := container.Scan(cfg, db).Cut.Sizes.Limit; got != limit {
		t.Errorf("Scan cuts at %d characters, want %d", got, limit)
	}
	texts, err := container.IndexTexts(cfg, db, v)
	if err != nil {
		t.Fatal(err)
	}
	if texts.Sizes.Limit != limit {
		t.Errorf("IndexTexts cuts at %d characters, want %d", texts.Sizes.Limit, limit)
	}
	opener := container.VaultOpener(cfg, db)
	if got := opener.GetScan().Cut.Sizes.Limit; got != limit {
		t.Errorf("the scan of VaultOpenerWith cuts at %d characters, want %d", got, limit)
	}
	if got := opener.GetRefresh().Cut.Sizes.Limit; got != limit {
		t.Errorf("the refresh of VaultOpenerWith cuts at %d characters, want %d", got, limit)
	}
}

// UpdateIndex and ReadWholeVault hide the cut they make, so the recipe each one records says it.
func TestTheNotesUpdateIndexAndReadWholeVaultSaveAreRecordedAtTheSizeThePluginReports(t *testing.T) {
	cfg, _, _ := openChunkingConfig(t, 300)
	openEmbedder(t, cfg)
	db := makeIndex(t)
	v := vaultfixture.NewVault(t, map[string]string{
		"Hedge.md": "# Hedge\n\nThe hedge along the lane is thick and tangled.\n",
		"Gate.md":  "# Gate\n\nThe gate of the orchard sticks in wet weather.\n",
	})
	recipe := chunking.NewCutNotes(300).Recipe()
	if recipe == "" {
		t.Fatal("a cut has no recipe")
	}
	if err := db.Vaults().Register(t.Context(), v.ID); err != nil {
		t.Fatal(err)
	}

	if err := container.UpdateIndex(cfg, db)(t.Context(), v, []string{"Hedge.md"}); err != nil {
		t.Fatal(err)
	}
	read, err := container.ReadWholeVault(cfg, t.Context(), db, unit{"model"}, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := read.Execute(t.Context(), v, indexing.Reporting{}); err != nil {
		t.Fatal(err)
	}

	other, err := db.Sources().GetSourcesWithOtherRecipe(t.Context(), v.ID, domain.KindNote, []string{recipe}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Errorf("%s were not recorded with the recipe of a cut at 300 tokens", strings.Join(other, ", "))
	}
}

func TestAScanChecksTheRecordedChunkingOnlyWhereTheSizeIsKnown(t *testing.T) {
	db := makeIndex(t)
	cfg := container.NewConfig(container.Config{})

	if scan := container.Scan(cfg, db); scan.ShouldRecut {
		t.Error("a scan with no chunk size known checks the recorded cut")
	}
	cfg.Resources.Chunks.Set(256)
	scan := container.Scan(cfg, db)
	if !scan.ShouldRecut {
		t.Error("a scan with a chunk size known does not check the recorded cut")
	}
}

func TestANewChunkSizeMakesTheNextScanReadTheNotesAgain(t *testing.T) {
	db := makeIndex(t)
	cfg := container.NewConfig(container.Config{})
	v := vaultfixture.NewVault(t, map[string]string{
		"Hedge.md":   "# Hedge\n\nThe hedge along the lane is thick and tangled.\n",
		"Gate.md":    "# Gate\n\nThe gate of the orchard sticks in wet weather.\n",
		"Compost.md": "# Compost\n\nThe compost heap sits behind the shed.\n",
	})

	cfg.Resources.Chunks.Set(256)
	first, err := container.Scan(cfg, db).Execute(t.Context(), v)
	if err != nil || first.Indexed != 3 {
		t.Fatalf("first scan: %+v, %v", first, err)
	}
	cfg.Resources.Chunks.Set(512)
	second, err := container.Scan(cfg, db).Execute(t.Context(), v)
	if err != nil || second.Indexed != 3 {
		t.Errorf("scan after the chunk size changed indexed %d notes (%v), want 3", second.Indexed, err)
	}
	third, err := container.Scan(cfg, db).Execute(t.Context(), v)
	if err != nil || third.Indexed != 0 {
		t.Errorf("scan at the same chunk size indexed %d notes (%v), want none", third.Indexed, err)
	}
}
