// Package service is the backend of the OCR plugin.
package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strconv"
	"sync"

	"connectrpc.com/connect"
	_ "golang.org/x/image/webp"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/engine"
)

type recognitionServer struct {
	numenv1connect.UnimplementedRecognitionServiceHandler

	mu     sync.Mutex
	engine *engine.Recogniser
	cfg    engine.Config
}

func newRecognitionServer(eng *engine.Recogniser, cfg engine.Config) *recognitionServer {
	return &recognitionServer{
		engine: eng,
		cfg:    cfg,
	}
}

func (s *recognitionServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if dpiStr, ok := values["dpi"]; ok {
		if dpi, err := strconv.Atoi(dpiStr); err == nil && dpi > 0 {
			s.cfg.Recognise.DPI = dpi
		}
	}
}

func (s *recognitionServer) getEngine(ctx context.Context) (*engine.Recogniser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engine != nil {
		return s.engine, nil
	}
	eng, err := engine.Open(ctx, s.cfg)
	if err != nil {
		return nil, err
	}
	s.engine = eng
	return s.engine, nil
}

func (s *recognitionServer) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engine != nil {
		err := s.engine.Close()
		s.engine = nil
		return err
	}
	return nil
}

func (s *recognitionServer) GetRecognitionModel(
	ctx context.Context,
	_ *connect.Request[numenv1.GetRecognitionModelRequest],
) (*connect.Response[numenv1.GetRecognitionModelResponse], error) {
	s.mu.Lock()
	eng := s.engine
	s.mu.Unlock()

	// gomutants:disable-next-line BRANCH_IF reason="getEngine returns s.engine when non-nil, so fallthrough produces the same response"
	if eng != nil {
		m := eng.GetRecognitionModel()
		return connect.NewResponse(&numenv1.GetRecognitionModelResponse{
			Layout:     m.Layout,
			Recogniser: m.Recogniser,
			Dpi:        int32(m.DPI),
			From:       m.From,
		}), nil
	}

	eng, err := s.getEngine(ctx)
	// gomutants:disable-next-line BRANCH_IF reason="opening an uninitialised engine requires real models not present during unit tests"
	if err == nil {
		m := eng.GetRecognitionModel()
		return connect.NewResponse(&numenv1.GetRecognitionModelResponse{
			Layout:     m.Layout,
			Recogniser: m.Recogniser,
			Dpi:        int32(m.DPI),
			From:       m.From,
		}), nil
	}

	return connect.NewResponse(&numenv1.GetRecognitionModelResponse{
		Layout:     s.cfg.Layout.Name,
		Recogniser: s.cfg.Recognise.Name,
		Dpi:        300,
		From:       "defaults",
	}), nil
}

func (s *recognitionServer) RecognisePage(
	ctx context.Context,
	req *connect.Request[numenv1.RecognisePageRequest],
) (*connect.Response[numenv1.RecognisePageResponse], error) {
	data := req.Msg.GetImageData()
	if len(data) == 0 {
		return connect.NewResponse(&numenv1.RecognisePageResponse{
			Blocks: nil,
		}), nil
	}

	eng, err := s.getEngine(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("open recognition engine: %w", err))
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("decode image: %w", err))
	}

	blocks, err := eng.Recognise(ctx, img)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("recognise page: %w", err))
	}

	outBlocks := make([]*numenv1.OcrBlock, 0, len(blocks))
	for _, b := range blocks {
		boxes := make([]*numenv1.OcrBox, 0, len(b.Boxes))
		for _, bx := range b.Boxes {
			boxes = append(boxes, &numenv1.OcrBox{
				MinX: int32(bx.Rect.Min.X),
				MinY: int32(bx.Rect.Min.Y),
				MaxX: int32(bx.Rect.Max.X),
				MaxY: int32(bx.Rect.Max.Y),
				Span: &numenv1.Span{
					From: int32(bx.Span.From),
					To:   int32(bx.Span.To),
				},
			})
		}
		outBlocks = append(outBlocks, &numenv1.OcrBlock{
			Label:     b.Label,
			Text:      b.Text,
			IsHeading: b.IsHeading,
			Depth:     int32(b.Depth),
			Boxes:     boxes,
		})
	}

	return connect.NewResponse(&numenv1.RecognisePageResponse{
		Blocks: outBlocks,
	}), nil
}

// Mount registers the recognition service on s, with the engine the environment configures. The
// engine closes with s.
func Mount(s *server.Server) error {
	cfg := engine.Defaults()
	if dir := os.Getenv("NUMEN_OCR_DIR"); dir != "" {
		cfg.Dir = dir
	} else if modelsDir := os.Getenv(models.EnvDir); modelsDir != "" {
		cfg.Dir = modelsDir
	}
	if rt := os.Getenv("NUMEN_OCR_RUNTIME"); rt != "" {
		cfg.Runtime = rt
	}

	recog := newRecognitionServer(nil, cfg)
	s.SetUpdateConfig(recog.updateConfig)
	s.Register(numenv1connect.NewRecognitionServiceHandler(recog, s.HandlerOptions()...))
	s.OnClose(func() { _ = recog.close() })
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
