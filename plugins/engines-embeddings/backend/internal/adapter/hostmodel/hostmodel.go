// Package hostmodel asks the host for vectors through the operations llm.embed and llm.profile. The host
// picks the connection that answers.
package hostmodel

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

// Models adapts the connections of the host to the port of the use case.
type Models struct {
	caller connection.Caller
}

// New returns Models that call through caller.
func New(caller connection.Caller) Models {
	return Models{caller: caller}
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
	result, err := llm.Embed(ctx, m.caller, &llm.EmbedRequest{Texts: texts}, priority)
	if err != nil {
		return nil, readError(err)
	}
	vectors := make([][]float32, len(result.GetVectors()))
	for i, one := range result.GetVectors() {
		vectors[i] = one.GetValues()
	}
	return vectors, nil
}

func (m Models) GetEmbeddingProfile(ctx context.Context) (model.ConnectionProfile, error) {
	profile, err := llm.GetProfile(ctx, m.caller)
	if err != nil {
		return model.ConnectionProfile{}, readError(err)
	}
	return model.ConnectionProfile{
		Model: profile.GetModel(), Source: profile.GetSource(), Pooling: profile.GetPooling(),
		QueryPrefix: profile.GetQueryPrefix(), DocumentPrefix: profile.GetDocumentPrefix(),
		Dimensions: int(profile.GetWidth()), MaxTokens: int(profile.GetMaxTokens()),
	}, nil
}

func readError(err error) error {
	var failed *connection.AllFailedError
	var stopped *connection.StoppedError
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, connection.ErrNoConnection), errors.As(err, &failed), errors.As(err, &stopped):
		return ports.ConnectionError{Err: err}
	}
	return err
}
