package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/onnx"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/openai"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
)

// Embedders are what makes the vectors a vault is searched by and what makes
// the vector a question is asked with.
//
// They are one model where the settings name one placement. Naming a placement
// for questions is what puts a vault indexed over a network within reach of a
// machine that has none.
//
// A model on this machine is fetched and compiled behind this, so a person who
// has just installed the application has a window rather than a wait. Until it
// is here the words answer alone, and what it is — the model, its width — is
// known from the settings all the same.
//
// An embedder is optional: an installation with none answers with nothing, and
// a placement that cannot be built — no key, no base URL — is that same
// absence, carrying the reason with it.
func (c Config) Embedders(tasks *task.Tasks) (indexing, asking port.Embedder, close func() error, why error) {
	first, why := c.placed(c.Embedding.Indexing, tasks)
	if why != nil || first == nil {
		return nil, nil, nil, why
	}
	if c.Embedding.Query.Use == "" {
		return first.Filling(), first.Asking(), first.Close, nil
	}

	second, why := c.placed(c.Embedding.Query, tasks)
	if why != nil {
		_ = first.Close()
		return nil, nil, nil, why
	}
	if second == nil {
		return first.Filling(), nil, first.Close, nil
	}
	// Two placements are asked whether they are one model, once both are here.
	// Nothing in a settings file shows it, and a question embedded in another
	// space finds nothing however well it is written.
	go func() {
		held := context.Background()
		if first.Wait(held) != nil || second.Wait(held) != nil {
			return
		}
		if err := embedding.Agree(held, first.Filling(), second.Filling()); err != nil {
			second.Disown(err)
			failed(tasks, c.Embedding.Query.Local.Name, err)
		}
	}()
	return first.Filling(), second.Asking(), both(first.Close, second.Close), nil
}

// Embedder is what makes the vectors a vault is searched by, waited for. A run
// with nowhere to show that a model is arriving waits for it instead.
func (c Config) Embedder() (port.Embedder, func() error, error) {
	switch where := c.Embedding.Indexing; where.Use {
	case embed.UseService:
		client, err := openai.New(c.Embedding.Model, where.Service)
		if err != nil {
			return nil, nil, err
		}
		return client, func() error { return nil }, nil

	case embed.UseLocal:
		model, err := onnx.Open(c.Embedding.Model, where.Local, nil)
		if err != nil {
			return nil, nil, err
		}
		return model, model.Close, nil
	}
	return nil, nil, nil
}

// placed is what one placement makes: a service, which answers at once, or a
// model on this machine, which is loaded behind the window. Nothing for a
// placement that names neither.
func (c Config) placed(where embed.Placement, tasks *task.Tasks) (*Embedding, error) {
	switch where.Use {
	case embed.UseService:
		client, err := openai.New(c.Embedding.Model, where.Service)
		if err != nil {
			return nil, err
		}
		held := Arriving(client.Model())
		held.Landed(client, nil)
		return held, nil

	case embed.UseLocal:
		is := c.Embedding.Model
		held := Arriving(port.EmbeddingModel{
			Name: is.Name, Dimensions: is.Dimensions, MaxTokens: is.MaxTokens, Pooling: is.Pooling,
		})
		doing := preparing(tasks, where.Local.Name)
		doing(0, 0)
		go func() {
			model, err := onnx.Open(is, where.Local, doing)
			if err != nil {
				held.Landed(nil, err)
				failed(tasks, where.Local.Name, err)
				return
			}
			held.Landed(model, nil)
			ready(tasks, where.Local.Name)
		}()
		return held, nil
	}
	return nil, nil
}

// What the arrival of a model is called in the list of what is being done. Two
// placements are two models, so the name is part of what it is called.
func gettingReady(name string) string { return "getting ready: " + name }

// preparing tells the list how far the model has got. Fetching it and compiling
// it are one wait to whoever is watching, and the count is the bytes of it that
// are here.
func preparing(tasks *task.Tasks, name string) onnx.Fetching {
	if tasks == nil {
		return nil
	}
	return func(done, total int64) {
		tasks.Set(task.Task{
			ID: gettingReady(name), Doing: "Preparing the model", About: name,
			Done: done, Total: total,
		})
	}
}

func ready(tasks *task.Tasks, name string) {
	if tasks != nil {
		tasks.Done(gettingReady(name))
	}
}

// failed leaves the model in the list under what stopped it. A vault that will
// never be searched by meaning is not a vault that is quietly searched by
// words.
func failed(tasks *task.Tasks, name string, why error) {
	if tasks != nil {
		tasks.Set(task.Task{
			ID: gettingReady(name), Doing: "Preparing the model", About: name,
			Failed: why.Error(),
		})
	}
}

// both is one closer for two, letting go of the second whatever the first says.
func both(first, second func() error) func() error {
	return func() error {
		var why error
		if second != nil {
			why = second()
		}
		if first != nil {
			if err := first(); why == nil {
				why = err
			}
		}
		return why
	}
}
