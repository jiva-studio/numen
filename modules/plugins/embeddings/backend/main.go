package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/embeddings/backend/engine"
)

type lifecycleServer struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	shutdownTrigger chan struct{}
	emb             *embeddingServer
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
	if s.emb != nil {
		s.emb.updateConfig(req.Msg.GetValues())
	}
	return connect.NewResponse(&numenv1.ConfigureResponse{
		Success: true,
	}), nil
}

type embeddingServer struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler

	modelInfo  engine.ModelInfo
	localModel engine.LocalModel

	mu       sync.Mutex
	embedder *engine.Embedder
}

func (s *embeddingServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if model, ok := values["model"]; ok && model != "" {
		s.modelInfo.Name = model
	}
}

func newEmbeddingServer() *embeddingServer {
	info, local := engine.DefaultBGEM3Config()

	if name := os.Getenv("NUMEN_EMBEDDING_MODEL"); name != "" {
		info.Name = name
		if name != "bge-m3" && name != "bge-m3-int8" {
			local.Name = name
		}
	}
	if repo := os.Getenv("NUMEN_EMBEDDING_REPO"); repo != "" {
		local.Name = repo
	}
	if dir := os.Getenv("NUMEN_EMBEDDING_DIR"); dir != "" {
		local.Dir = dir
	}
	if file := os.Getenv("NUMEN_EMBEDDING_FILE"); file != "" {
		local.File = file
	}
	if envDim := os.Getenv("NUMEN_EMBEDDING_DIMENSIONS"); envDim != "" {
		if d, err := strconv.Atoi(envDim); err == nil && d > 0 {
			info.Dimensions = d
		}
	}
	if envTokens := os.Getenv("NUMEN_EMBEDDING_MAX_TOKENS"); envTokens != "" {
		if t, err := strconv.Atoi(envTokens); err == nil && t > 0 {
			info.MaxTokens = t
		}
	}
	if pooling := os.Getenv("NUMEN_EMBEDDING_POOLING"); pooling != "" {
		info.Pooling = pooling
	}
	if runtimeLib := os.Getenv("NUMEN_ONNX_RUNTIME"); runtimeLib != "" {
		local.Runtime = runtimeLib
	}
	if provider := os.Getenv("NUMEN_ONNX_PROVIDER"); provider != "" {
		local.Provider = provider
	}
	if threads := os.Getenv("NUMEN_ONNX_THREADS"); threads != "" {
		if th, err := strconv.Atoi(threads); err == nil && th > 0 {
			local.Threads = th
		}
	}

	return &embeddingServer{
		modelInfo:  info,
		localModel: local,
	}
}

func (s *embeddingServer) getEmbedder(ctx context.Context) (*engine.Embedder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.embedder != nil {
		return s.embedder, nil
	}

	emb, err := engine.Open(ctx, s.modelInfo, s.localModel, nil)
	if err != nil {
		return nil, fmt.Errorf("opening onnx embedding engine: %w", err)
	}
	s.embedder = emb
	s.modelInfo.Concurrency = emb.Concurrency()
	return emb, nil
}

func (s *embeddingServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.embedder != nil {
		err := s.embedder.Close()
		s.embedder = nil
		return err
	}
	return nil
}

func (s *embeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetEmbeddingModelRequest],
) (*connect.Response[numenv1.GetEmbeddingModelResponse], error) {
	concurrency := s.modelInfo.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	return connect.NewResponse(&numenv1.GetEmbeddingModelResponse{
		Name:        s.modelInfo.Name,
		Dimensions:  int32(s.modelInfo.Dimensions),
		MaxTokens:   int32(s.modelInfo.MaxTokens),
		Pooling:     s.modelInfo.Pooling,
		From:        s.modelInfo.From,
		Concurrency: int32(concurrency),
	}), nil
}

func (s *embeddingServer) EmbedText(
	ctx context.Context,
	req *connect.Request[numenv1.EmbedTextRequest],
) (*connect.Response[numenv1.EmbedTextResponse], error) {
	text := req.Msg.GetText()

	if isServiceConfigured() {
		vec, err := s.computeServiceVector(ctx, text)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&numenv1.EmbedTextResponse{
			Vector: vec,
		}), nil
	}

	emb, err := s.getEmbedder(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	vectors, err := emb.Embed(ctx, []string{text})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("embed text: %w", err))
	}
	if len(vectors) == 0 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("no vector returned for text"))
	}

	return connect.NewResponse(&numenv1.EmbedTextResponse{
		Vector: vectors[0],
	}), nil
}

func (s *embeddingServer) EmbedBatch(
	ctx context.Context,
	req *connect.Request[numenv1.EmbedBatchRequest],
) (*connect.Response[numenv1.EmbedBatchResponse], error) {
	texts := req.Msg.GetTexts()
	if len(texts) == 0 {
		return connect.NewResponse(&numenv1.EmbedBatchResponse{
			Vectors: nil,
		}), nil
	}

	if isServiceConfigured() {
		vectors := make([]*numenv1.FloatVector, len(texts))
		for i, t := range texts {
			vec, err := s.computeServiceVector(ctx, t)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("service text %d: %w", i, err))
			}
			vectors[i] = &numenv1.FloatVector{
				Values: vec,
			}
		}
		return connect.NewResponse(&numenv1.EmbedBatchResponse{
			Vectors: vectors,
		}), nil
	}

	emb, err := s.getEmbedder(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	denseVectors, err := emb.Embed(ctx, texts)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("embed batch: %w", err))
	}
	if len(denseVectors) != len(texts) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("expected %d vectors, got %d", len(texts), len(denseVectors)))
	}

	vectors := make([]*numenv1.FloatVector, len(denseVectors))
	for i, v := range denseVectors {
		vectors[i] = &numenv1.FloatVector{
			Values: v,
		}
	}

	return connect.NewResponse(&numenv1.EmbedBatchResponse{
		Vectors: vectors,
	}), nil
}

type openAIEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func isServiceConfigured() bool {
	key := os.Getenv("NUMEN_EMBEDDING_KEY")
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	baseURL := os.Getenv("NUMEN_EMBEDDING_URL")
	return key != "" && baseURL != ""
}

func (s *embeddingServer) computeServiceVector(ctx context.Context, text string) ([]float32, error) {
	key := os.Getenv("NUMEN_EMBEDDING_KEY")
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	baseURL := os.Getenv("NUMEN_EMBEDDING_URL")
	if key == "" || baseURL == "" {
		return nil, fmt.Errorf("remote service configuration is missing")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := strings.TrimSuffix(baseURL, "/") + "/embeddings"
	reqBody, err := json.Marshal(openAIEmbeddingRequest{
		Input: []string{text},
		Model: s.modelInfo.Name,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBytes))
	}

	var openAIResp openAIEmbeddingResponse
	if err := json.Unmarshal(respBytes, &openAIResp); err != nil {
		return nil, err
	}
	if len(openAIResp.Data) > 0 && len(openAIResp.Data[0].Embedding) > 0 {
		return openAIResp.Data[0].Embedding, nil
	}

	return nil, fmt.Errorf("no embedding returned by service")
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

	shutdownCh := make(chan struct{})
	embedSrv := newEmbeddingServer()
	life := &lifecycleServer{shutdownTrigger: shutdownCh, emb: embedSrv}

	interceptors := connect.WithInterceptors(authCheck(tokenFlag))

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life, interceptors))
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(embedSrv, interceptors))

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

	_ = embedSrv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
