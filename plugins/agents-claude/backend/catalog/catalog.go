// Package catalog keeps the list of models the program accepts and answers which depths of
// reasoning a model supports.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/models"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
)

// Catalog lists models through a cache, one list per command line of the program.
type Catalog struct {
	lister ports.ModelLister
	lists  *cache.Cache[[]models.Model]
}

// New creates a catalog that asks lister for the models. An async catalog answers with no model
// while the first list loads, and the fallback models once that load has failed.
func New(lister ports.ModelLister, isAsync bool, extra ...cache.Option) (*Catalog, error) {
	if lister == nil {
		return nil, errors.New("catalog: the model lister is required")
	}
	return &Catalog{
		lister: lister,
		lists: cache.New[[]models.Model](
			"models",
			slices.Concat([]cache.Option{
				cache.WithTTL(models.Lifetime),
				cache.WithTimeout(models.Timeout),
				cache.WithAsync(isAsync),
			}, extra)...,
		),
	}, nil
}

// Get returns the models the program on the command line lists.
func (c *Catalog) Get(ctx context.Context, command, env []string) ([]models.Model, error) {
	key := strings.Join(command, "\x00")
	cmdCopy := slices.Clone(command)
	envCopy := slices.Clone(env)
	list, found, err := c.lists.Get(ctx, key, func(ctx context.Context) ([]models.Model, error) {
		return c.lister.ListModels(ctx, cmdCopy, envCopy)
	})
	if err != nil {
		return nil, err
	}
	if !found && c.lists.HasFailed(key) {
		return models.GetFallback(), nil
	}
	return list, nil
}

// List returns the models of the program as the agent service lists them. A program that is not
// installed, or does not answer, lists the fallback models.
func (c *Catalog) List(ctx context.Context, command string, extra, env []string) []*reasoning.Model {
	var list []models.Model
	var err error
	if command == "" {
		err = errors.New("the program is not installed")
	} else {
		list, err = c.Get(ctx, slices.Concat([]string{command}, extra), env)
	}
	if err != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Diagnostic log to os.Stderr does not affect the returned fallback models."
		fmt.Fprintf(os.Stderr, "list models of claude: %v\n", err)
		list = models.GetFallback()
	}
	out := make([]*reasoning.Model, 0, len(list))
	for _, one := range list {
		out = append(out, one.GetReasoningModel())
	}
	return out
}

// GetReasoning returns the depth of reasoning a run of the model is given: the requested one when
// the model supports it, or when the model is not listed and the request is a known depth.
func (c *Catalog) GetReasoning(model, requested string) string {
	if model == "" {
		model = models.DefaultName
	}
	levels, isListed := c.findLevels(model)
	if isListed {
		return reasoning.Pick(levels, requested)
	}
	if reasoning.IsKnown(requested) {
		return requested
	}
	return ""
}

func (c *Catalog) findLevels(model string) ([]reasoning.Level, bool) {
	for _, list := range c.lists.Values() {
		for _, items := range list {
			if items.Value == model {
				return items.GetLevels(), true
			}
		}
	}
	return nil, false
}
