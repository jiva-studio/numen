package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

// The values of the setting `source`.
const (
	sourceLocal      = "local"
	sourceConnection = "connection"
)

// modelClient is the model API of the host: the rows of the setting `connections` decide which
// service answers.
type modelClient interface {
	Embed(ctx context.Context, request connection.EmbedRequest) (connection.EmbedResult, error)
	GetEmbeddingProfile(ctx context.Context) (connection.EmbeddingProfile, error)
}

// newModelClient creates the client of the host the plugin process was started for.
func newModelClient() (modelClient, error) {
	return connection.NewClientFromEnv()
}

func readSource(values map[string]string) string {
	if values["source"] == sourceConnection {
		return sourceConnection
	}
	return sourceLocal
}

func (s *embeddingServer) isConnectionLocked() bool {
	return s.source == sourceConnection
}

// profileFetchLimit bounds one read of the profile from the host, which may start the provider and
// measure the width of the model with one request.
const profileFetchLimit = time.Minute

// profileFetch is one read of the profile. It runs on its own, so a caller that stops waiting leaves it
// running for the next call.
type profileFetch struct {
	finished    chan struct{}
	cancel  context.CancelFunc
	profile connection.EmbeddingProfile
	err     error
}

// resetConnectionLocked forgets the profile and ends the read of it, so the next call asks again.
func (s *embeddingServer) resetConnectionLocked() {
	if s.fetch != nil {
		s.fetch.cancel()
		s.fetch = nil
	}
	s.connectionProfile = nil
}

// getConnectionProfile returns the profile of the embedding model of the connection. The host reads it
// from the provider once for a configuration of the plugin; a failed read is not kept.
func (s *embeddingServer) getConnectionProfile(ctx context.Context) (connection.EmbeddingProfile, modelClient, error) {
	models, err := s.getModelClient()
	if err != nil {
		return connection.EmbeddingProfile{}, nil, rpcerror.New(rpcerror.FailedPrecondition, err)
	}
	s.mu.Lock()
	if cached := s.connectionProfile; cached != nil {
		s.mu.Unlock()
		return *cached, models, nil
	}
	fetch := s.fetch
	if fetch == nil {
		fetch = s.startFetchLocked(ctx, models)
	}
	s.mu.Unlock()
	select {
	case <-fetch.finished:
		if fetch.err != nil {
			return connection.EmbeddingProfile{}, nil, readConnectionError(fetch.err)
		}
		return fetch.profile, models, nil
	case <-ctx.Done():
		return connection.EmbeddingProfile{}, nil, ctx.Err()
	}
}

func (s *embeddingServer) startFetchLocked(caller context.Context, models modelClient) *profileFetch {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(caller), profileFetchLimit)
	fetch := &profileFetch{finished: make(chan struct{}), cancel: cancel}
	s.fetch = fetch
	go func() {
		defer cancel()
		profile, err := models.GetEmbeddingProfile(ctx)
		s.mu.Lock()
		defer s.mu.Unlock()
		fetch.profile, fetch.err = profile, err
		if s.fetch == fetch {
			s.fetch = nil
			if err == nil {
				s.connectionProfile = &profile
			}
		}
		close(fetch.finished)
	}()
	return fetch
}

func (s *embeddingServer) getModelClient() (modelClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.models != nil {
		return s.models, nil
	}
	models, err := s.newModels()
	if err != nil {
		return nil, err
	}
	s.models = models
	return models, nil
}

// readConnectionError keeps the cancellation of a call as it is and gives every failure of the rows the
// code of a setting that does not allow the call; the text names each connection and its reason.
func readConnectionError(err error) error {
	var failed *connection.AllFailedError
	var rejected *connection.RejectedError
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, connection.ErrNoConnection), errors.As(err, &failed), errors.As(err, &rejected):
		return rpcerror.New(rpcerror.FailedPrecondition, err)
	}
	return rpcerror.New(rpcerror.Internal, err)
}

func (s *embeddingServer) getConnectionModelInfo(ctx context.Context) (embedding.ModelInfo, error) {
	profile, _, err := s.getConnectionProfile(ctx)
	if err != nil {
		return embedding.ModelInfo{}, err
	}
	s.mu.Lock()
	minChunkLength, chunkTokens, concurrency := s.modelInfo.MinChunkLength, s.chunkTokens, s.modelInfo.Concurrency
	s.mu.Unlock()
	if concurrency <= 0 {
		concurrency = defaultPasses
	}
	return embedding.ModelInfo{
		Name:           profile.Model,
		Dimensions:     int32(profile.Dimensions),
		MaxTokens:      int32(profile.MaxTokens),
		Pooling:        profile.Pooling,
		From:           profile.Source,
		Concurrency:    int32(getIndexShare(concurrency)),
		MinChunkLength: int32(minChunkLength),
		ChunkTokens:    int32(ClampChunkTokens(chunkTokens, profile.MaxTokens)),
		Handling:       getHandlingHash(connectionInfo(profile)),
	}, nil
}

func connectionInfo(profile connection.EmbeddingProfile) engine.ModelInfo {
	return engine.ModelInfo{
		MaxTokens: profile.MaxTokens,
		Handling:  engine.Handling{QueryPrefix: profile.QueryPrefix, DocumentPrefix: profile.DocumentPrefix},
	}
}

// embedThroughConnection embeds the texts with the prefix the profile names for the role. A person's
// question goes ahead of the indexing in the queue of the connection.
func (s *embeddingServer) embedThroughConnection(ctx context.Context, role embedding.Role, texts []string) ([][]float32, error) {
	profile, models, err := s.getConnectionProfile(ctx)
	if err != nil {
		return nil, err
	}
	handling := connectionInfo(profile).Handling
	engineRole, priority := engine.RoleDocument, connection.PriorityBackground
	if role == embedding.RoleQuery {
		engineRole, priority = engine.RoleQuery, connection.PriorityInteractive
	}
	result, err := models.Embed(ctx, connection.EmbedRequest{Texts: handling.Apply(engineRole, texts), Priority: priority})
	if err != nil {
		return nil, readConnectionError(err)
	}
	return checkVectors(result.Vectors, len(texts), profile.Dimensions)
}

func checkVectors(vectors [][]float32, count, width int) ([][]float32, error) {
	if len(vectors) != count {
		return nil, rpcerror.New(rpcerror.Internal, fmt.Errorf("expected %d vectors, got %d", count, len(vectors)))
	}
	for _, one := range vectors {
		if len(one) != width {
			return nil, rpcerror.New(rpcerror.Internal, fmt.Errorf("the connection returned a vector of %d dimensions, its profile says %d", len(one), width))
		}
	}
	return vectors, nil
}
