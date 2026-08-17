package container

import (
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/onnx"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/openai"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Embedder is what turns text into vectors, chosen by this installation's
// configuration.
//
// An embedder is optional: an installation with none answers with nothing, and a
// model that cannot be opened — no file, no key, a width this build does not
// expect — is that same absence, carrying the reason with it.
//
// The closer is returned separately because a local model holds a session that
// has to be let go of, and a service holds nothing.
func (c Config) Embedder() (embedder port.Embedder, close func() error, why error) {
	cfg := c.Embedding

	switch cfg.Use {
	case embed.UseService:
		client, err := openai.New(cfg.Service)
		if err != nil {
			return nil, nil, err
		}
		return client, func() error { return nil }, nil

	case embed.UseLocal:
		model, err := onnx.Open(cfg.Local)
		if err != nil {
			return nil, nil, err
		}
		return model, model.Close, nil
	}

	return nil, nil, nil
}
