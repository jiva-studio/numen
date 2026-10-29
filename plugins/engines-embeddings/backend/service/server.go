// Package service is the backend of the embeddings plugin.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

type embeddingServer struct {
	modelInfo  engine.ModelInfo
	localModel engine.LocalModel

	open   openEmbedder
	isMock bool

	mu       sync.Mutex
	embedder textEmbedder
	apiKey   string
	baseURL  string
	provider string
	// localName is the name of the model the plugin runs on this machine, and remoteModel the model
	// the settings name for a remote service.
	localName   string
	remoteModel string
}

// textEmbedder is a model that turns texts into vectors.
type textEmbedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Concurrency() int
	Close() error
}

// openEmbedder loads the model a server embeds with.
type openEmbedder func(ctx context.Context, info engine.ModelInfo, local engine.LocalModel) (textEmbedder, error)

// providerLocal is the provider that runs the model on this machine.
const providerLocal = "local"

// envMockML names the variable that selects the mock embedder for automated tests.
const envMockML = "NUMEN_MOCK_ML"

func openLocalModel(ctx context.Context, info engine.ModelInfo, local engine.LocalModel) (textEmbedder, error) {
	emb, err := engine.Open(ctx, info, local, nil)
	if err != nil {
		return nil, err
	}
	return emb, nil
}

func openMockModel(context.Context, engine.ModelInfo, engine.LocalModel) (textEmbedder, error) {
	return engine.MockEmbedder{}, nil
}

// updateConfig applies the settings of the selected provider: the host passes
// the URL of a custom service, and the model and key of a remote one, only
// while that provider is selected. A key that the host omits is kept.
func (s *embeddingServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if provider := values["provider"]; provider != "" {
		s.provider = provider
	}
	model := values["model"]
	if model != "" && !isLocalPresetModel(model) {
		s.remoteModel = model
	}
	if s.provider != providerLocal && s.provider != "" && s.remoteModel == "" {
		s.remoteModel = getDefaultRemoteModel(s.provider)
	}
	if key, ok := server.ReadSecret(values, server.SecretAPIKey); ok {
		s.apiKey = key
	}
	s.baseURL = values["baseUrl"]
	if s.provider == providerLocal || s.provider == "" {
		s.updateLocalModelLocked(model)
	}
	s.modelInfo.Name = s.localName
	if s.provider != providerLocal && s.remoteModel != "" {
		s.modelInfo.Name = s.remoteModel
	}
	if !s.isMock {
		s.modelInfo.Dimensions = resolveModelDimensions(s.modelInfo.Name)
		applyDimensionEnv(&s.modelInfo)
	}
}

func isLocalPresetModel(model string) bool {
	switch model {
	case "Xenova/bge-m3", "Xenova/multilingual-e5-small", "Xenova/bge-small-en-v1.5", "bge-m3", "bge-m3-int8":
		return true
	default:
		return false
	}
}

func getDefaultRemoteModel(provider string) string {
	if provider == "openrouter" {
		return "openai/text-embedding-3-small"
	}
	return "text-embedding-3-small"
}

func resolveModelDimensions(model string) int {
	switch model {
	case "Xenova/multilingual-e5-small", "Xenova/bge-small-en-v1.5", "Xenova/all-MiniLM-L6-v2", "Xenova/all-MiniLM-L12-v2":
		return 384
	case "openai/text-embedding-3-small", "text-embedding-3-small", "openai/text-embedding-ada-002", "text-embedding-ada-002":
		return 1536
	case "openai/text-embedding-3-large", "text-embedding-3-large":
		return 3072
	case "qwen/qwen3-embedding-8b":
		return 4096
	default:
		return 1024
	}
}

func (s *embeddingServer) updateLocalModelLocked(model string) {
	if s.isMock {
		return
	}
	prevRepo := s.localModel.Name
	prevName := s.localName
	switch model {
	case "", "Xenova/bge-m3", "bge-m3", "bge-m3-int8",
		"openai/text-embedding-3-small", "text-embedding-3-small",
		"openai/text-embedding-3-large", "text-embedding-3-large",
		"openai/text-embedding-ada-002", "text-embedding-ada-002",
		"baai/bge-m3", "qwen/qwen3-embedding-8b":
		defInfo, defLocal := engine.DefaultBGEM3Config()
		applyModelOverrides(&defInfo, &defLocal)
		applyRuntimeEnv(&defLocal)
		s.localModel = defLocal
		s.localName = defInfo.Name
	default:
		s.localModel = s.localModel.WithRepository(model)
		s.localModel.File = ""
		s.localName = model
	}
	if (s.localModel.Name != prevRepo || s.localName != prevName) && s.embedder != nil {
		_ = s.embedder.Close()
		s.embedder = nil
	}
}

func newEmbeddingServer() *embeddingServer {
	if os.Getenv(envMockML) == "true" {
		info := engine.GetMockModel()
		return &embeddingServer{modelInfo: info, localName: info.Name, open: openMockModel, isMock: true}
	}
	info, local := engine.DefaultBGEM3Config()
	applyModelOverrides(&info, &local)
	applyDimensionEnv(&info)
	applyRuntimeEnv(&local)

	return &embeddingServer{
		modelInfo:  info,
		localName:  info.Name,
		localModel: local,
		open:       openLocalModel,
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

func (s *embeddingServer) getEmbedder(ctx context.Context) (textEmbedder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.embedder != nil {
		return s.embedder, nil
	}

	localInfo := s.modelInfo
	if !s.isMock {
		localInfo.Name = s.localName
		localInfo.Dimensions = resolveModelDimensions(s.localName)
		applyDimensionEnv(&localInfo)
	}
	emb, err := s.open(ctx, localInfo, s.localModel)
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

func (s *embeddingServer) GetModelInfo(_ context.Context) (embedding.ModelInfo, error) {
	s.mu.Lock()
	info := s.modelInfo
	s.mu.Unlock()
	passes := info.Concurrency
	if passes <= 0 {
		passes = 4
	}
	return embedding.ModelInfo{
		Name:        info.Name,
		Dimensions:  int32(info.Dimensions),
		MaxTokens:   int32(info.MaxTokens),
		Pooling:     info.Pooling,
		From:        info.From,
		Concurrency: int32(getIndexShare(passes)),
	}, nil
}

// getIndexShare returns the number of concurrent index calls allowed on a model
// that runs several passes at once: one fewer than the passes, and at least
// one, so a query on the same model always has a free pass.
func getIndexShare(passes int) int {
	return max(passes-1, 1)
}

func (s *embeddingServer) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	key, baseURL := s.getService()
	if key != "" && baseURL != "" {
		return s.embedRemoteTexts(ctx, key, baseURL, texts)
	}
	if s.isRemoteProvider() {
		if baseURL == "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("embedding service URL is not configured"))
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("embedding API key is not configured"))
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

	return denseVectors, nil
}

func (s *embeddingServer) embedRemoteTexts(ctx context.Context, key, baseURL string, texts []string) ([][]float32, error) {
	model := s.getModelName()
	vectors := make([][]float32, len(texts))
	for i, t := range texts {
		vec, err := computeServiceVector(ctx, key, baseURL, model, t)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("service text %d: %w", i, err))
		}
		if len(vec) > 0 {
			s.mu.Lock()
			s.modelInfo.Dimensions = len(vec)
			s.mu.Unlock()
		}
		vectors[i] = vec
	}
	return vectors, nil
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

// isRemoteProvider reports whether the configured provider is a remote HTTP service.
func (s *embeddingServer) isRemoteProvider() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.isMock && s.provider != "" && s.provider != providerLocal
}

// getService returns the key Configure handed over and the endpoint of the
// remote service: the configured one, else the one in NUMEN_EMBEDDING_URL. The
// remote service answers only when both are set.
func (s *embeddingServer) getService() (key, baseURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.isMock || s.provider == providerLocal:
		return "", ""
	case s.provider == "openai":
		return s.apiKey, "https://api.openai.com/v1"
	case s.provider == "openrouter":
		return s.apiKey, "https://openrouter.ai/api/v1"
	}
	baseURL = s.baseURL
	if baseURL == "" {
		baseURL = os.Getenv("NUMEN_EMBEDDING_URL")
	}
	return s.apiKey, baseURL
}

// getModelName returns the name of the model that creates the vectors.
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
	embedding.Mount(s, embedSrv)
	s.OnClose(func() { _ = embedSrv.Close() })
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
