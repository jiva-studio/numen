package integration_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/downloaderbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/proofreaderbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transcriberbridge"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
)

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
	bridge, err := proofreaderbridge.Open(host)
	if err != nil {
		t.Fatalf("OpenProofreaderBridge failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	// The host has no model routed for proofreading, so the plugin has nothing to
	// ask and invents no text.
	lines := []domain.ProofreadLine{{Number: 0, Text: "This is test sentence."}}
	res, err := bridge.ProofreadSpeech(ctx, lines, 0)
	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Fatalf("ProofreadSpeech with real plugin: got %v, want %v", err, completionport.ErrUnavailable)
	}
	if len(res.Corrections) != 0 {
		t.Fatalf("expected no corrections, got %v", res.Corrections)
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
	bridge, err := transcriberbridge.Open(ctx, host)
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

	rawWav := encodeTestWav(16000, 1, make([]int16, 16000))
	rec, err := bridge.Open(ctx, rawWav)
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
	for _, seg := range segments {
		_, err := bridge.Transcribe(ctx, seg)
		if err != nil {
			t.Fatalf("Transcribe failed: %v", err)
		}
	}
}

func encodeTestWav(rate, channels int, samples []int16) []byte {
	body := new(bytes.Buffer)
	for _, one := range samples {
		_ = binary.Write(body, binary.LittleEndian, one)
	}

	out := new(bytes.Buffer)
	out.WriteString("RIFF")
	_ = binary.Write(out, binary.LittleEndian, uint32(36+body.Len()))
	out.WriteString("WAVEfmt ")
	_ = binary.Write(out, binary.LittleEndian, uint32(16))
	_ = binary.Write(out, binary.LittleEndian, uint16(1))
	_ = binary.Write(out, binary.LittleEndian, uint16(channels))
	_ = binary.Write(out, binary.LittleEndian, uint32(rate))
	_ = binary.Write(out, binary.LittleEndian, uint32(rate*channels*2))
	_ = binary.Write(out, binary.LittleEndian, uint16(channels*2))
	_ = binary.Write(out, binary.LittleEndian, uint16(16))
	out.WriteString("data")
	_ = binary.Write(out, binary.LittleEndian, uint32(body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
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
	bridge, err := embedderbridge.Open(ctx, host)
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

func TestRealDownloaderPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	downloaderBin := filepath.Join(pluginsDir, "downloader", "bin", "downloader")
	if _, err := os.Stat(downloaderBin); os.IsNotExist(err) {
		t.Skipf("downloader binary not found at %s; skipping real process E2E test", downloaderBin)
	}

	host := plugin.NewHost("desktop.editor", pluginsDir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx := t.Context()
	bridge, err := downloaderbridge.Open(ctx, host)
	if err != nil {
		t.Fatalf("OpenDownloaderBridge with real binary failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	model, err := bridge.GetDownloadModel(ctx, domain.URL("https://example.com/test"))
	if err != nil {
		t.Fatalf("GetDownloadModel failed: %v", err)
	}
	if model.Producer == "" {
		t.Fatalf("expected valid producer from real downloader plugin, got %+v", model)
	}

	can, err := bridge.CanHandle(ctx, domain.URL("https://example.com/test"))
	if err != nil {
		t.Fatalf("CanHandle failed: %v", err)
	}
	if !can {
		t.Fatalf("expected CanHandle to be true for https://example.com/test")
	}
}
