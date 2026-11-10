package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/remote"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/limits"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/providers"
)

// defaultPasses is the passes a model is taken to run when it names none.
const defaultPasses = 4

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
		passes = defaultPasses
	}
	return embedding.ModelInfo{
		Name:           info.Name,
		Dimensions:     int32(info.Dimensions),
		MaxTokens:      int32(info.MaxTokens),
		Pooling:        info.Pooling,
		From:           info.From,
		Concurrency:    int32(limits.GetIndexShare(passes)),
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
	if s.isMock || providers.IsRemote(s.provider) {
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
	vectors, err := remote.FetchVectors(ctx, key, baseURL, s.getModelName(), texts)
	if err != nil {
		// gomutants:disable-next-line ERRORF_WRAP reason="rpcerror.New extracts only error message string"
		return nil, rpcerror.New(rpcerror.Internal, err)
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,CONDITIONALS_NEGATION,INTEGER_DECREMENT,INTEGER_INCREMENT,BRANCH_IF reason="a vector fetched without an error is not empty"
	if dimensions := len(vectors[len(vectors)-1]); dimensions > 0 {
		s.mu.Lock()
		s.modelInfo.Dimensions = dimensions
		s.mu.Unlock()
	}
	return vectors, nil
}

func (s *embeddingServer) isRemoteProvider() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.isMock && providers.IsRemote(s.provider)
}

func (s *embeddingServer) getService() (key, baseURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isMock || s.provider == providers.Local {
		return "", ""
	}
	if hosted := providers.GetServiceURL(s.provider); hosted != "" {
		return s.apiKey, hosted
	}
	baseURL = s.baseURL
	if baseURL == "" {
		baseURL = environment.GetServiceURL()
	}
	return s.apiKey, baseURL
}

func (s *embeddingServer) getModelName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.modelInfo.Name
}
