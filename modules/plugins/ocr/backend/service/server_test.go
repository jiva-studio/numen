package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"connectrpc.com/connect"
	read "github.com/getcharzp/go-ocr"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
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
	life := server.NewLifecycleServer(make(chan struct{}))
	resp, err := life.Ping(context.Background(), connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if resp.Msg.GetStatus() != "ok" {
		t.Errorf("status = %q, want %q", resp.Msg.GetStatus(), "ok")
	}
}

func TestLifecycleShutdown(t *testing.T) {
	life := server.NewLifecycleServer(make(chan struct{}))
	resp, err := life.Shutdown(context.Background(), connect.NewRequest(&numenv1.ShutdownRequest{}))
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
	life := server.NewLifecycleServer(
		make(chan struct{}),
		server.WithLifecycleUpdateConfig(recog.updateConfig),
	)

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		PluginId: "md.numen.ocr",
		Values: map[string]string{
			"dpi": "450",
		},
	})
	resp, err := life.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("configure success = false, want true")
	}
	if recog.cfg.Recognise.DPI != 450 {
		t.Errorf("expected DPI 450, got %d", recog.cfg.Recognise.DPI)
	}

	for _, invalid := range []string{"-100", "0", "not-a-number"} {
		reqInvalid := connect.NewRequest(&numenv1.ConfigureRequest{
			PluginId: "md.numen.ocr",
			Values:   map[string]string{"dpi": invalid},
		})
		if _, err := life.Configure(context.Background(), reqInvalid); err != nil {
			t.Fatalf("configure invalid failed: %v", err)
		}
		if recog.cfg.Recognise.DPI != 450 {
			t.Errorf("DPI should remain 450 on invalid value %q, got %d", invalid, recog.cfg.Recognise.DPI)
		}
	}

	reqOne := connect.NewRequest(&numenv1.ConfigureRequest{
		PluginId: "md.numen.ocr",
		Values:   map[string]string{"dpi": "1"},
	})
	if _, err := life.Configure(context.Background(), reqOne); err != nil {
		t.Fatalf("configure dpi 1 failed: %v", err)
	}
	if recog.cfg.Recognise.DPI != 1 {
		t.Errorf("expected DPI 1, got %d", recog.cfg.Recognise.DPI)
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
	if resp.Msg.GetDpi() != 300 {
		t.Errorf("dpi = %d, want 300", resp.Msg.GetDpi())
	}
	if resp.Msg.GetFrom() != "defaults" {
		t.Errorf("from = %q, want %q", resp.Msg.GetFrom(), "defaults")
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
	if !errors.Is(err, image.ErrFormat) {
		t.Errorf("expected wrapped image.ErrFormat, got %v", err)
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

func TestRecognisePageOpenEngineError(t *testing.T) {
	server := newRecognitionServer(nil, engine.Config{})
	pngBytes := createTestPNG(t, 10, 10)
	req := connect.NewRequest(&numenv1.RecognisePageRequest{ImageData: pngBytes})
	_, err := server.RecognisePage(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want %v", connect.CodeOf(err), connect.CodeInternal)
	}
	if errors.Unwrap(errors.Unwrap(err)) == nil {
		t.Errorf("expected wrapped inner error, got %v", err)
	}
}

func TestRecognisePageRecogniseError(t *testing.T) {
	broken := errors.New("recognition failed")
	testEng := engine.NewTestRecogniser(
		func(image.Image) ([]engine.Region, error) {
			return nil, broken
		},
		func(image.Image) ([]read.RecResult, error) { return nil, nil },
		engine.Model{},
	)
	server := newRecognitionServer(testEng, engine.Defaults())
	pngBytes := createTestPNG(t, 10, 10)
	req := connect.NewRequest(&numenv1.RecognisePageRequest{ImageData: pngBytes})
	_, err := server.RecognisePage(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want %v", connect.CodeOf(err), connect.CodeInternal)
	}
	if !errors.Is(err, broken) {
		t.Errorf("expected wrapped error %v, got %v", broken, err)
	}
}
