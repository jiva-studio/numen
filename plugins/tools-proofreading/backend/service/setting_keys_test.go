package service

import "testing"

func TestRequestSizesAreReadFromTheCamelCaseKeys(t *testing.T) {
	s := newSettings()
	s.updateConfig(map[string]string{
		"batchSize": "7", "inFlight": "3", "maxEditDistance": "0.45",
	})

	got := s.getConfig()
	if got.batchSize != 7 || got.inFlight != 3 || got.maxEditDistance != 0.45 {
		t.Errorf("batch size %d, in flight %d, max edit distance %v; want 7, 3 and 0.45",
			got.batchSize, got.inFlight, got.maxEditDistance)
	}
}

func TestRequestSizesAreNotReadFromTheSnakeCaseKeys(t *testing.T) {
	s := newSettings()
	before := s.getConfig()
	s.updateConfig(map[string]string{
		"batch_size": "7", "in_flight": "3", "max_edit_distance": "0.45",
	})

	if got := s.getConfig(); got != before {
		t.Errorf("config = %+v, want it unchanged at %+v", got, before)
	}
}
