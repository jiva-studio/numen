// Package catalog reports the models the configured chat API offers: whether it can be reached, and the
// one program the agent runs with those models.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// ProgramName is the identifier a task uses to select the agent, and the key of its settings section.
const ProgramName = "numen"

// programTitle is the agent's display name.
const programTitle = "Numen Agent"

// listingLimit is how long the chat API has to list its models.
const listingLimit = 15 * time.Second

// Catalog lists the models of the chat API the settings name, and keeps the listing until the
// settings that produced it change.
type Catalog struct {
	store  *settings.Store
	lister ports.ModelLister

	mu       sync.Mutex
	models   []conversation.Model
	isListed bool
}

// New returns a catalog that reads the chat API from store and lists its models with lister.
func New(store *settings.Store, lister ports.ModelLister) (*Catalog, error) {
	if store == nil || lister == nil {
		return nil, errors.New("a catalog needs settings and a model lister")
	}
	return &Catalog{store: store, lister: lister, mu: sync.Mutex{}, models: nil, isListed: false}, nil
}

// Reset drops the listing kept, so the next call asks the chat API again.
func (c *Catalog) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.models, c.isListed = nil, false
}

// ListModels returns the models of the chat API, from the listing kept where there is one.
func (c *Catalog) ListModels(ctx context.Context) ([]conversation.Model, error) {
	c.mu.Lock()
	if c.isListed {
		defer c.mu.Unlock()
		return c.models, nil
	}
	current := c.store.Get()
	c.mu.Unlock()

	listing, stop := context.WithTimeout(ctx, listingLimit)
	defer stop()
	models, err := c.lister.ListModels(listing, current)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// A failure triggers a new request, and a listing fetched under settings that have since changed is not kept.
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.store.Get()
	if err == nil && now.BaseURL == current.BaseURL && now.Key == current.Key {
		c.models, c.isListed = models, true
	}
	return models, err
}

// GetState is unreachable while the chat API does not list its models.
func (c *Catalog) GetState(ctx context.Context) *agentrun.State {
	state := &agentrun.State{Unreachable: ""}
	if _, err := c.ListModels(ctx); err != nil {
		state.Unreachable = fmt.Sprintf("the chat API cannot be reached: %v", err)
	}
	return state
}

// GetPrograms lists the one program this agent runs: the chat API the settings name, installed
// while it lists its models, with those models and the configured one as the default. It runs only
// the vault tools a task names.
func (c *Catalog) GetPrograms(ctx context.Context) []*agentrun.Program {
	models, err := c.ListModels(ctx)
	configured := c.store.Get().GetModel("")
	program := &agentrun.Program{
		Name:        ProgramName,
		Title:       programTitle,
		IsInstalled: err == nil,
		LimitsTools: true,
		IsDefault:   false,
		Models:      nil,
	}
	hasConfigured := false
	for _, one := range models {
		if one.Name == configured {
			hasConfigured = true
		}
		program.Models = append(program.Models, &reasoning.Model{
			Name: one.Name, Title: one.Title, IsDefault: one.Name == configured,
			ReasoningLevels: one.ReasoningLevels, DefaultReasoning: one.DefaultReasoning,
		})
	}
	if !hasConfigured && configured != "" && len(models) > 0 {
		program.Models = append(program.Models, &reasoning.Model{
			Name: configured, Title: configured, IsDefault: true,
			ReasoningLevels: nil, DefaultReasoning: "",
		})
	}
	return []*agentrun.Program{program}
}
