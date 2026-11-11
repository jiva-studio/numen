package container

import (
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func makePlan(workers int) settingsdomain.ResolvedCompute {
	return settingsdomain.ResolvedCompute{Workers: workers}
}

func TestASearchExtractsAsManySourcesAtOnceAsThePlanHasWorkers(t *testing.T) {
	cfg := NewConfig(Config{})
	cfg.Resources.machine = Machine{ResolvedCompute: makePlan(5)}
	got := newSearchOver(searchOf(cfg), search.Index{}, nil, nil).Extractors
	if got != 5 {
		t.Errorf("a search extracts %d sources at once, want the 5 planned", got)
	}
}

func TestNewSearchOver_UsesDefaultExtractorCountWithoutPlan(t *testing.T) {
	want := search.New(search.Index{}, nil, nil, nil, nil, nil).Extractors
	if got := newSearchOver(searchOf((NewConfig(Config{}))), search.Index{}, nil, nil).Extractors; got != want {
		t.Errorf("a search extracts %d sources at once, want %d", got, want)
	}
}
