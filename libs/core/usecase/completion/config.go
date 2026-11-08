package completion

// DefaultConcurrency is how many questions an agent answers at once when the
// settings name no limit.
const DefaultConcurrency = 1

// Config is the section of the settings file that selects the agent that
// handles the queries of a plugin whose own setting names none.
type Config struct {
	Default Route
}

// Route is an agent, the model it answers with, and the limit of its questions.
type Route struct {
	Agent       string
	Model       string
	Concurrency int
}

// GetLimit is how many questions one agent answers at once.
func (r Route) GetLimit() int {
	if r.Concurrency < 1 {
		return DefaultConcurrency
	}
	return r.Concurrency
}
