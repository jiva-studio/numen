// Package hostmodel asks the model API of the host for vectors. The host picks the connection that
// answers.
package hostmodel

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

// Client is the model API of the host, as the embeddings plugin uses it.
type Client interface {
	Embed(ctx context.Context, request connection.EmbedRequest) (connection.EmbedResult, error)
	GetEmbeddingProfile(ctx context.Context) (connection.EmbeddingProfile, error)
}

// Models adapts a Client to the port of the use case.
type Models struct {
	client Client
}

// New returns Models that call client.
func New(client Client) Models {
	return Models{client: client}
}

// OpenFromEnvironment connects to the host the process environment names.
func OpenFromEnvironment() (ports.Models, error) {
	client, err := connection.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return New(client), nil
}

// OpenFromVariables returns the function that connects to the host the variables name.
func OpenFromVariables(vars map[string]string) ports.OpenModels {
	return func() (ports.Models, error) {
		client, err := connection.NewClientFromVars(vars)
		if err != nil {
			return nil, err
		}
		return New(client), nil
	}
}

// Embed returns one vector per text. A query is embedded as a person's action and a document in the
// background.
func (m Models) Embed(ctx context.Context, role model.Role, texts []string) ([][]float32, error) {
	priority := connection.PriorityBackground
	if role == model.RoleQuery {
		priority = connection.PriorityInteractive
	}
	result, err := m.client.Embed(ctx, connection.EmbedRequest{Texts: texts, Priority: priority})
	if err != nil {
		return nil, readError(err)
	}
	return result.Vectors, nil
}

func (m Models) GetEmbeddingProfile(ctx context.Context) (model.ConnectionProfile, error) {
	profile, err := m.client.GetEmbeddingProfile(ctx)
	if err != nil {
		return model.ConnectionProfile{}, readError(err)
	}
	return model.ConnectionProfile{
		Model: profile.Model, Source: profile.Source, Pooling: profile.Pooling,
		QueryPrefix: profile.QueryPrefix, DocumentPrefix: profile.DocumentPrefix,
		Dimensions: profile.Dimensions, MaxTokens: profile.MaxTokens,
	}, nil
}

func readError(err error) error {
	var failed *connection.AllFailedError
	var rejected *connection.RejectedError
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, connection.ErrNoConnection):
		return ports.ConnectionError{Err: ports.ErrNoConnection}
	case errors.As(err, &failed), errors.As(err, &rejected):
		return ports.ConnectionError{Err: err}
	}
	return err
}
