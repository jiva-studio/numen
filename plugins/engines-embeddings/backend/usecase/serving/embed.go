package serving

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

// defaultPasses is the passes a model is taken to run when it names none.
const defaultPasses = 4

// PreconditionError is the failure of a call that the settings do not allow. Any other error of the
// server is a fault.
type PreconditionError struct {
	Err error
}

// Error returns the message of the failed precondition.
func (e PreconditionError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the cause of the failed precondition.
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

// GetRecipe is everything about the model that decides what a vector is, as one value: the provider, the
// name, the width, the token limit, the pooling and the handling. Two vectors of different recipes cannot be
// compared. The provider comes first: two places serving one name are two sets of vectors.
func (m ModelInfo) GetRecipe() string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s", m.From, m.Name, m.Dimensions, m.MaxTokens, m.Pooling, m.Handling)
}

// GetModelInfo returns the model the server serves. It fails with a PreconditionError while the
// local model cannot be resolved, and while no connection answers for the model of the connections.
func (s *Server) GetModelInfo(ctx context.Context) (ModelInfo, error) {
	s.mu.Lock()
	if s.isConnectionLocked() {
		s.mu.Unlock()
		return s.getConnectionModelInfo(ctx)
	}
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

func getHandlingHash(info model.ModelInfo) string {
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
	if s.isMock || s.isConnectionLocked() {
		return nil
	}
	return s.localErr
}

func (s *Server) applyHandling(role model.Role, texts []string) []string {
	s.mu.Lock()
	handling := s.modelInfo.Handling
	s.mu.Unlock()
	return handling.Apply(role, texts)
}

// EmbedTexts returns one vector per text, made for the role. A local model reads the texts with the
// prefix its profile names for the role; a connection is given the prefixes of its profile. A call the
// settings do not allow fails with a PreconditionError.
func (s *Server) EmbedTexts(ctx context.Context, role model.Role, texts []string) ([][]float32, error) {
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
		return nil, PreconditionError{Err: err}
	}
	texts = s.applyHandling(role, texts)
	emb, err := s.getEmbedder(ctx)
	if err != nil {
		return nil, err
	}

	denseVectors, err := emb.Embed(ctx, texts)
	if errors.Is(err, ports.ErrClosed) {
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
