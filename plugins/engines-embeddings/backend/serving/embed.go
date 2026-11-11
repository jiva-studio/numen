package serving

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/limits"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/providers"
)

// defaultPasses is the passes a model is taken to run when it names none.
const defaultPasses = 4

// PreconditionError is a refusal because the settings do not allow the call. Any other error of the
// server is a fault.
type PreconditionError struct {
	Err error
}

// Error returns the message of the refusal.
func (e PreconditionError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the cause of the refusal.
func (e PreconditionError) Unwrap() error {
	return e.Err
}

// ModelInfo describes the model the server serves. Concurrency is the passes the index may take.
type ModelInfo struct {
	Name           string
	Dimensions     int
	MaxTokens      int
	Pooling        string
	From           string
	Concurrency    int
	FloorHint      float64
	MinChunkLength int
	ChunkTokens    int
	Handling       string
}

// GetModelInfo returns the model the server serves. It refuses with a PreconditionError while the
// local model cannot be resolved.
func (s *Server) GetModelInfo() (ModelInfo, error) {
	s.mu.Lock()
	info := s.modelInfo
	localErr := s.getLocalErrLocked()
	s.mu.Unlock()
	if localErr != nil {
		return ModelInfo{}, PreconditionError{Err: localErr}
	}
	passes := info.Concurrency
	if passes <= 0 {
		passes = defaultPasses
	}
	return ModelInfo{
		Name:           info.Name,
		Dimensions:     info.Dimensions,
		MaxTokens:      info.MaxTokens,
		Pooling:        info.Pooling,
		From:           info.From,
		Concurrency:    limits.GetIndexShare(passes),
		FloorHint:      info.FloorHint,
		MinChunkLength: info.MinChunkLength,
		ChunkTokens:    info.ChunkTokens,
		Handling:       getHandlingHash(info),
	}, nil
}

func getHandlingHash(info engine.ModelInfo) string {
	if info.Handling.IsZero() {
		return ""
	}
	return info.Handling.Hash(info.MaxTokens)
}

func (s *Server) getLocalErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocalErrLocked()
}

func (s *Server) getLocalErrLocked() error {
	if s.isMock || providers.IsRemote(s.provider) {
		return nil
	}
	return s.localErr
}

func (s *Server) applyHandling(role engine.Role, texts []string) []string {
	s.mu.Lock()
	handling := s.modelInfo.Handling
	s.mu.Unlock()
	return handling.Apply(role, texts)
}

// EmbedTexts returns one vector per text. A local model reads the texts with the prefix its profile
// names for the role; a remote service reads them as they are. A refusal is a PreconditionError.
func (s *Server) EmbedTexts(ctx context.Context, role engine.Role, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	key, baseURL := s.getService()
	if key != "" && baseURL != "" {
		return s.embedRemoteTexts(ctx, key, baseURL, texts)
	}
	if s.isRemoteProvider() {
		if baseURL == "" {
			return nil, PreconditionError{Err: errors.New("embedding service URL is not configured")}
		}
		return nil, PreconditionError{Err: errors.New("embedding API key is not configured")}
	}

	if err := s.getLocalErr(); err != nil {
		return nil, PreconditionError{Err: err}
	}
	texts = s.applyHandling(role, texts)
	emb, err := s.getEmbedder(ctx)
	if err != nil {
		return nil, err
	}

	denseVectors, err := emb.Embed(ctx, texts)
	if errors.Is(err, engine.ErrClosed) {
		// A change of the performance profile retired the model while this batch waited for a pass.
		if emb, err = s.getEmbedder(ctx); err == nil {
			denseVectors, err = emb.Embed(ctx, texts)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("embed batch: %w", err)
	}
	if len(denseVectors) != len(texts) {
		return nil, fmt.Errorf("expected %d vectors, got %d", len(texts), len(denseVectors))
	}

	return denseVectors, nil
}

func (s *Server) embedRemoteTexts(ctx context.Context, key, baseURL string, texts []string) ([][]float32, error) {
	vectors, err := s.fetchVectors(ctx, key, baseURL, s.getModelName(), texts)
	if err != nil {
		return nil, err
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,CONDITIONALS_NEGATION,INTEGER_DECREMENT,INTEGER_INCREMENT,BRANCH_IF reason="a vector fetched without an error is not empty"
	if dimensions := len(vectors[len(vectors)-1]); dimensions > 0 {
		s.mu.Lock()
		s.modelInfo.Dimensions = dimensions
		s.mu.Unlock()
	}
	return vectors, nil
}

func (s *Server) isRemoteProvider() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.isMock && providers.IsRemote(s.provider)
}

func (s *Server) getService() (key, baseURL string) {
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
		baseURL = s.env.GetServiceURL()
	}
	return s.apiKey, baseURL
}

func (s *Server) getModelName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.modelInfo.Name
}
