// Package catalog keeps the models the Antigravity program lists, folded by reasoning level, and
// chooses the listed model that a model and a level of a row name.
package catalog

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/domain/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
)

const (
	modelListLimit = 15 * time.Second
	failureBackoff = 30 * time.Second
)

// Lister lists the models of the program that command starts.
type Lister interface {
	List(ctx context.Context, command []string) ([]model.Model, error)
}

// Catalog keeps the last list of models each program gave.
type Catalog struct {
	lister Lister
	models *cache.Cache[[]model.Model]
}

// New returns a catalog. With isAsync, a list that is not cached is loaded in the background.
func New(lister Lister, isAsync bool) *Catalog {
	return &Catalog{
		lister: lister,
		models: cache.New[[]model.Model](
			"models",
			cache.WithTimeout(modelListLimit),
			cache.WithFailureBackoff(failureBackoff),
			cache.WithAsync(isAsync),
		),
	}
}

// List returns the models the program lists, each family of levels as one model. A failed load leaves
// the last list of the cache, or none.
func (c *Catalog) List(ctx context.Context, program string) []reasoning.Family {
	list := func(ctx context.Context) ([]model.Model, error) { return c.lister.List(ctx, []string{program}) }
	items, _, _ := c.models.Get(ctx, program, list)
	return model.Fold(items)
}

// ResolveModel returns the listed model that the model name and the reasoning level choose. The default
// model and a name that is no family stay as they are.
func (c *Catalog) ResolveModel(ctx context.Context, program, modelName, level string) string {
	if modelName == "" || modelName == model.DefaultName {
		return modelName
	}
	return reasoning.ResolveModel(c.List(ctx, program), modelName, level)
}
