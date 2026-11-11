// Package ports declares what the use cases of the embeddings plugin ask of the machine, the model
// and the remote services.
package ports

import (
	"context"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

// Embedder is a model that turns texts into vectors.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Concurrency() int
	Close() error
}

// Machine is the process the server runs in.
type Machine interface {
	CountCPUs() int
	SetMaxProcs(count int)
	LowerNice(nice int) error
}

// ProfileSource reads the model profiles. It returns the profiles it could read together with the
// error that kept it from reading the shipped ones.
type ProfileSource interface {
	Load() (*model.Profiles, error)
}

// Environment is what the process environment sets for the model.
type Environment interface {
	GetOverrides() model.Overrides
	GetServiceURL() string
	IsMockModel() bool
	GetMockLogPath() string
	WaitMockOpenHold(ctx context.Context) error
}

// OpenEmbedder loads the model a server embeds with.
type OpenEmbedder func(ctx context.Context, info model.ModelInfo, local model.LocalModel) (Embedder, error)

// NewMockEmbedder returns the stand-in for the model that appends the texts it is given to the file at logPath, when logPath is not empty.
type NewMockEmbedder func(logPath string) Embedder

// FetchVectors asks a remote service for one vector per text.
type FetchVectors func(ctx context.Context, key, baseURL, model string, texts []string) ([][]float32, error)
