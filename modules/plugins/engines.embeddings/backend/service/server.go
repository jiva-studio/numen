// Package service is the backend of the embeddings plugin.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/engines.embeddings/backend/engine"
)

type embeddingServer struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler

	modelInfo  engine.ModelInfo
	localModel engine.LocalModel

	mu       sync.Mutex
	embedder *engine.Embedder
	apiKey   string
	baseURL  string
	provider string
}

func (s *embeddingServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if provider, ok := values["provider"]; ok && provider != "" {
		s.provider = provider
	}
	if model, ok := values["model"]; ok && model != "" {
		s.modelInfo.Name = model
	}
	if key, ok := server.ReadSecret(values, server.SecretAPIKey); ok {
		s.apiKey = key
	}
	if base, ok := values["baseUrl"]; ok {
		s.baseURL = base
	}
}

func newEmbeddingServer() *embeddingServer {
	info, local := engine.DefaultBGEM3Config()
	applyModelOverrides(&info, &local)
	applyDimensionEnv(&info)
	applyRuntimeEnv(&local)

	return &embeddingServer{
		modelInfo:  info,
		localModel: local,
	}
}

func applyModelOverrides(info *engine.ModelInfo, local *engine.LocalModel) {
	if name := os.Getenv("NUMEN_EMBEDDING_MODEL"); name != "" {
		info.Name = name
		if name != "bge-m3" && name != "bge-m3-int8" {
			*local = local.WithRepository(name)
		}
	}
	if repo := os.Getenv("NUMEN_EMBEDDING_REPO"); repo != "" {
		*local = local.WithRepository(repo)
	}
	if dir := os.Getenv("NUMEN_EMBEDDING_DIR"); dir != "" {
		local.Dir = dir
	}
	if file := os.Getenv("NUMEN_EMBEDDING_FILE"); file != "" {
		local.File = file
	}
}

func parsePositiveInt(val string) int {
	if val == "" {
		return 0
	}
	if d, err := strconv.Atoi(val); err == nil && d > 0 && d <= math.MaxInt32 {
		return d
	}
	return 0
}

func applyDimensionEnv(info *engine.ModelInfo) {
	if d := parsePositiveInt(os.Getenv("NUMEN_EMBEDDING_DIMENSIONS")); d > 0 {
		info.Dimensions = d
	}
	if t := parsePositiveInt(os.Getenv("NUMEN_EMBEDDING_MAX_TOKENS")); t > 0 {
		info.MaxTokens = t
	}
	if pooling := os.Getenv("NUMEN_EMBEDDING_POOLING"); pooling != "" {
		info.Pooling = pooling
	}
}

func applyRuntimeEnv(local *engine.LocalModel) {
	if runtimeLib := os.Getenv("NUMEN_ONNX_RUNTIME"); runtimeLib != "" {
		local.Runtime = runtimeLib
	}
	if threads := os.Getenv("NUMEN_ONNX_THREADS"); threads != "" {
		if th, err := strconv.Atoi(threads); err == nil && th > 0 {
			local.Threads = th
		}
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
	s.mu.Lock()
	info := s.modelInfo
	s.mu.Unlock()
	concurrency := info.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	return connect.NewResponse(&numenv1.GetEmbeddingModelResponse{
		Name:        info.Name,
		Dimensions:  int32(info.Dimensions),
		MaxTokens:   int32(info.MaxTokens),
		Pooling:     info.Pooling,
		From:        info.From,
		Concurrency: int32(concurrency),
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

	if key, baseURL := s.getService(); key != "" && baseURL != "" {
		model := s.getModelName()
		vectors := make([]*numenv1.FloatVector, len(texts))
		for i, t := range texts {
			vec, err := computeServiceVector(ctx, key, baseURL, model, t)
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

// getService returns the key Configure handed over and the endpoint of the
// remote service: the configured one, else the one in NUMEN_EMBEDDING_URL. The
// remote service answers only when both are set.
func (s *embeddingServer) getService() (key, baseURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.provider == "local" {
		return "", ""
	}
	baseURL = s.baseURL
	if baseURL == "" {
		baseURL = os.Getenv("NUMEN_EMBEDDING_URL")
	}
	if s.provider == "openai" && baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	} else if s.provider == "openrouter" && baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	return s.apiKey, baseURL
}

// getModelName returns the name of the model the vectors are asked for.
func (s *embeddingServer) getModelName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.modelInfo.Name
}

func computeServiceVector(ctx context.Context, key, baseURL, model, text string) ([]float32, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := strings.TrimSuffix(baseURL, "/") + "/embeddings"
	reqBody, err := json.Marshal(openAIEmbeddingRequest{
		Input: []string{text},
		Model: model,
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

// Mount registers the embedding service on s, with the model the environment configures. The
// embedder closes with s.
func Mount(s *server.Server) error {
	embedSrv := newEmbeddingServer()
	s.SetUpdateConfig(embedSrv.updateConfig)
	s.Register(numenv1connect.NewEmbeddingServiceHandler(embedSrv, s.HandlerOptions()...))
	s.OnClose(func() { _ = embedSrv.Close() })
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
