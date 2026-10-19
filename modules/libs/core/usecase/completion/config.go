package completion

import "encoding/json"

// DefaultConcurrency is how many questions an agent answers at once when the
// settings name no limit.
const DefaultConcurrency = 1

// Config is the section of the settings file that says which agent answers the
// questions of a plugin whose own setting names none.
type Config struct {
	// Default is the agent and the model a plugin naming no agent is answered
	// by, and how many questions one agent answers at once.
	Default Route `json:"default"`
}

// Route is an agent, the model it answers with, and the limit of its questions.
type Route struct {
	// Agent is the id of the agent plugin that answers. Empty is the provider
	// with the lowest id.
	Agent string `json:"agent,omitempty"`
	// Model is the model the agent answers with. Empty leaves it to the agent.
	Model string `json:"model,omitempty"`
	// Concurrency is how many questions one agent answers at once. Below one is
	// DefaultConcurrency.
	Concurrency int `json:"concurrency,omitempty"`
}

// GetLimit is how many questions one agent answers at once.
func (r Route) GetLimit() int {
	if r.Concurrency < 1 {
		return DefaultConcurrency
	}
	return r.Concurrency
}

// UnmarshalJSON reads the section. A file of an older shape holds its default
// under routes.default, which is read where the section names no default; the
// routes of single purposes are not read.
func (c *Config) UnmarshalJSON(data []byte) error {
	var stored struct {
		Default *Route           `json:"default"`
		Routes  map[string]Route `json:"routes"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	if stored.Default != nil {
		c.Default = *stored.Default
		return nil
	}
	c.Default = stored.Routes["default"]
	return nil
}
