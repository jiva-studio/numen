package completion_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"
)

// A purpose is answered by its own route, and by the default route when it has
// none.
func TestAPurposeIsRoutedToItsOwnRouteThenToTheDefault(t *testing.T) {
	config := completion.Config{Routes: map[string]completion.Route{
		"proofreading":          {Agent: "a", Model: "m1"},
		completion.DefaultRoute: {Agent: "b", Model: "m2"},
	}}
	for _, one := range []struct {
		purpose, name, agent string
	}{
		{"proofreading", "proofreading", "a"},
		{"summary", completion.DefaultRoute, "b"},
		{"", completion.DefaultRoute, "b"},
	} {
		name, route, found := config.GetRoute(one.purpose)
		if !found || name != one.name || route.Agent != one.agent {
			t.Errorf("GetRoute(%q) = %q, %+v, %v; want %q on agent %q",
				one.purpose, name, route, found, one.name, one.agent)
		}
	}
}

// A configuration without the purpose and without a default routes nothing.
func TestNoRouteIsFoundWhereNothingIsConfigured(t *testing.T) {
	for _, config := range []completion.Config{
		{},
		{Routes: map[string]completion.Route{"proofreading": {Agent: "a"}}},
	} {
		if _, _, found := config.GetRoute("summary"); found {
			t.Errorf("%+v routes a purpose it names no route for", config)
		}
	}
}

// An empty purpose never reads a route filed under the empty name.
func TestAnEmptyPurposeIsAnsweredByTheDefaultRoute(t *testing.T) {
	config := completion.Config{Routes: map[string]completion.Route{
		"":                      {Agent: "wrong"},
		completion.DefaultRoute: {Agent: "right"},
	}}
	if _, route, _ := config.GetRoute(""); route.Agent != "right" {
		t.Errorf("an empty purpose was routed to %q", route.Agent)
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
