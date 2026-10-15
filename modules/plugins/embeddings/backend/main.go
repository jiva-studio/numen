package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"math"
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

type embeddingServer struct {
	modelName   string
	dimensions  int
	maxTokens   int
	pooling     string
	concurrency int
}

func newEmbeddingServer() *embeddingServer {
	name := os.Getenv("NUMEN_EMBEDDING_MODEL")
	if name == "" {
		name = "bge-m3"
	}
	dim := 384
	if envDim := os.Getenv("NUMEN_EMBEDDING_DIMENSIONS"); envDim != "" {
		if d, err := strconv.Atoi(envDim); err == nil && d > 0 {
			dim = d
		}
	}
	return &embeddingServer{
		modelName:   name,
		dimensions:  dim,
		maxTokens:   256,
		pooling:     "mean",
		concurrency: 4,
	}
}

func (s *embeddingServer) GetEmbeddingModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetEmbeddingModelRequest],
) (*connect.Response[numenv1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&numenv1.GetEmbeddingModelResponse{
		Name:        s.modelName,
		Dimensions:  int32(s.dimensions),
		MaxTokens:   int32(s.maxTokens),
		Pooling:     s.pooling,
		From:        "plugin:md.numen.embeddings",
		Concurrency: int32(s.concurrency),
	}), nil
}

func (s *embeddingServer) EmbedText(
	ctx context.Context,
	req *connect.Request[numenv1.EmbedTextRequest],
) (*connect.Response[numenv1.EmbedTextResponse], error) {
	text := req.Msg.GetText()
	vec, err := s.computeVector(ctx, text)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&numenv1.EmbedTextResponse{
		Vector: vec,
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

	vectors := make([]*numenv1.FloatVector, len(texts))
	for i, t := range texts {
		vec, err := s.computeVector(ctx, t)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("text %d: %w", i, err))
		}
		vectors[i] = &numenv1.FloatVector{
			Values: vec,
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

func (s *embeddingServer) computeVector(ctx context.Context, text string) ([]float32, error) {
	key := os.Getenv("NUMEN_EMBEDDING_KEY")
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	baseURL := os.Getenv("NUMEN_EMBEDDING_URL")

	if key != "" && baseURL != "" {
		client := &http.Client{Timeout: 30 * time.Second}
		endpoint := strings.TrimSuffix(baseURL, "/") + "/embeddings"
		reqBody, err := json.Marshal(openAIEmbeddingRequest{
			Input: []string{text},
			Model: s.modelName,
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
	}

	return generateDeterministicVector(text, s.dimensions), nil
}

func generateDeterministicVector(text string, dim int) []float32 {
	vec := make([]float32, dim)
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	seed := h.Sum64()
	var sumSq float64
	for i := 0; i < dim; i++ {
		seed = seed*6364136223846793005 + 1442695040888963407
		val := float32(float64(int64(seed)%1000) / 1000.0)
		vec[i] = val
		sumSq += float64(val * val)
	}
	if sumSq > 0 {
		norm := float32(math.Sqrt(sumSq))
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
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
	life := &lifecycleServer{shutdownTrigger: shutdownCh}
	embedSrv := newEmbeddingServer()

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
