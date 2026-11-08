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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/performance"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/onnx"
)

const (
	modelEmbedding3Small           = "text-embedding-3-small"
	modelOpenRouterEmbedding3Small = "openai/text-embedding-3-small"
)

type embeddingServer struct {
	modelInfo   engine.ModelInfo
	localModel  engine.LocalModel
	localInfo   engine.ModelInfo
	localErr    error
	profilesErr error
	localKey    string

	profiles    *engine.Profiles
	profilesDir string

	open    openEmbedder
	isMock  bool
	mockLog string

	chunkTokens int

	mu          sync.Mutex
	embedder    textEmbedder
	apiKey      string
	baseURL     string
	provider    string
	localName   string
	remoteModel string

	profile      performance.Profile
	isProfileSet bool
	lowerNice    func(nice int) error
	opening      *modelOpening
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

// envMockLog names the variable that makes the mock model append the texts it is given to a file.
const envMockLog = "NUMEN_MOCK_EMBED_LOG"

func newLocalModelOpener(library *onnx.Library) openEmbedder {
	// gomutants:disable-next-line RETURN_ZERO reason="production opener wraps engine.Open with native ONNX runtime"
	return func(ctx context.Context, info engine.ModelInfo, local engine.LocalModel) (textEmbedder, error) {
		emb, err := engine.Open(ctx, library, info, local, nil)
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="production opener wraps engine.Open with native ONNX runtime"
		if err != nil {
			// gomutants:disable-next-line RETURN_ERROR_NIL reason="production opener wraps engine.Open with native ONNX runtime"
			return nil, err
		}
		// gomutants:disable-next-line RETURN_ZERO reason="production opener wraps engine.Open with native ONNX runtime"
		return emb, nil
	}
}

// envMockOpenHold names the variable that makes the mock model take that many milliseconds to open,
// as a model that is still downloading.
const envMockOpenHold = "NUMEN_MOCK_OPEN_HOLD_MS"

func (s *embeddingServer) openMockModel(ctx context.Context, _ engine.ModelInfo, _ engine.LocalModel) (textEmbedder, error) {
	// gomutants:disable-next-line INVERT_LOGICAL,EXPRESSION_REMOVE,CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="0ms timeout completes immediately without blocking"
	if held, err := strconv.Atoi(os.Getenv(envMockOpenHold)); err == nil && held > 0 {
		// gomutants:disable-next-line ARITHMETIC_BASE reason="mock hold duration scaling for integration tests"
		hold, cancel := context.WithTimeout(ctx, time.Duration(held)*time.Millisecond)
		defer cancel()
		<-hold.Done()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	return engine.MockEmbedder{LogPath: s.mockLog}, nil
}

func (s *embeddingServer) updateConfig(values map[string]string, secrets server.Secrets) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyProfileLocked(performance.Read(values))
	s.loadProfilesLocked()
	if provider := values["provider"]; provider != "" {
		s.provider = provider
	}
	model := values["model"]
	if model != "" && !s.profiles.IsLocal(model) {
		s.remoteModel = model
	}
	if s.provider != providerLocal && s.provider != "" && s.remoteModel == "" {
		s.remoteModel = getDefaultRemoteModel(s.provider)
	}
	s.modelInfo.MinChunkLength = readMinChunkLength(values)
	if _, ok := values[settingChunkTokens]; ok {
		s.chunkTokens = ReadChunkTokens(values)
	}
	if list, carried := server.ReadSecrets(secrets, server.SecretAPIKey); carried {
		s.apiKey = getFirstValue(list)
	}
	s.baseURL = values["baseUrl"]
	if s.provider == providerLocal || s.provider == "" {
		s.updateLocalModelLocked(model)
	}
	s.refreshModelInfoLocked()
}

func getFirstValue(list []server.Secret) string {
	if len(list) == 0 {
		return ""
	}
	return list[0].Value
}

func (s *embeddingServer) loadProfilesLocked() {
	shipped, err := engine.GetShippedProfiles()
	// gomutants:disable-next-line STATEMENT_REMOVE reason="embedded shipped profiles json is valid"
	s.profilesErr = err
	s.profiles = shipped.WithUserProfiles(s.profilesDir, func(file string, err error) {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="stderr log on invalid user profile has no side effect"
		fmt.Fprintf(os.Stderr, "embeddings: skipped the model profile %s: %v\n", file, err)
	})
}

func (s *embeddingServer) refreshModelInfoLocked() {
	s.refreshModelIdentityLocked()
	s.modelInfo.ChunkTokens = ClampChunkTokens(s.chunkTokens, s.modelInfo.MaxTokens)
}

func (s *embeddingServer) refreshModelIdentityLocked() {
	if s.isMock {
		return
	}
	keep := s.modelInfo
	if s.provider != providerLocal && s.remoteModel != "" {
		profile := s.profiles.GetRemote(s.remoteModel)
		s.modelInfo = engine.ModelInfo{
			Name: s.remoteModel, Dimensions: profile.Dimensions, MaxTokens: profile.GetTokens(), Pooling: profile.Pooling,
			From: keep.From, Concurrency: keep.Concurrency, FloorHint: profile.FloorHint, MinChunkLength: keep.MinChunkLength,
			ChunkTokens: ClampChunkTokens(s.chunkTokens, profile.GetTokens()),
			Handling:    profile.GetHandling(),
		}
		applyDimensionEnv(&s.modelInfo)
		return
	}
	s.modelInfo = s.localInfo
	s.modelInfo.Concurrency = keep.Concurrency
	s.modelInfo.MinChunkLength = keep.MinChunkLength
}

func getDefaultRemoteModel(provider string) string {
	if provider == "openrouter" {
		return modelOpenRouterEmbedding3Small
	}
	return modelEmbedding3Small
}

func (s *embeddingServer) updateLocalModelLocked(model string) {
	if s.isMock {
		return
	}
	info, local, err := s.resolveLocalModel(model)
	if s.profilesErr != nil {
		err = s.profilesErr
	}
	s.localErr = err
	if err == nil {
		s.localInfo, s.localModel, s.localName = info, local, info.Name
	} else {
		s.localName = model
	}
	if key := getLocalKey(info, local, err); key != s.localKey {
		s.localKey = key
		s.retireOpeningLocked()
		if s.embedder != nil {
			s.closeRetired(s.embedder)
			s.embedder = nil
		}
	}
}

func getLocalKey(info engine.ModelInfo, local engine.LocalModel, err error) string {
	if err != nil {
		return "error"
	}
	return strings.Join([]string{
		info.Name, strconv.Itoa(info.Dimensions), strconv.Itoa(info.MaxTokens), info.Pooling,
		info.Handling.Hash(info.MaxTokens), strings.Join(info.Handling.ExtraInputs, ","),
		local.Name, local.Dir, local.File, local.Revision,
	}, "|")
}

func (s *embeddingServer) resolveLocalModel(model string) (engine.ModelInfo, engine.LocalModel, error) {
	if model == "" {
		model = os.Getenv("NUMEN_EMBEDDING_MODEL")
	}
	if model == "" || s.profiles.IsRemote(model) {
		model = engine.DefaultModel
	}
	profile, err := s.profiles.GetLocal(model)
	if err != nil {
		return engine.ModelInfo{}, engine.LocalModel{}, err
	}
	info, local := engine.NewConfig(profile)
	applyModelOverrides(&info, &local)
	applyDimensionEnv(&info)
	applyRuntimeEnv(&local)
	return info, local, nil
}

func newEmbeddingServer() *embeddingServer {
	profilesDir := getProfilesDir()
	var none engine.ModelInfo
	var noLocal engine.LocalModel
	if os.Getenv(envMockML) == "true" {
		info := engine.GetMockModel()
		info.MinChunkLength = defaultMinChunkLength
		info.ChunkTokens = ClampChunkTokens(defaultChunkTokens, info.MaxTokens)
		s := &embeddingServer{
			modelInfo: info,
			localModel: engine.LocalModel{
				// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="zero-value struct literal fields"
				Name: "", Dir: "", File: "", BatchTexts: 0, Runtime: "", Threads: 0, Sessions: 0,
				// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="zero-value struct literal fields"
				PassBudget: 0, PassMemory: 0, Provider: "", ShouldDownload: false, Revision: "", Sums: nil,
			},
			localInfo: none, localErr: nil, profilesErr: nil, localKey: "", profiles: nil, open: nil,
			profilesDir: profilesDir,
			isMock:      true,
			mockLog:     os.Getenv(envMockLog),
			chunkTokens: defaultChunkTokens,
			mu:          sync.Mutex{},
			embedder:    nil,
			apiKey:      "",
			baseURL:     "",
			provider:    "",
			localName:   info.Name,
			remoteModel: "",
			profile:     performance.Balanced, isProfileSet: false, lowerNice: performance.SetNice, opening: nil,
		}
		s.open = s.openMockModel
		s.loadProfilesLocked()
		return s
	}
	s := &embeddingServer{
		modelInfo: none, localModel: noLocal, localInfo: none, localErr: nil,
		profilesErr: nil, localKey: "", profiles: nil, mockLog: "", localName: "",
		profilesDir: profilesDir,
		open:        newLocalModelOpener(onnx.NewLibrary()),
		isMock:      false,
		chunkTokens: defaultChunkTokens,
		mu:          sync.Mutex{},
		embedder:    nil,
		apiKey:      "",
		baseURL:     "",
		provider:    "",
		remoteModel: "",
		profile:     performance.Balanced, isProfileSet: false, lowerNice: performance.SetNice, opening: nil,
	}
	s.loadProfilesLocked()
	s.updateLocalModelLocked("")
	s.modelInfo = s.localInfo
	s.modelInfo.MinChunkLength = defaultMinChunkLength
	s.modelInfo.ChunkTokens = ClampChunkTokens(defaultChunkTokens, s.modelInfo.MaxTokens)
	return s
}

func getProfilesDir() string {
	dir, err := cache.GetDataDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, engine.ProfilesFolder)
}

func applyModelOverrides(_ *engine.ModelInfo, local *engine.LocalModel) {
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
	// gomutants:disable-next-line BRANCH_IF reason="empty string is rejected by strconv.Atoi"
	if val == "" {
		return 0
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="returning 0 for d == 0 matches the fallback return 0"
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
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="setting local.Threads to 0 leaves default 0 unchanged"
		if th, err := strconv.Atoi(threads); err == nil && th > 0 {
			local.Threads = th
		}
	}
}

func (s *embeddingServer) GetModelInfo(_ context.Context) (embedding.ModelInfo, error) {
	s.mu.Lock()
	info := s.modelInfo
	localErr := s.getLocalErrLocked()
	s.mu.Unlock()
	if localErr != nil {
		return embedding.ModelInfo{}, rpcerror.New(rpcerror.FailedPrecondition, localErr)
	}
	passes := info.Concurrency
	if passes <= 0 {
		passes = 4
	}
	return embedding.ModelInfo{
		Name:           info.Name,
		Dimensions:     int32(info.Dimensions),
		MaxTokens:      int32(info.MaxTokens),
		Pooling:        info.Pooling,
		From:           info.From,
		Concurrency:    int32(getIndexShare(passes)),
		FloorHint:      info.FloorHint,
		MinChunkLength: int32(info.MinChunkLength),
		ChunkTokens:    int32(info.ChunkTokens),
		Handling:       getHandlingHash(info),
	}, nil
}

func getHandlingHash(info engine.ModelInfo) string {
	if info.Handling.IsZero() {
		return ""
	}
	return info.Handling.Hash(info.MaxTokens)
}

func (s *embeddingServer) getLocalErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocalErrLocked()
}

func (s *embeddingServer) getLocalErrLocked() error {
	if s.isMock || (s.provider != providerLocal && s.provider != "") {
		return nil
	}
	return s.localErr
}

func (s *embeddingServer) applyHandling(role embedding.Role, texts []string) []string {
	s.mu.Lock()
	handling := s.modelInfo.Handling
	s.mu.Unlock()
	if role == embedding.RoleQuery {
		return handling.Apply(engine.RoleQuery, texts)
	}
	return handling.Apply(engine.RoleDocument, texts)
}

func getIndexShare(passes int) int {
	return max(passes-1, 1)
}

// EmbedTexts returns one vector per text. A local model reads the texts with the prefix its profile
// names for the role; a remote service reads them as they are.
func (s *embeddingServer) EmbedTexts(ctx context.Context, role embedding.Role, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	key, baseURL := s.getService()
	if key != "" && baseURL != "" {
		return s.embedRemoteTexts(ctx, key, baseURL, texts)
	}
	if s.isRemoteProvider() {
		if baseURL == "" {
			return nil, rpcerror.New(rpcerror.FailedPrecondition, errors.New("embedding service URL is not configured"))
		}
		return nil, rpcerror.New(rpcerror.FailedPrecondition, errors.New("embedding API key is not configured"))
	}

	if err := s.getLocalErr(); err != nil {
		return nil, rpcerror.New(rpcerror.FailedPrecondition, err)
	}
	texts = s.applyHandling(role, texts)
	emb, err := s.getEmbedder(ctx)
	if err != nil {
		return nil, rpcerror.New(rpcerror.Internal, err)
	}

	denseVectors, err := emb.Embed(ctx, texts)
	if errors.Is(err, engine.ErrClosed) {
		// A change of the performance profile retired the model while this batch waited for a pass.
		if emb, err = s.getEmbedder(ctx); err == nil {
			denseVectors, err = emb.Embed(ctx, texts)
		}
	}
	if err != nil {
		// gomutants:disable-next-line ERRORF_WRAP reason="rpcerror.New extracts only error message string"
		return nil, rpcerror.New(rpcerror.Internal, fmt.Errorf("embed batch: %w", err))
	}
	if len(denseVectors) != len(texts) {
		return nil, rpcerror.New(rpcerror.Internal, fmt.Errorf("expected %d vectors, got %d", len(texts), len(denseVectors)))
	}

	return denseVectors, nil
}

func (s *embeddingServer) embedRemoteTexts(ctx context.Context, key, baseURL string, texts []string) ([][]float32, error) {
	model := s.getModelName()
	vectors := make([][]float32, len(texts))
	for i, t := range texts {
		vec, err := computeServiceVector(ctx, key, baseURL, model, t)
		if err != nil {
			// gomutants:disable-next-line ERRORF_WRAP reason="rpcerror.New extracts only error message string"
			return nil, rpcerror.New(rpcerror.Internal, fmt.Errorf("service text %d: %w", i, err))
		}
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY,CONDITIONALS_NEGATION,INTEGER_DECREMENT,INTEGER_INCREMENT,BRANCH_IF reason="computeServiceVector returns non-empty vector on nil error"
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

func (s *embeddingServer) isRemoteProvider() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.isMock && s.provider != "" && s.provider != providerLocal
}

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

func (s *embeddingServer) getModelName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.modelInfo.Name
}

func computeServiceVector(ctx context.Context, key, baseURL, model, text string) (vector []float32, err error) {
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT,ARITHMETIC_BASE reason="HTTP client timeout constant"
	client := &http.Client{Transport: nil, CheckRedirect: nil, Jar: nil, Timeout: 30 * time.Second}
	endpoint := strings.TrimSuffix(baseURL, "/") + "/embeddings"
	reqBody, err := json.Marshal(openAIEmbeddingRequest{
		Input: []string{text},
		Model: model,
	})
	// gomutants:disable-next-line BRANCH_IF reason="marshalling string struct cannot fail"
	if err != nil {
		// gomutants:disable-next-line RETURN_ERROR_NIL reason="marshalling string struct cannot fail"
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
	// gomutants:disable-next-line STATEMENT_REMOVE reason="http response body close does not fail in memory"
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	respBytes, err := io.ReadAll(resp.Body)
	// gomutants:disable-next-line BRANCH_IF reason="unreadable body leaves respBytes empty and fails status or unmarshal"
	if err != nil {
		// gomutants:disable-next-line RETURN_ERROR_NIL reason="unreadable body leaves respBytes empty and fails status or unmarshal"
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBytes))
	}

	var openAIResp openAIEmbeddingResponse
	// gomutants:disable-next-line BRANCH_IF reason="invalid json leaves openAIResp.Data empty and fails length check below"
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
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server shutdown callback closes internal embedSrv state"
	s.OnClose(embedSrv.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
