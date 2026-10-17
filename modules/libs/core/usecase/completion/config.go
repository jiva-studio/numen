package completion

// DefaultRoute is the name of the route a purpose without a route of its own
// is answered by.
const DefaultRoute = "default"

// DefaultConcurrency is how many questions a route answers at once when it
// names no limit.
const DefaultConcurrency = 1

// Config is the section of the settings file that says where each purpose is
// answered.
type Config struct {
	// Routes are where each purpose is answered, and default for the rest. A
	// route names an agent, a model and how many questions it answers at once.
	Routes map[string]Route `json:"routes,omitempty"`
}

// Route is where the questions of one purpose go.
type Route struct {
	// Agent is the id of the agent plugin that answers. Empty is the agent the
	// installation provides.
	Agent string `json:"agent,omitempty"`
	// Model is the model the agent answers with. Empty leaves it to the agent.
	Model string `json:"model,omitempty"`
	// Concurrency is how many questions the route answers at once. Below one is
	// DefaultConcurrency.
	Concurrency int `json:"concurrency,omitempty"`
}

// GetLimit is how many questions the route answers at once.
func (r Route) GetLimit() int {
	if r.Concurrency < 1 {
		return DefaultConcurrency
	}
	return r.Concurrency
}

// GetRoute returns the route that answers the purpose, and the name it is
// held under: the purpose's own, else DefaultRoute. It reports false when
// neither is configured.
func (c Config) GetRoute(purpose string) (string, Route, bool) {
	if route, found := c.Routes[purpose]; found && purpose != "" {
		return purpose, route, true
	}
	route, found := c.Routes[DefaultRoute]
	return DefaultRoute, route, found
}
