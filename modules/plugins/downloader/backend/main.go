package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/downloader/backend/download"
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

type downloadServer struct {
	downloader *download.Downloader
}

func (s *downloadServer) CanHandle(
	_ context.Context,
	req *connect.Request[numenv1.CanHandleRequest],
) (*connect.Response[numenv1.CanHandleResponse], error) {
	can := s.downloader.CanHandle(req.Msg.GetUrl())
	return connect.NewResponse(&numenv1.CanHandleResponse{
		CanHandle: can,
	}), nil
}

func (s *downloadServer) GetDownloadModel(
	_ context.Context,
	req *connect.Request[numenv1.GetDownloadModelRequest],
) (*connect.Response[numenv1.GetDownloadModelResponse], error) {
	model := s.downloader.GetDownloadModel(req.Msg.GetUrl())
	return connect.NewResponse(&numenv1.GetDownloadModelResponse{
		Tool:     model.Tool,
		Version:  model.Version,
		Producer: model.Producer,
	}), nil
}

func (s *downloadServer) GetMetadata(
	ctx context.Context,
	req *connect.Request[numenv1.GetMetadataRequest],
) (*connect.Response[numenv1.GetMetadataResponse], error) {
	meta, err := s.downloader.GetMetadata(ctx, req.Msg.GetUrl())
	if err != nil {
		if errors.Is(err, download.ErrNothingDownloaded) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}
	return connect.NewResponse(&numenv1.GetMetadataResponse{
		Title:       meta.Title,
		Bytes:       meta.Bytes,
		Length:      int32(meta.Length),
		Language:    meta.Language,
		Captions:    meta.Captions,
		IsAutomatic: meta.IsAutomatic,
	}), nil
}

func (s *downloadServer) GetText(
	ctx context.Context,
	req *connect.Request[numenv1.GetTextRequest],
) (*connect.Response[numenv1.GetTextResponse], error) {
	pref := download.PreferredCaptions{}
	if pc := req.Msg.GetPreferredCaptions(); pc != nil {
		pref.Languages = pc.GetLanguages()
		pref.IsAutomatic = pc.GetIsAutomatic()
	}
	text, err := s.downloader.GetText(ctx, req.Msg.GetUrl(), pref)
	if err != nil {
		if errors.Is(err, download.ErrNothingDownloaded) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}
	protoCues := make([]*numenv1.Cue, len(text.Cues))
	for i, c := range text.Cues {
		protoCues[i] = &numenv1.Cue{
			Text: c.Text,
			From: int32(c.From),
			To:   int32(c.To),
		}
	}
	return connect.NewResponse(&numenv1.GetTextResponse{
		Producer: text.Producer,
		Cues:     protoCues,
		Prose:    text.Prose,
		Title:    text.Title,
		Length:   int32(text.Length),
	}), nil
}

type streamChunkWriter struct {
	stream *connect.ServerStream[numenv1.DownloadMediaResponse]
}

func (w *streamChunkWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	err := w.stream.Send(&numenv1.DownloadMediaResponse{
		Chunk: p,
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *downloadServer) DownloadMedia(
	ctx context.Context,
	req *connect.Request[numenv1.DownloadMediaRequest],
	stream *connect.ServerStream[numenv1.DownloadMediaResponse],
) error {
	writer := &streamChunkWriter{stream: stream}
	copyInfo, err := s.downloader.Download(ctx, req.Msg.GetUrl(), writer)
	if err != nil {
		if errors.Is(err, download.ErrNothingDownloaded) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return err
	}
	// Send final response with media type and extension
	return stream.Send(&numenv1.DownloadMediaResponse{
		MediaType: copyInfo.MediaType,
		Extension: copyInfo.Extension,
	})
}

type authInterceptor struct {
	token string
}

func (a *authInterceptor) checkToken(header http.Header) error {
	if a.token == "" {
		return nil
	}
	authHeader := header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token != a.token {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("unauthorized"))
	}
	return nil
}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := a.checkToken(req.Header()); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := a.checkToken(conn.RequestHeader()); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

func authCheck(expectedToken string) connect.Interceptor {
	return &authInterceptor{token: expectedToken}
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

	ctx := context.Background()
	downloader, err := download.New(ctx, download.Defaults())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize downloader: %v\n", err)
		os.Exit(1)
	}

	shutdownCh := make(chan struct{})
	life := &lifecycleServer{shutdownTrigger: shutdownCh}
	downloadSvc := &downloadServer{downloader: downloader}

	interceptors := connect.WithInterceptors(authCheck(tokenFlag))

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life, interceptors))
	mux.Handle(numenv1connect.NewDownloadServiceHandler(downloadSvc, interceptors))

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
	_ = os.Stdout.Sync()

	stdinEOF := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				close(stdinEOF)
				return
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-shutdownCh:
	case <-stdinEOF:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
