package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"connectrpc.com/connect"
	read "github.com/getcharzp/go-ocr"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/engine"
)

func createTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return buf.Bytes()
}

func TestLifecyclePing(t *testing.T) {
	server := &lifecycleServer{shutdownTrigger: make(chan struct{})}
	resp, err := server.Ping(context.Background(), connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if resp.Msg.GetStatus() != "ok" {
		t.Errorf("status = %q, want %q", resp.Msg.GetStatus(), "ok")
	}
}

func TestLifecycleShutdown(t *testing.T) {
	server := &lifecycleServer{shutdownTrigger: make(chan struct{})}
	resp, err := server.Shutdown(context.Background(), connect.NewRequest(&numenv1.ShutdownRequest{}))
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("shutdown success = false, want true")
	}
}

func TestLifecycleConfigure(t *testing.T) {
	cfg := engine.Defaults()
	recog := newRecognitionServer(nil, cfg)
	server := &lifecycleServer{
		shutdownTrigger: make(chan struct{}),
		recog:           recog,
	}

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		PluginId: "md.numen.ocr",
		Values: map[string]string{
			"dpi": "450",
		},
	})
	resp, err := server.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("configure success = false, want true")
	}
	if recog.cfg.Recognise.DPI != 450 {
		t.Errorf("expected DPI 450, got %d", recog.cfg.Recognise.DPI)
	}
}

func TestGetRecognitionModelDefault(t *testing.T) {
	cfg := engine.Defaults()
	cfg.Layout.Name = "custom-layout.onnx"
	cfg.Recognise.Name = "custom-rec.onnx"
	server := newRecognitionServer(nil, cfg)

	resp, err := server.GetRecognitionModel(context.Background(), connect.NewRequest(&numenv1.GetRecognitionModelRequest{}))
	if err != nil {
		t.Fatalf("get recognition model: %v", err)
	}
	if resp.Msg.GetLayout() == "" || resp.Msg.GetRecogniser() == "" {
		t.Errorf("model layout/recogniser should not be empty: %+v", resp.Msg)
	}
}

func TestRecognisePageEmptyImageData(t *testing.T) {
	server := newRecognitionServer(nil, engine.Defaults())
	req := connect.NewRequest(&numenv1.RecognisePageRequest{ImageData: nil})
	resp, err := server.RecognisePage(context.Background(), req)
	if err != nil {
		t.Fatalf("recognise empty page: %v", err)
	}
	if len(resp.Msg.GetBlocks()) != 0 {
		t.Errorf("got %d blocks, want 0", len(resp.Msg.GetBlocks()))
	}
}

func TestRecognisePageInvalidImageData(t *testing.T) {
	server := newRecognitionServer(nil, engine.Defaults())
	// Inject a non-nil engine so it passes engine initialization
	testEng := engine.NewTestRecogniser(
		func(image.Image) ([]engine.Region, error) { return nil, nil },
		func(image.Image) ([]read.RecResult, error) { return nil, nil },
		engine.Model{Layout: "test-layout", Recogniser: "test-rec", DPI: 300, From: "test"},
	)
	server.engine = testEng

	req := connect.NewRequest(&numenv1.RecognisePageRequest{ImageData: []byte("not-an-image")})
	_, err := server.RecognisePage(context.Background(), req)
	if err == nil {
		t.Fatal("expected error on invalid image data, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want %v", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}

func TestRecognisePageWithEngine(t *testing.T) {
	mockLayout := func(img image.Image) ([]engine.Region, error) {
		return []engine.Region{
			{
				Label: "paragraph",
				Score: 0.98,
				Rect:  image.Rect(20, 30, 200, 100),
				Order: 0,
			},
		}, nil
	}

	mockRead := func(img image.Image) ([]read.RecResult, error) {
		return []read.RecResult{
			{
				Text:  "Restored PaddleOCR Engine",
				Score: 0.99,
				Box:   [4]int{5, 5, 150, 35},
			},
		}, nil
	}

	modelInfo := engine.Model{
		Layout:     "PP-DocLayoutV3",
		Recogniser: "PP-OCRv6_rec_tiny",
		DPI:        300,
		From:       "test-builtin",
	}

	testEng := engine.NewTestRecogniser(mockLayout, mockRead, modelInfo)
	server := newRecognitionServer(testEng, engine.Defaults())

	// Test GetRecognitionModel with loaded engine
	modelResp, err := server.GetRecognitionModel(context.Background(), connect.NewRequest(&numenv1.GetRecognitionModelRequest{}))
	if err != nil {
		t.Fatalf("GetRecognitionModel failed: %v", err)
	}
	if modelResp.Msg.GetLayout() != "PP-DocLayoutV3" {
		t.Errorf("layout = %q, want %q", modelResp.Msg.GetLayout(), "PP-DocLayoutV3")
	}
	if modelResp.Msg.GetRecogniser() != "PP-OCRv6_rec_tiny" {
		t.Errorf("recogniser = %q, want %q", modelResp.Msg.GetRecogniser(), "PP-OCRv6_rec_tiny")
	}

	// Test RecognisePage
	pngBytes := createTestPNG(t, 400, 400)
	req := connect.NewRequest(&numenv1.RecognisePageRequest{ImageData: pngBytes})
	resp, err := server.RecognisePage(context.Background(), req)
	if err != nil {
		t.Fatalf("RecognisePage failed: %v", err)
	}

	blocks := resp.Msg.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}

	block := blocks[0]
	if block.GetLabel() != "paragraph" {
		t.Errorf("label = %q, want %q", block.GetLabel(), "paragraph")
	}
	if block.GetText() != "Restored PaddleOCR Engine" {
		t.Errorf("text = %q, want %q", block.GetText(), "Restored PaddleOCR Engine")
	}
	if len(block.GetBoxes()) != 1 {
		t.Fatalf("got %d boxes, want 1", len(block.GetBoxes()))
	}

	box := block.GetBoxes()[0]
	if box.GetSpan() == nil {
		t.Fatal("box span is nil")
	}
	if box.GetSpan().GetFrom() != 0 || box.GetSpan().GetTo() != int32(len("Restored PaddleOCR Engine")) {
		t.Errorf("span = [%d, %d], want [0, %d]", box.GetSpan().GetFrom(), box.GetSpan().GetTo(), len("Restored PaddleOCR Engine"))
	}
	if box.GetMinX() <= 0 || box.GetMinY() <= 0 {
		t.Errorf("box coordinates = (%d, %d, %d, %d), expected positive coordinates",
			box.GetMinX(), box.GetMinY(), box.GetMaxX(), box.GetMaxY())
	}
}
