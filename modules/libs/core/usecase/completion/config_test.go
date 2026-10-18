package completion_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"
)

// The default stands under ai.default, with the agent, the model and the limit.
func TestTheDefaultIsReadFromItsOwnKey(t *testing.T) {
	var config completion.Config
	if err := json.Unmarshal([]byte(`{"default": {"agent": "a", "model": "m", "concurrency": 3}}`), &config); err != nil {
		t.Fatal(err)
	}
	if want := (completion.Route{Agent: "a", Model: "m", Concurrency: 3}); config.Default != want {
		t.Errorf("read %+v, want %+v", config.Default, want)
	}
}

// A file of the older shape keeps its default route, and the routes of single
// purposes are not read.
func TestAnOlderFileKeepsItsDefaultRoute(t *testing.T) {
	var config completion.Config
	older := `{"routes": {"default": {"agent": "a", "model": "m", "concurrency": 2}, "proofreading": {"agent": "b"}}}`
	if err := json.Unmarshal([]byte(older), &config); err != nil {
		t.Fatal(err)
	}
	if want := (completion.Route{Agent: "a", Model: "m", Concurrency: 2}); config.Default != want {
		t.Errorf("read %+v, want %+v", config.Default, want)
	}

	if err := json.Unmarshal([]byte(`{"routes": {"proofreading": {"agent": "b"}}}`), &config); err != nil {
		t.Fatal(err)
	}
	if config.Default != (completion.Route{}) {
		t.Errorf("a purpose's route was read as the default: %+v", config.Default)
	}
}

// The default of the new shape stands over the routes of the older one.
func TestTheDefaultStandsOverTheOlderRoutes(t *testing.T) {
	var config completion.Config
	both := `{"default": {"agent": "new"}, "routes": {"default": {"agent": "old"}}}`
	if err := json.Unmarshal([]byte(both), &config); err != nil {
		t.Fatal(err)
	}
	if config.Default.Agent != "new" {
		t.Errorf("read agent %q, want new", config.Default.Agent)
	}
}

// A limit below one is the default one.
func TestARouteWithoutALimitAnswersOneQuestionAtATime(t *testing.T) {
	for _, limit := range []int{0, -3} {
		if got := (completion.Route{Concurrency: limit}).GetLimit(); got != completion.DefaultConcurrency {
			t.Errorf("a concurrency of %d gives a limit of %d", limit, got)
		}
	}
	if got := (completion.Route{Concurrency: 4}).GetLimit(); got != 4 {
		t.Errorf("a concurrency of 4 gives a limit of %d", got)
	}
}
