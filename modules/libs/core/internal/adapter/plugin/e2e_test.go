package plugin_test

import (
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

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
