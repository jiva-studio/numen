// Package service is the backend of the embeddings plugin.
package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/performance"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/onnx"
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

	mu        sync.Mutex
	embedder  textEmbedder
	localName string
	source    string

	fetch             *profileFetch
	connectionProfile *connection.EmbeddingProfile
	models            modelClient
	newModels         func() (modelClient, error)

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

func (s *embeddingServer) updateConfig(values map[string]string, _ server.Secrets) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyProfileLocked(performance.Read(values))
	s.loadProfilesLocked()
	s.source = readSource(values)
	s.resetConnectionLocked()
	s.modelInfo.MinChunkLength = readMinChunkLength(values)
	if _, ok := values[settingChunkTokens]; ok {
		s.chunkTokens = ReadChunkTokens(values)
	}
	if !s.isConnectionLocked() {
		s.updateLocalModelLocked(values["model"])
	}
	s.refreshModelInfoLocked()
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
	if s.isConnectionLocked() {
		return
	}
	keep := s.modelInfo
	s.modelInfo = s.localInfo
	s.modelInfo.Concurrency = keep.Concurrency
	s.modelInfo.MinChunkLength = keep.MinChunkLength
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
	if model == "" {
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
			localName:   info.Name,
			source:      sourceLocal, fetch: nil, connectionProfile: nil, models: nil, newModels: newModelClient,
			profile: performance.Balanced, isProfileSet: false, lowerNice: performance.SetNice, opening: nil,
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
		source:      sourceLocal, fetch: nil, connectionProfile: nil, models: nil, newModels: newModelClient,
		profile: performance.Balanced, isProfileSet: false, lowerNice: performance.SetNice, opening: nil,
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

func (s *embeddingServer) GetModelInfo(ctx context.Context) (embedding.ModelInfo, error) {
	s.mu.Lock()
	if s.isConnectionLocked() {
		s.mu.Unlock()
		return s.getConnectionModelInfo(ctx)
	}
	info := s.modelInfo
	localErr := s.getLocalErrLocked()
	s.mu.Unlock()
	if localErr != nil {
		return embedding.ModelInfo{}, rpcerror.New(rpcerror.FailedPrecondition, localErr)
	}
	passes := info.Concurrency
	if passes <= 0 {
		passes = defaultPasses
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
	if s.isMock || s.isConnectionLocked() {
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

// defaultPasses is the number of batches a model runs in parallel when it reports none.
const defaultPasses = 4

func getIndexShare(passes int) int {
	return max(passes-1, 1)
}

// EmbedTexts returns one vector per text, made for the role. A local model reads the texts with the
// prefix its profile names for the role; a connection is given the prefixes of its profile.
func (s *embeddingServer) EmbedTexts(ctx context.Context, role embedding.Role, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	s.mu.Lock()
	isConnection := s.isConnectionLocked()
	s.mu.Unlock()
	if isConnection {
		return s.embedThroughConnection(ctx, role, texts)
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

// Mount registers the embedding service on s, with the model the environment configures. The
// embedder closes with s. The model API of the host is the one the environment names.
func Mount(s *server.Server) error {
	return mount(s, newModelClient)
}

func mount(s *server.Server, newModels func() (modelClient, error)) error {
	embedSrv := newEmbeddingServer()
	embedSrv.newModels = newModels
	s.SetUpdateConfig(embedSrv.updateConfig)
	embedding.Mount(s, embedSrv)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server shutdown callback closes internal embedSrv state"
	s.OnClose(embedSrv.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. The model API of the host is the one
// those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, func() (modelClient, error) { return connection.NewClientFromVars(vars) })
	})
}
