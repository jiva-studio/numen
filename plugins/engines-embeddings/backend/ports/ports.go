// Package ports declares what the use cases of the embeddings plugin ask of the machine, the model
// and the host.
package ports

import (
	"context"
	"errors"

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
// error that kept it from reading the shipped ones. A user file that cannot be read is skipped and
// handed to report.
type ProfileSource interface {
	Load(report func(file string, err error)) (*model.Profiles, error)
}

// Environment is what the process environment sets for the model.
type Environment interface {
	GetOverrides() model.Overrides
	IsMockModel() bool
	GetMockLogPath() string
	WaitMockOpenHold(ctx context.Context) error
}

// OpenEmbedder loads the model a server embeds with.
type OpenEmbedder func(ctx context.Context, info model.ModelInfo, local model.LocalModel) (Embedder, error)

// NewMockEmbedder returns the stand-in for the model that appends the texts it is given to the file at logPath, when logPath is not empty.
type NewMockEmbedder func(logPath string) Embedder

// Models is the model API of the host: the connections the plugin holds decide which service answers.
type Models interface {
	// Embed returns one vector per text. A query is embedded as a person's action and a document in the
	// background.
	Embed(ctx context.Context, role model.Role, texts []string) ([][]float32, error)
	GetEmbeddingProfile(ctx context.Context) (model.ConnectionProfile, error)
}

// OpenModels connects to the model API of the host.
type OpenModels func() (Models, error)

// ErrNoConnection is the failure of a call while the settings of the plugin name no connection. Its text is
// what a person reads in the notice of the indexing.
var ErrNoConnection = errors.New("choose a connection in the settings of Vector Embeddings")

// ConnectionError is the failure of a call that no connection answered: the plugin has none, every
// one failed, or one refused the request. Its text names each connection and the reason.
type ConnectionError struct {
	Err error
}

// Error returns the text that names the connections and the reasons.
func (e ConnectionError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the cause of the failure.
func (e ConnectionError) Unwrap() error {
	return e.Err
}
