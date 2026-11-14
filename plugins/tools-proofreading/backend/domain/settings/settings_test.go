package settings

import "testing"

func TestRequestSizesAreReadFromTheCamelCaseKeys(t *testing.T) {
	got := ParseConfig(map[string]string{
		"batchSize": "7", "inFlight": "3", "maxEditDistance": "0.45",
	}, DefaultConfig())

	if got.BatchSize != 7 || got.InFlight != 3 || got.MaxEditDistance != 0.45 {
		t.Errorf("batch size %d, in flight %d, max edit distance %v; want 7, 3 and 0.45",
			got.BatchSize, got.InFlight, got.MaxEditDistance)
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
