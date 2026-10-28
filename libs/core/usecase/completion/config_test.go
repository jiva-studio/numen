package completion_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/completion"
)

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
