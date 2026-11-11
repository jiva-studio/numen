// Package catalog keeps the list of models the program accepts and answers which depths of
// reasoning a model supports.
package catalog

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
)

// Catalog lists models through a cache, one list per command line of the program.
type Catalog struct {
	lister ports.ModelLister
	lists  *cache.Cache[[]model.Model]
	report func(error)
}

// New creates a catalog that asks lister for the models. An async catalog answers with no model
// while the first list loads, and the fallback models once that load has failed.
func New(lister ports.ModelLister, isAsync bool, extra ...cache.Option) (*Catalog, error) {
	if lister == nil {
		return nil, errors.New("catalog: the model lister is required")
	}
	return &Catalog{
		lister: lister,
		lists: cache.New[[]model.Model](
			"models",
			slices.Concat([]cache.Option{
				cache.WithTTL(model.Lifetime),
				cache.WithTimeout(model.Timeout),
				cache.WithAsync(isAsync),
			}, extra)...,
		),
		report: nil,
	}, nil
}

// SetReport sets the callback that receives listing errors when List falls back.
func (c *Catalog) SetReport(report func(error)) {
	c.report = report
}

// Get returns the models the program on the command line lists.
func (c *Catalog) Get(ctx context.Context, command, env []string) ([]model.Model, error) {
	key := strings.Join(command, "\x00")
	cmdCopy := slices.Clone(command)
	envCopy := slices.Clone(env)
	list, found, err := c.lists.Get(ctx, key, func(ctx context.Context) ([]model.Model, error) {
		return c.lister.ListModels(ctx, cmdCopy, envCopy)
	})
	if err != nil {
		return nil, err
	}
	if !found && c.lists.HasFailed(key) {
		return model.GetFallback(), nil
	}
	return list, nil
}

// List returns the models of the program as the agent service lists them. A program that is not
// installed, or does not answer, lists the fallback models.
func (c *Catalog) List(ctx context.Context, command string, extra, env []string) []*reasoning.Model {
	var list []model.Model
	var err error
	if command == "" {
		err = errors.New("the program is not installed")
	} else {
		list, err = c.Get(ctx, slices.Concat([]string{command}, extra), env)
	}
	if err != nil {
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="Diagnostic error reporting does not affect the returned fallback models."
		if c.report != nil {
			c.report(err)
		}
		list = model.GetFallback()
	}
	out := make([]*reasoning.Model, 0, len(list))
	for _, one := range list {
		out = append(out, one.GetReasoningModel())
	}
	return out
}

// GetReasoning returns the depth of reasoning a run of the model is given: the requested one when
// the model supports it, or when the model is not listed and the request is a known depth.
func (c *Catalog) GetReasoning(modelName, requested string) string {
	if modelName == "" {
		modelName = model.DefaultName
	}
	levels, isListed := c.findLevels(modelName)
	if isListed {
		return reasoning.Pick(levels, requested)
	}
	if reasoning.IsKnown(requested) {
		return requested
	}
	return ""
}

func (c *Catalog) findLevels(modelName string) ([]reasoning.Level, bool) {
	for _, list := range c.lists.Values() {
		for _, items := range list {
			if items.Value == modelName {
				return items.GetLevels(), true
			}
		}
	}
	return nil, false
}
