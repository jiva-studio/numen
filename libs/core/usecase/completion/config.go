package completion

// DefaultConcurrency is how many questions an agent answers at once when the
// settings name no limit.
const DefaultConcurrency = 1

// Config is the section of the settings file that says which agent answers the
// questions of a plugin whose own setting names none.
type Config struct {
	// Default is the agent and the model a plugin naming no agent is answered
	// by, and how many questions one agent answers at once.
	Default Route
}

// Route is an agent, the model it answers with, and the limit of its questions.
type Route struct {
	// Agent is the id of the agent plugin that answers. Empty is the provider
	// with the lowest id.
	Agent string
	// Model is the model the agent answers with. Empty leaves it to the agent.
	Model string
	// Concurrency is how many questions one agent answers at once. Below one is
	// DefaultConcurrency.
	Concurrency int
}

// GetLimit is how many questions one agent answers at once.
func (r Route) GetLimit() int {
	if r.Concurrency < 1 {
		return DefaultConcurrency
	}
	return r.Concurrency
}
