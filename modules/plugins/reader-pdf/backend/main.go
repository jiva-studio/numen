package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image/jpeg"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/pdf"
)

type lifecycleServer struct {
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

type documentServer struct{}

func (s *documentServer) GetDocument(
	_ context.Context,
	req *connect.Request[numenv1.GetDocumentRequest],
) (*connect.Response[numenv1.GetDocumentResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("path traversal outside allowed scope"))
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("open file: %w", err))
	}
	defer file.Close()

	scan, err := pdf.OpenReader(file)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("read pdf: %w", err))
	}
	defer scan.Close()

	pageCount := scan.GetPageCount()
	pages := make([]*numenv1.Page, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		w, h, err := scan.GetPageSize(i)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("page size: %w", err))
		}
		pages = append(pages, &numenv1.Page{
			Width:  w,
			Height: h,
		})
	}

	return connect.NewResponse(&numenv1.GetDocumentResponse{
		Pages: pages,
	}), nil
}

func (s *documentServer) RenderPage(
	_ context.Context,
	req *connect.Request[numenv1.RenderPageRequest],
) (*connect.Response[numenv1.RenderPageResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("path traversal outside allowed scope"))
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("open file: %w", err))
	}
	defer file.Close()

	scan, err := pdf.OpenReader(file)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("read pdf: %w", err))
	}
	defer scan.Close()

	dpi := int(req.Msg.GetDpi())
	if dpi <= 0 {
		dpi = 150
	}

	img, err := scan.RenderPage(int(req.Msg.GetPage()), dpi)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("render page: %w", err))
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode jpeg: %w", err))
	}

	bounds := img.Bounds()
	return connect.NewResponse(&numenv1.RenderPageResponse{
		ImageData:   buf.Bytes(),
		ContentType: "image/jpeg",
		Width:       float64(bounds.Dx()),
		Height:      float64(bounds.Dy()),
	}), nil
}

func (s *documentServer) ExtractText(
	_ context.Context,
	req *connect.Request[numenv1.ExtractTextRequest],
) (*connect.Response[numenv1.ExtractTextResponse], error) {
	raw := req.Msg.GetData()
	if len(raw) == 0 {
		path := req.Msg.GetPath()
		if path == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path or data required"))
		}
		var err error
		raw, err = os.ReadFile(path)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read file: %w", err))
		}
	}

	book, err := pdf.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("parse pdf: %w", err))
	}

	starts := make([]int32, len(book.Pages))
	for i, p := range book.Pages {
		starts[i] = int32(p.Offset)
	}

	return connect.NewResponse(&numenv1.ExtractTextResponse{
		Text:       book.Text,
		PageStarts: starts,
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
	doc := &documentServer{}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life))
	mux.Handle(numenv1connect.NewDocumentServiceHandler(doc))

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
