package agent

import (
	"context"
	"sync"
	"time"
)

// askLimit is how long an agent has to say which models it offers.
const askLimit = 8 * time.Second

// AgentModel is one model an agent says it offers.
type AgentModel struct {
	Name  string
	Title string
	// IsDefault is set where the agent names the model it answers with unasked.
	IsDefault bool
}

// modelSource asks one agent which models it offers. An agent that cannot say
// answers none, and so does one that fails to.
type modelSource interface {
	GetModels(ctx context.Context) ([]AgentModel, error)
}

// session is what the agents reported since this process began.
var session = &catalogue{}

// catalogue keeps what each agent reported, for as long as the way of asking it stays the same.
type catalogue struct {
	mu   sync.Mutex
	kept map[string]keptModels
}

type keptModels struct {
	key    string
	models []AgentModel
}

// get are the models the agent offers: the ones kept, or those it reports now. A failure is kept as none.
func (c *catalogue) get(use, key string, source modelSource) []AgentModel {
	c.mu.Lock()
	known, isKnown := c.kept[use]
	c.mu.Unlock()
	if isKnown && known.key == key {
		return known.models
	}

	ctx, stop := context.WithTimeout(context.Background(), askLimit)
	defer stop()
	models, err := source.GetModels(ctx)
	if err != nil {
		models = nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kept == nil {
		c.kept = map[string]keptModels{}
	}
	c.kept[use] = keptModels{key: key, models: models}
	return models
}

// clear drops everything kept.
func (c *catalogue) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.kept = nil
}
