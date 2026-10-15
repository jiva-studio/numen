package plugin_test

import (
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
)

func TestRealPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	ocrBin := filepath.Join(pluginsDir, "ocr", "bin", "ocr")
	if _, err := os.Stat(ocrBin); os.IsNotExist(err) {
		t.Skipf("OCR binary not found at %s; skipping real process E2E test", ocrBin)
	}

	host := plugin.NewHost("desktop.editor", pluginsDir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugins := host.GetPlugins()
	if len(plugins) == 0 {
		t.Fatal("expected to discover OCR plugin in real modules/plugins")
	}

	ctx := t.Context()
	bridge, err := plugin.OpenRecogniserBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenRecogniserBridge with real binary failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	model := bridge.GetRecognitionModel()
	if model.Layout == "" || model.Recogniser == "" {
		t.Fatalf("expected valid model from real OCR plugin, got %+v", model)
	}

	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	blocks, err := bridge.Recognise(ctx, img)
	if err != nil {
		t.Fatalf("Recognise with real plugin failed: %v", err)
	}
	if len(blocks) == 0 {
		t.Fatal("expected at least 1 recognised block from real plugin")
	}
	if blocks[0].Text == "" {
		t.Fatal("expected non-empty text in recognised block")
	}
}

func TestRealProofreadingPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	proofreadingBin := filepath.Join(pluginsDir, "proofreading", "bin", "proofreading")
	if _, err := os.Stat(proofreadingBin); os.IsNotExist(err) {
		t.Skipf("Proofreading binary not found at %s; skipping real process E2E test", proofreadingBin)
	}

	host := plugin.NewHost("desktop.editor", pluginsDir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx := t.Context()
	bridge, err := plugin.OpenProofreaderBridge(host)
	if err != nil {
		t.Fatalf("OpenProofreaderBridge failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	batches := []domain.ProofreadBatch{
		{Number: 1, Prompt: "This is test sentence."},
	}
	replies, err := bridge.Proofread(ctx, batches)
	if err != nil {
		t.Fatalf("Proofread with real plugin failed: %v", err)
	}
	if len(replies) == 0 {
		t.Fatal("expected at least 1 reply from real proofreading plugin")
	}
	if replies[1] == "" {
		t.Fatal("expected non-empty reply for batch 1")
	}
}

func TestRealTranscriptionPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	transBin := filepath.Join(pluginsDir, "transcription", "bin", "transcription")
	if _, err := os.Stat(transBin); os.IsNotExist(err) {
		t.Skipf("transcription binary not found at %s; skipping real process E2E test", transBin)
	}

	host := plugin.NewHost("desktop.editor", pluginsDir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugins := host.GetPlugins()
	if len(plugins) == 0 {
		t.Fatal("expected to discover transcription plugin in real modules/plugins")
	}

	ctx := t.Context()
	bridge, err := plugin.OpenTranscriberBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenTranscriberBridge with real binary failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	model := bridge.GetTranscriptionModel()
	if model.Model == "" || model.Segmenter == "" {
		t.Fatalf("expected valid model from real transcription plugin, got %+v", model)
	}

	rec, err := bridge.Open(ctx, []byte("test-audio-recording"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer rec.Close()

	if rec.GetDuration() <= 0 {
		t.Fatalf("expected positive duration, got %d", rec.GetDuration())
	}

	segments, err := rec.GetSegments(ctx, 0, 5)
	if err != nil {
		t.Fatalf("GetSegments failed: %v", err)
	}
	if len(segments) == 0 {
		t.Fatal("expected at least 1 audio segment")
	}

	text, err := bridge.Transcribe(ctx, segments[0])
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text == "" {
		t.Fatal("expected non-empty transcribed text")
	}
}

func TestRealBookPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	bookBin := filepath.Join(pluginsDir, "reader-epub", "bin", "reader-epub")
	if _, err := os.Stat(bookBin); os.IsNotExist(err) {
		t.Skipf("reader-epub binary not found at %s; skipping real process E2E test", bookBin)
	}

	host := plugin.NewHost("desktop.editor", pluginsDir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx := t.Context()
	bridge, err := plugin.OpenBookBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenBookBridge with real binary failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	// Query non-existent book - expect error from plugin service
	req := connect.NewRequest(&v1.GetBookRequest{Path: "nonexistent.epub"})
	_, err = bridge.GetBook(ctx, req)
	if err == nil {
		t.Fatal("expected error querying nonexistent book from real plugin")
	}
}

func TestRealEmbeddingPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	embeddingsBin := filepath.Join(pluginsDir, "embeddings", "bin", "embeddings")
	if _, err := os.Stat(embeddingsBin); os.IsNotExist(err) {
		t.Skipf("Embeddings binary not found at %s; skipping real process E2E test", embeddingsBin)
	}

	host := plugin.NewHost("desktop.editor", pluginsDir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugins := host.GetPlugins()
	if len(plugins) == 0 {
		t.Fatal("expected to discover embedding plugin in real modules/plugins")
	}

	ctx := t.Context()
	bridge, err := plugin.OpenEmbedderBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenEmbedderBridge with real binary failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	model := bridge.GetModel()
	if model.Name == "" || model.Dimensions <= 0 {
		t.Fatalf("expected valid model from real embedding plugin, got %+v", model)
	}

	vectors, err := bridge.Embed(ctx, []string{"sample search query", "another sentence"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if len(vectors[0]) != model.Dimensions {
		t.Fatalf("expected vector length %d, got %d", model.Dimensions, len(vectors[0]))
	}
}
