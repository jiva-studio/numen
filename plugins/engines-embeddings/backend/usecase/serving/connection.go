package serving

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/chunking"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

// profileFetchLimit bounds one read of the profile from the host, which may start the provider and
// measure the width of the model with one request.
const profileFetchLimit = time.Minute

// profileFetch is one read of the profile. It runs on its own, so a caller that stops waiting leaves it
// running for the next call.
type profileFetch struct {
	finished chan struct{}
	cancel   context.CancelFunc
	profile  model.ConnectionProfile
	err      error
}

// connectionSource reads the profile of the connection that answers for the plugin and keeps it until
// the settings change.
type connectionSource struct {
	open ports.OpenModels

	mu      sync.Mutex
	models  ports.Models
	fetch   *profileFetch
	profile *model.ConnectionProfile
}

func newConnectionSource(open ports.OpenModels) *connectionSource {
	return &connectionSource{open: open, mu: sync.Mutex{}, models: nil, fetch: nil, profile: nil}
}

func (c *connectionSource) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fetch != nil {
		c.fetch.cancel()
		c.fetch = nil
	}
	c.profile = nil
}

func (c *connectionSource) getModels() (ports.Models, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.models != nil {
		return c.models, nil
	}
	if c.open == nil {
		return nil, errors.New("the model API of the host is not available")
	}
	models, err := c.open()
	if err != nil {
		return nil, err
	}
	c.models = models
	return models, nil
}

func (c *connectionSource) getProfile(ctx context.Context) (model.ConnectionProfile, ports.Models, error) {
	models, err := c.getModels()
	if err != nil {
		return model.ConnectionProfile{}, nil, PreconditionError{Err: err}
	}
	c.mu.Lock()
	if cached := c.profile; cached != nil {
		c.mu.Unlock()
		return *cached, models, nil
	}
	fetch := c.fetch
	if fetch == nil {
		fetch = c.startFetchLocked(ctx, models)
	}
	c.mu.Unlock()
	select {
	case <-fetch.finished:
		if fetch.err != nil {
			return model.ConnectionProfile{}, nil, readConnectionError(fetch.err)
		}
		return fetch.profile, models, nil
	case <-ctx.Done():
		return model.ConnectionProfile{}, nil, ctx.Err()
	}
}

func (c *connectionSource) startFetchLocked(caller context.Context, models ports.Models) *profileFetch {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(caller), profileFetchLimit)
	var none model.ConnectionProfile
	fetch := &profileFetch{finished: make(chan struct{}), cancel: cancel, profile: none, err: nil}
	c.fetch = fetch
	go func() {
		defer cancel()
		profile, err := models.GetEmbeddingProfile(ctx)
		c.mu.Lock()
		defer c.mu.Unlock()
		fetch.profile, fetch.err = profile, err
		if c.fetch == fetch {
			c.fetch = nil
			if err == nil {
				c.profile = &profile
			}
		}
		close(fetch.finished)
	}()
	return fetch
}

func readConnectionError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if failure, isConnection := errors.AsType[ports.ConnectionError](err); isConnection {
		return PreconditionError{Err: failure}
	}
	return err
}

func (s *Server) getConnectionModelInfo(ctx context.Context) (ModelInfo, error) {
	profile, _, err := s.connection.getProfile(ctx)
	if err != nil {
		return ModelInfo{}, err
	}
	s.mu.Lock()
	minChunkLength, chunkTokens, concurrency := s.modelInfo.MinChunkLength, s.chunkTokens, s.modelInfo.Concurrency
	s.mu.Unlock()
	if concurrency <= 0 {
		concurrency = defaultPasses
	}
	return ModelInfo{
		Name:           profile.Model,
		Dimensions:     profile.Dimensions,
		MaxTokens:      profile.MaxTokens,
		Pooling:        profile.Pooling,
		From:           profile.Source,
		Concurrency:    limits.GetIndexShare(concurrency),
		FloorHint:      0,
		MinChunkLength: minChunkLength,
		ChunkTokens:    chunking.ClampTokens(chunkTokens, profile.MaxTokens),
		Handling:       profile.GetHandlingHash(),
	}, nil
}

func (s *Server) embedThroughConnection(ctx context.Context, role model.Role, texts []string) ([][]float32, error) {
	profile, models, err := s.connection.getProfile(ctx)
	if err != nil {
		return nil, err
	}
	vectors, err := models.Embed(ctx, role, profile.GetHandling().Apply(role, texts))
	if err != nil {
		return nil, readConnectionError(err)
	}
	return checkVectors(vectors, len(texts), profile.Dimensions)
}

func checkVectors(vectors [][]float32, count, width int) ([][]float32, error) {
	if len(vectors) != count {
		return nil, fmt.Errorf("expected %d vectors, got %d", count, len(vectors))
	}
	for _, one := range vectors {
		if len(one) != width {
			return nil, fmt.Errorf("the connection returned a vector of %d dimensions, its profile says %d", len(one), width)
		}
	}
	return vectors, nil
}
