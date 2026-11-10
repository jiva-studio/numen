// Package service is the backend of the embeddings plugin.
package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/chunking"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/limits"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/loading"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/providers"
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
	loader      *loading.Loader[modelSpec]
	apiKey      string
	baseURL     string
	provider    string
	localName   string
	remoteModel string

	profile      performance.Profile
	isProfileSet bool
	lowerNice    func(nice int) error
}

// openEmbedder loads the model a server embeds with.
type openEmbedder func(ctx context.Context, info engine.ModelInfo, local engine.LocalModel) (loading.Embedder, error)

// modelSpec is the model one load opens, with the function that opens it.
type modelSpec struct {
	open  openEmbedder
	info  engine.ModelInfo
	local engine.LocalModel
}

func (s *embeddingServer) newLoader() *loading.Loader[modelSpec] {
	return loading.New(&s.mu, loading.Ports[modelSpec]{
		Open: func(ctx context.Context, spec modelSpec) (loading.Embedder, error) {
			return spec.open(ctx, spec.info, spec.local)
		},
		LowerNice: func(nice int) error { return s.lowerNice(nice) },
		Report:    console.Report,
		Opened:    func(emb loading.Embedder) { s.modelInfo.Concurrency = emb.Concurrency() },
	})
}

func (s *embeddingServer) getEmbedder(ctx context.Context) (loading.Embedder, error) {
	return s.loader.Get(ctx, s.getStartLocked)
}

func (s *embeddingServer) getStartLocked() loading.Start[modelSpec] {
	info := s.modelInfo
	if !s.isMock {
		info = s.localInfo
	}
	local := s.localModel
	run := s.getLimitsLocked()
	if local.Threads == 0 {
		local.Threads = run.Threads
	}
	local.BatchTexts = run.BatchTexts
	return loading.Start[modelSpec]{Spec: modelSpec{open: s.open, info: info, local: local}, Nice: run.Nice}
}

func (s *embeddingServer) getLimitsLocked() limits.Run {
	return limits.GetRun(s.profile, runtime.NumCPU(), engine.GetDefaultBatchTexts())
}

func (s *embeddingServer) applyProfileLocked(profile performance.Profile) {
	if s.isProfileSet && profile == s.profile {
		return
	}
	before := s.getLimitsLocked()
	s.profile, s.isProfileSet = profile, true
	run := s.getLimitsLocked()
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr log on priority error has no side effect"
	if err := s.lowerNice(run.Nice); err != nil {
		console.Report("set the priority of the embeddings plugin", err)
	}
	if s.isMock {
		return
	}
	runtime.GOMAXPROCS(run.Threads)
	if before != run {
		s.loader.ResetLocked()
	}
}

func (s *embeddingServer) openMockModel(ctx context.Context, _ engine.ModelInfo, _ engine.LocalModel) (loading.Embedder, error) {
	if err := environment.WaitMockOpenHold(ctx); err != nil {
		return nil, err
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
	if providers.IsRemote(s.provider) && s.remoteModel == "" {
		s.remoteModel = providers.GetDefaultModel(s.provider)
	}
	s.modelInfo.MinChunkLength = chunking.ReadMinLength(values)
	if _, ok := values[chunking.TokensKey]; ok {
		s.chunkTokens = chunking.ReadTokens(values)
	}
	if list, carried := server.ReadSecrets(secrets, server.SecretAPIKey); carried {
		s.apiKey = getFirstValue(list)
	}
	s.baseURL = values["baseUrl"]
	if !providers.IsRemote(s.provider) {
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
	s.modelInfo.ChunkTokens = chunking.ClampTokens(s.chunkTokens, s.modelInfo.MaxTokens)
}

func (s *embeddingServer) refreshModelIdentityLocked() {
	if s.isMock {
		return
	}
	keep := s.modelInfo
	if s.provider != providers.Local && s.remoteModel != "" {
		profile := s.profiles.GetRemote(s.remoteModel)
		s.modelInfo = engine.ModelInfo{
			Name: s.remoteModel, Dimensions: profile.Dimensions, MaxTokens: profile.GetTokens(), Pooling: profile.Pooling,
			From: keep.From, Concurrency: keep.Concurrency, FloorHint: profile.FloorHint, MinChunkLength: keep.MinChunkLength,
			ChunkTokens: chunking.ClampTokens(s.chunkTokens, profile.GetTokens()),
			Handling:    profile.GetHandling(),
		}
		applyInfoOverrides(&s.modelInfo, environment.GetOverrides())
		return
	}
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
		s.loader.ResetLocked()
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
	overrides := environment.GetOverrides()
	if model == "" {
		model = overrides.Model
	}
	if model == "" || s.profiles.IsRemote(model) {
		model = engine.DefaultModel
	}
	profile, err := s.profiles.GetLocal(model)
	if err != nil {
		return engine.ModelInfo{}, engine.LocalModel{}, err
	}
	info, local := engine.NewConfig(profile)
	applyInfoOverrides(&info, overrides)
	applyLocalOverrides(&local, overrides)
	return info, local, nil
}

func applyInfoOverrides(info *engine.ModelInfo, overrides environment.Overrides) {
	if overrides.Dimensions > 0 {
		info.Dimensions = overrides.Dimensions
	}
	if overrides.MaxTokens > 0 {
		info.MaxTokens = overrides.MaxTokens
	}
	if overrides.Pooling != "" {
		info.Pooling = overrides.Pooling
	}
}

func applyLocalOverrides(local *engine.LocalModel, overrides environment.Overrides) {
	if overrides.Repository != "" {
		*local = local.WithRepository(overrides.Repository)
	}
	if overrides.Dir != "" {
		local.Dir = overrides.Dir
	}
	if overrides.File != "" {
		local.File = overrides.File
	}
	if overrides.Runtime != "" {
		local.Runtime = overrides.Runtime
	}
	if overrides.Threads > 0 {
		local.Threads = overrides.Threads
	}
}

func getProfilesDir() string {
	dir, err := cache.GetDataDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, engine.ProfilesFolder)
}

// Close ends the load of the model and closes the loaded model.
func (s *embeddingServer) Close() error {
	return s.loader.Close()
}
