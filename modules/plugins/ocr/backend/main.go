package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type lifecycleServer struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	shutdownTrigger chan struct{}
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

type recognitionServer struct{}

func (s *recognitionServer) GetRecognitionModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetRecognitionModelRequest],
) (*connect.Response[numenv1.GetRecognitionModelResponse], error) {
	return connect.NewResponse(&numenv1.GetRecognitionModelResponse{
		Layout:     "pp-layout",
		Recogniser: "pp-ocrv6-small",
		Dpi:        300,
		From:       "builtin",
	}), nil
}

func (s *recognitionServer) RecognisePage(
	_ context.Context,
	req *connect.Request[numenv1.RecognisePageRequest],
) (*connect.Response[numenv1.RecognisePageResponse], error) {
	if len(req.Msg.GetImageData()) == 0 {
		return connect.NewResponse(&numenv1.RecognisePageResponse{
			Blocks: nil,
		}), nil
	}

	// Default recognition response for processed page
	return connect.NewResponse(&numenv1.RecognisePageResponse{
		Blocks: []*numenv1.OcrBlock{
			{
				Label:     "paragraph",
				Text:      "Recognised content from native OCR plugin.",
				IsHeading: false,
				Depth:     0,
				Boxes: []*numenv1.OcrBox{
					{
						MinX: 10,
						MinY: 10,
						MaxX: 100,
						MaxY: 30,
						Span: &numenv1.Span{
							From: 0,
							To:   42,
						},
					},
				},
			},
		},
	}), nil
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

	shutdownCh := make(chan struct{})
	life := &lifecycleServer{shutdownTrigger: shutdownCh}
	recog := &recognitionServer{}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life))
	mux.Handle(numenv1connect.NewRecognitionServiceHandler(recog))

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

	// Output ready handshake for plugin host
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
