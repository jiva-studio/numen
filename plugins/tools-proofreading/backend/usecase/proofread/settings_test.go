package proofread

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

func TestDefaultConfigAndBounds(t *testing.T) {
	def := DefaultConfig()
	if def.BatchSize != 40 || def.Overlap != 4 || def.InFlight != 4 || def.MaxEditDistance != numbered.MaxEditDistance {
		t.Fatalf("DefaultConfig() = %+v", def)
	}

	atLeast := ParseConfig(map[string]string{
		"batchSize": "1", "overlap": "0", "inFlight": "1", "maxEditDistance": "0",
	}, def)
	if atLeast.BatchSize != 1 || atLeast.Overlap != 0 || atLeast.InFlight != 1 || atLeast.MaxEditDistance != 0 {
		t.Errorf("at least bounds = %+v", atLeast)
	}

	belowLeast := ParseConfig(map[string]string{
		"batchSize": "0", "overlap": "-1", "inFlight": "0", "maxEditDistance": "-0.1",
	}, def)
	if belowLeast != def {
		t.Errorf("below least bounds = %+v, want %+v", belowLeast, def)
	}

	invalid := ParseConfig(map[string]string{
		"batchSize": "bad", "overlap": "bad", "inFlight": "bad", "maxEditDistance": "bad",
	}, def)
	if invalid != def {
		t.Errorf("invalid values = %+v, want %+v", invalid, def)
	}
}

func TestRequestSizesAreReadFromTheCamelCaseKeys(t *testing.T) {
	got := ParseConfig(map[string]string{
		"batchSize": "7", "overlap": "2", "inFlight": "3", "maxEditDistance": "0.45",
	}, DefaultConfig())

	if got.BatchSize != 7 || got.Overlap != 2 || got.InFlight != 3 || got.MaxEditDistance != 0.45 {
		t.Errorf("batch size %d, overlap %d, in flight %d, max edit distance %v; want 7, 2, 3 and 0.45",
			got.BatchSize, got.Overlap, got.InFlight, got.MaxEditDistance)
	}
}

func TestRequestSizesAreNotReadFromTheSnakeCaseKeys(t *testing.T) {
	before := DefaultConfig()

	got := ParseConfig(map[string]string{
		"batch_size": "7", "in_flight": "3", "max_edit_distance": "0.45",
	}, before)

	if got != before {
		t.Errorf("config = %+v, want it unchanged at %+v", got, before)
	}
}
