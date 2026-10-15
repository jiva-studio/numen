package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"
	_ "golang.org/x/image/webp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/engine"
)

type lifecycleServer struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	shutdownTrigger chan struct{}
	recog           *recognitionServer
}

func (s *lifecycleServer) Ping(
	_ context.Context,
	_ *connect.Request[numenv1.PingRequest],
) (*connect.Response[numenv1.PingResponse], error) {
	return connect.NewResponse(&numenv1.PingResponse{
		Status: "ok",
	}), nil
}

func (s *lifecycleServer) Shutdown(
	_ context.Context,
	_ *connect.Request[numenv1.ShutdownRequest],
) (*connect.Response[numenv1.ShutdownResponse], error) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(s.shutdownTrigger)
	}()
	return connect.NewResponse(&numenv1.ShutdownResponse{
		Success: true,
	}), nil
}

func (s *lifecycleServer) Configure(
	_ context.Context,
	req *connect.Request[numenv1.ConfigureRequest],
) (*connect.Response[numenv1.ConfigureResponse], error) {
	if s.recog != nil {
		s.recog.updateConfig(req.Msg.GetValues())
	}
	return connect.NewResponse(&numenv1.ConfigureResponse{
		Success: true,
	}), nil
}

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

func authCheck(expectedToken string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if expectedToken != "" {
				authHeader := req.Header().Get("Authorization")
				expected := "Bearer " + expectedToken
				if authHeader != expected {
					return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
				}
			}
			return next(ctx, req)
		}
	}
}

func main() {
	var portFlag int
	var tokenFlag string

	flag.IntVar(&portFlag, "port", 0, "Port to listen on")
	flag.StringVar(&tokenFlag, "token", "", "Authentication token")
	flag.Parse()

	if portFlag == 0 {
		if envPort := os.Getenv("NUMEN_PLUGIN_PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil {
				portFlag = p
			}
		}
	}
	if tokenFlag == "" {
		tokenFlag = os.Getenv("NUMEN_PLUGIN_TOKEN")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", portFlag))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port

	cfg := engine.Defaults()
	if dir := os.Getenv("NUMEN_OCR_DIR"); dir != "" {
		cfg.Dir = dir
	} else if modelsDir := os.Getenv("NUMEN_MODELS_DIR"); modelsDir != "" {
		cfg.Dir = modelsDir
	}
	if rt := os.Getenv("NUMEN_OCR_RUNTIME"); rt != "" {
		cfg.Runtime = rt
	}

	shutdownCh := make(chan struct{})
	recog := newRecognitionServer(nil, cfg)
	defer recog.close()
	life := &lifecycleServer{shutdownTrigger: shutdownCh, recog: recog}

	interceptors := connect.WithInterceptors(authCheck(tokenFlag))

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life, interceptors))
	mux.Handle(numenv1connect.NewRecognitionServiceHandler(recog, interceptors))

	handler := h2c.NewHandler(mux, &http2.Server{})
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	fmt.Printf("NUMEN_READY port=%d\n", actualPort)
	os.Stdout.Sync()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-shutdownCh:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
